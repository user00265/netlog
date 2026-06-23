package callbook

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"netlog/internal/config"
	"netlog/internal/dxcc"
	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/validate"
)

// refreshCooldown caps how often a forced refresh actually contacts the external
// callbooks for a given callsign, protecting the operator's API quota from being
// exhausted by rapid refresh requests.
const refreshCooldown = 30 * time.Second

// dxccLookup is the subset of the DXCC manager the resolver needs (kept small for
// testability).
type dxccLookup interface {
	Lookup(callsign string) (dxcc.Result, bool)
}

// Resolver looks up callsigns across the configured providers (primary first,
// then fallbacks), enriches the result with DXCC/flag data, and caches it.
type Resolver struct {
	providers []Provider
	store     *store.Store
	dxcc      dxccLookup
	logger    *slog.Logger
}

// NewResolver builds a Resolver. Only providers with credentials configured (in
// the configured order) are used.
func NewResolver(cfg config.Callbook, st *store.Store, dx dxccLookup, logger *slog.Logger, client *http.Client) *Resolver {
	hc := newHTTPClient(client, logger)
	var providers []Provider
	for _, name := range cfg.Order {
		if !cfg.Enabled(name) {
			continue
		}
		switch name {
		case ProviderQRZ:
			providers = append(providers, NewQRZ(hc, cfg.QRZ.Username, cfg.QRZ.Password, cfg.QRZ.Agent))
		case ProviderHamQTH:
			providers = append(providers, NewHamQTH(hc, cfg.HamQTH.Username, cfg.HamQTH.Password, cfg.HamQTH.Program))
		}
	}
	return &Resolver{providers: providers, store: st, dxcc: dx, logger: logger}
}

// Cached returns cached data for a callsign without contacting any provider.
func (r *Resolver) Cached(ctx context.Context, callsign string) (models.CallsignData, error) {
	return r.store.GetCallsign(ctx, validate.NormalizeCallsign(callsign))
}

// Lookup resolves a callsign. Unless forceRefresh is set, a cached record is
// returned if present. Otherwise providers are tried in order; whichever answers
// first wins, the result is enriched with DXCC/flag data, cached, and returned.
// Even when no callbook answers, a DXCC match alone yields a (flag-only) record.
func (r *Resolver) Lookup(ctx context.Context, callsign string, forceRefresh bool) (models.CallsignData, error) {
	norm := validate.NormalizeCallsign(callsign)
	if !validate.ValidCallsign(norm) {
		return models.CallsignData{}, errors.New("invalid callsign")
	}

	if cached, err := r.store.GetCallsign(ctx, norm); err == nil {
		// Serve the cache unless a refresh is requested and the cooldown elapsed.
		if !forceRefresh || withinCooldown(cached.LastLookupAt) {
			return cached, nil
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return models.CallsignData{}, err
	}

	rec := r.queryProviders(ctx, norm)
	if rec != nil {
		rec.sanitize()
	}

	var dxccResult dxcc.Result
	dxccOK := false
	if r.dxcc != nil {
		dxccResult, dxccOK = r.dxcc.Lookup(norm)
	}

	if rec == nil && !dxccOK {
		return models.CallsignData{}, ErrCallsignNotFound
	}

	var data models.CallsignData
	if rec != nil {
		data = rec.toCallsignData()
	} else {
		data = models.CallsignData{Callsign: norm}
	}
	// The cache key must be the callsign we queried, never one echoed back by the
	// external API (which we don't trust to be consistent or well-formed).
	data.Callsign = norm
	enrichWithDXCC(&data, dxccResult, dxccOK)

	now := models.Now()
	data.LastLookupAt = &now

	var rawQRZ, rawHamQTH *string
	if rec != nil {
		switch rec.Source {
		case ProviderQRZ:
			rawQRZ = rec.RawPtr()
		case ProviderHamQTH:
			rawHamQTH = rec.RawPtr()
		}
	}
	if err := r.store.UpsertCallsign(ctx, data, rawQRZ, rawHamQTH); err != nil {
		return models.CallsignData{}, err
	}
	return data, nil
}

// queryProviders tries each provider in order, returning the first record. A
// not-found result moves on to the next provider; other errors are logged and
// also fall through so a flaky primary doesn't block the fallback.
func (r *Resolver) queryProviders(ctx context.Context, callsign string) *Record {
	for _, p := range r.providers {
		rec, err := p.Lookup(ctx, callsign)
		switch {
		case err == nil && rec != nil:
			return rec
		case errors.Is(err, ErrCallsignNotFound):
			continue
		case err != nil:
			r.logger.WarnContext(ctx, "callbook provider error",
				slog.String("provider", p.Name()),
				slog.String("callsign", callsign),
				slog.String("error", err.Error()))
		}
	}
	return nil
}

// withinCooldown reports whether a cached entry was looked up recently enough
// that a forced refresh should be suppressed.
func withinCooldown(lastLookupAt *string) bool {
	if lastLookupAt == nil {
		return false
	}
	t, err := models.ParseTime(*lastLookupAt)
	if err != nil {
		return false
	}
	return time.Since(t) < refreshCooldown
}

// enrichWithDXCC fills DXCC-derived fields (flag, country, zones) that the
// callbook did not supply.
func enrichWithDXCC(data *models.CallsignData, res dxcc.Result, ok bool) {
	if !ok {
		return
	}
	data.FlagISO2 = res.FlagISO2
	if data.DXCC == nil {
		adif := res.ADIF
		data.DXCC = &adif
	}
	if data.Continent == "" {
		data.Continent = res.Continent
	}
	if data.Country == "" {
		data.Country = res.Name
	}
	if data.CQZone == nil && res.CQZone != 0 {
		cq := res.CQZone
		data.CQZone = &cq
	}
	if data.ITUZone == nil && res.ITUZone != 0 {
		itu := res.ITUZone
		data.ITUZone = &itu
	}
}
