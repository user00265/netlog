// Package sync implements the server side of the offline-first protocol: it
// applies a batch of client changes (the outbox) authoritatively and serves
// changed records for syncing down. The server owns conflict resolution and the
// authoritative UTC timestamps for net start/stop.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"netlog/internal/models"
	"netlog/internal/store"
	"netlog/internal/validate"
)

// Entity / op / status constants.
const (
	EntityNet     = "net"
	EntityCheckIn = "checkin"

	OpPut    = "put"
	OpDelete = "delete"

	StatusApplied  = "applied"
	StatusConflict = "conflict"
	StatusError    = "error"
)

// Field length caps for untrusted free text. Generous for real use; they exist
// to bound storage and rendering, not to constrain normal logging.
const (
	maxNetName      = 200
	maxNetNotes     = 20_000
	maxNickname     = 80
	maxCheckinNotes = 2_000
)

// Enricher triggers asynchronous callbook/DXCC enrichment for a callsign.
type Enricher interface {
	Lookup(ctx context.Context, callsign string, forceRefresh bool) (models.CallsignData, error)
}

// Notifier publishes events to connected SSE clients. It may be nil.
type Notifier interface {
	Broadcast(event, data string)
}

// Change is one outbox entry from the client.
type Change struct {
	Entity string          `json:"entity"`
	Op     string          `json:"op"`
	ID     string          `json:"id"`
	Data   json.RawMessage `json:"data"`
}

// Result reports the outcome of applying a single change.
type Result struct {
	ID      string          `json:"id"`
	Entity  string          `json:"entity"`
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Net     *models.Net     `json:"net,omitempty"`
	CheckIn *models.CheckIn `json:"checkin,omitempty"`
}

// PushRequest is the body of POST /api/sync.
type PushRequest struct {
	Changes []Change `json:"changes"`
}

// PushResponse returns per-change results and the server clock.
type PushResponse struct {
	ServerTime string   `json:"serverTime"`
	Results    []Result `json:"results"`
}

// PullResponse returns records changed since a timestamp (tombstones included).
type PullResponse struct {
	ServerTime string           `json:"serverTime"`
	Nets       []models.Net     `json:"nets"`
	CheckIns   []models.CheckIn `json:"checkins"`
}

// Service applies and serves sync changes.
type Service struct {
	store    *store.Store
	enricher Enricher
	notifier Notifier
	logger   *slog.Logger
}

// NewService builds a sync Service. enricher and notifier may be nil.
func NewService(s *store.Store, enricher Enricher, notifier Notifier, logger *slog.Logger) *Service {
	return &Service{store: s, enricher: enricher, notifier: notifier, logger: logger}
}

// notify publishes an SSE event when a notifier is configured.
func (s *Service) notify(event, data string) {
	if s.notifier != nil {
		s.notifier.Broadcast(event, data)
	}
}

// Pull returns all records changed strictly after since.
func (s *Service) Pull(ctx context.Context, since string) (PullResponse, error) {
	nets, err := s.store.ChangedNetsSince(ctx, since)
	if err != nil {
		return PullResponse{}, err
	}
	checkins, err := s.store.ChangedCheckinsSince(ctx, since)
	if err != nil {
		return PullResponse{}, err
	}
	// Return empty arrays rather than nil so clients can always iterate.
	if nets == nil {
		nets = []models.Net{}
	}
	if checkins == nil {
		checkins = []models.CheckIn{}
	}
	return PullResponse{ServerTime: models.Now(), Nets: nets, CheckIns: checkins}, nil
}

// Push applies a batch of changes for the authenticated user. If any change was
// applied, connected clients are told to sync down (so other devices/sessions
// pick the change up).
func (s *Service) Push(ctx context.Context, user models.User, req PushRequest) PushResponse {
	resp := PushResponse{ServerTime: models.Now()}
	applied := false
	for _, ch := range req.Changes {
		r := s.apply(ctx, user, ch)
		if r.Status == StatusApplied {
			applied = true
		}
		resp.Results = append(resp.Results, r)
	}
	if applied {
		s.notify("sync", "1")
	}
	return resp
}

func (s *Service) apply(ctx context.Context, user models.User, ch Change) Result {
	switch ch.Entity {
	case EntityNet:
		return s.applyNet(ctx, user, ch)
	case EntityCheckIn:
		return s.applyCheckin(ctx, user, ch)
	default:
		return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusError, Message: "unknown entity"}
	}
}

func conflict(ch Change, msg string) Result {
	return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusConflict, Message: msg}
}

func failed(ch Change, msg string) Result {
	return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusError, Message: msg}
}

// canManageNet reports whether the user may mutate the net (its NCS or an admin).
func canManageNet(user models.User, n models.Net) bool {
	return user.IsAdmin() || user.ID == n.NCSUserID
}

// netPatch is a partial net update. Pointer fields distinguish "absent" from a
// zero value, so a status-only change doesn't clobber name/notes and vice versa.
type netPatch struct {
	Name    *string `json:"name"`
	NetDate *string `json:"netDate"`
	Status  *string `json:"status"`
	Notes   *string `json:"notes"`
}

func (s *Service) applyNet(ctx context.Context, user models.User, ch Change) Result {
	var p netPatch
	if err := json.Unmarshal(ch.Data, &p); err != nil {
		return failed(ch, "invalid net payload")
	}
	if !validate.ValidID(ch.ID) {
		return failed(ch, "invalid net id")
	}
	now := models.Now()

	existing, err := s.store.GetNet(ctx, ch.ID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		if ch.Op == OpDelete {
			return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied} // already gone
		}
		return s.createNet(ctx, user, ch, p, now)
	case err != nil:
		return failed(ch, "lookup net")
	}

	if !canManageNet(user, existing) {
		return conflict(ch, "not authorized for this net")
	}
	if existing.Status == models.NetClosed && ch.Op != OpDelete {
		return conflict(ch, "net is closed")
	}

	if ch.Op == OpDelete {
		existing.DeletedAt = &now
		existing.UpdatedAt = now
		if err := s.store.UpdateNet(ctx, existing); err != nil {
			return failed(ch, "delete net")
		}
		return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied, Net: &existing}
	}

	// Field edits on a still-editable net. Pointers mean only supplied fields
	// change (notes is auto-saved separately from status/name). All free text is
	// sanitized and length-capped before storage.
	if p.Name != nil {
		if name := validate.CleanLine(*p.Name, maxNetName); name != "" {
			existing.Name = name
		}
	}
	if p.NetDate != nil && *p.NetDate != "" {
		if !validate.ValidDate(*p.NetDate) {
			return failed(ch, "invalid net date")
		}
		existing.NetDate = *p.NetDate
	}
	if p.Notes != nil {
		existing.Notes = validate.CleanMultiline(*p.Notes, maxNetNotes)
	}
	if msg := applyNetTransition(&existing, derefStatus(p.Status), now); msg != "" {
		return conflict(ch, msg)
	}
	existing.UpdatedAt = now
	if err := s.store.UpdateNet(ctx, existing); err != nil {
		return failed(ch, "update net")
	}
	return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied, Net: &existing}
}

func derefStatus(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Service) createNet(ctx context.Context, user models.User, ch Change, p netPatch, now string) Result {
	name, netDate := "", ""
	if p.Name != nil {
		name = validate.CleanLine(*p.Name, maxNetName)
	}
	if p.NetDate != nil {
		netDate = strings.TrimSpace(*p.NetDate)
	}
	if name == "" || netDate == "" {
		return failed(ch, "net requires name and date")
	}
	if !validate.ValidDate(netDate) {
		return failed(ch, "invalid net date")
	}
	n := models.Net{
		ID:        ch.ID,
		Name:      name,
		NetDate:   netDate,
		NCSUserID: user.ID,
		Status:    models.NetPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if p.Notes != nil {
		n.Notes = validate.CleanMultiline(*p.Notes, maxNetNotes)
	}
	// Allow opening as part of creation.
	if msg := applyNetTransition(&n, derefStatus(p.Status), now); msg != "" {
		return conflict(ch, msg)
	}
	if err := s.store.CreateNet(ctx, n); err != nil {
		return failed(ch, "create net")
	}
	return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied, Net: &n}
}

// applyNetTransition enforces the net lifecycle. The server stamps the
// authoritative UTC start/end times; once set they are immutable. Returns a
// non-empty message on an illegal transition.
func applyNetTransition(n *models.Net, desired, now string) string {
	if desired == "" || desired == n.Status {
		return ""
	}
	switch {
	case n.Status == models.NetPending && desired == models.NetOpen:
		n.Status = models.NetOpen
		if n.StartAt == nil {
			n.StartAt = &now
		}
	case n.Status == models.NetOpen && desired == models.NetClosed:
		n.Status = models.NetClosed
		if n.EndAt == nil {
			n.EndAt = &now
		}
	case n.Status == models.NetPending && desired == models.NetClosed:
		// Cancel a net that never opened.
		n.Status = models.NetClosed
		if n.EndAt == nil {
			n.EndAt = &now
		}
	default:
		return "illegal net status transition"
	}
	return ""
}

func (s *Service) applyCheckin(ctx context.Context, user models.User, ch Change) Result {
	var in models.CheckIn
	if err := json.Unmarshal(ch.Data, &in); err != nil {
		return failed(ch, "invalid checkin payload")
	}
	if !validate.ValidID(ch.ID) {
		return failed(ch, "invalid checkin id")
	}
	in.ID = ch.ID
	now := models.Now()

	existing, err := s.store.GetCheckin(ctx, in.ID)
	isNew := errors.Is(err, store.ErrNotFound)
	if err != nil && !isNew {
		return failed(ch, "lookup checkin")
	}

	netID := in.NetID
	if !isNew {
		netID = existing.NetID
	}
	net, err := s.store.GetNet(ctx, netID)
	if errors.Is(err, store.ErrNotFound) {
		return conflict(ch, "net not found")
	} else if err != nil {
		return failed(ch, "lookup net")
	}
	if !canManageNet(user, net) {
		return conflict(ch, "not authorized for this net")
	}

	// Deletes (tombstones) are allowed regardless of net status: an offline NCS
	// may have deleted a check-in while the net was open and only synced after it
	// closed. Blocking it would leave the tombstone stuck in the outbox forever
	// and the row would resurrect on the next pull.
	if ch.Op == OpDelete {
		if isNew {
			return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied}
		}
		existing.DeletedAt = &now
		existing.UpdatedAt = now
		if err := s.store.UpdateCheckin(ctx, existing); err != nil {
			return failed(ch, "delete checkin")
		}
		return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied, CheckIn: &existing}
	}

	// Creates and edits require an open net.
	if net.Status != models.NetOpen {
		return conflict(ch, "net is not open")
	}

	callsign := validate.NormalizeCallsign(in.Callsign)
	if !validate.ValidCallsign(callsign) {
		return failed(ch, "invalid callsign")
	}
	nickname := validate.CleanLine(in.Nickname, maxNickname)
	notes := validate.CleanMultiline(in.Notes, maxCheckinNotes)

	var result models.CheckIn
	if isNew {
		seq := in.Seq
		if seq <= 0 {
			if seq, err = s.store.NextCheckinSeq(ctx, net.ID); err != nil {
				return failed(ch, "assign sequence")
			}
		}
		checkedAt := in.CheckedInAt
		if _, perr := models.ParseTime(checkedAt); perr != nil {
			// Don't trust a client-supplied timestamp we can't parse.
			checkedAt = now
		}
		result = models.CheckIn{
			ID:          in.ID,
			NetID:       net.ID,
			Callsign:    callsign,
			Nickname:    nickname,
			HasTraffic:  in.HasTraffic,
			ShortTime:   in.ShortTime,
			Notes:       notes,
			Seq:         seq,
			CheckedInAt: checkedAt,
			CreatedBy:   &user.ID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.store.CreateCheckin(ctx, result); err != nil {
			return failed(ch, "create checkin")
		}
	} else {
		existing.Callsign = callsign
		existing.Nickname = nickname
		existing.HasTraffic = in.HasTraffic
		existing.ShortTime = in.ShortTime
		existing.Notes = notes
		if in.Seq > 0 {
			existing.Seq = in.Seq
		}
		if _, perr := models.ParseTime(in.CheckedInAt); perr == nil {
			existing.CheckedInAt = in.CheckedInAt
		}
		existing.UpdatedAt = now
		if err := s.store.UpdateCheckin(ctx, existing); err != nil {
			return failed(ch, "update checkin")
		}
		result = existing
	}

	s.enrich(callsign)
	return Result{ID: ch.ID, Entity: ch.Entity, Status: StatusApplied, CheckIn: &result}
}

// enrich fires a background callbook/DXCC lookup so the check-in's flag and
// callbook details populate without blocking the sync response.
func (s *Service) enrich(callsign string) {
	if s.enricher == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := s.enricher.Lookup(ctx, callsign, false); err != nil {
			s.logger.DebugContext(ctx, "enrichment failed", slog.String("callsign", callsign), slog.String("error", err.Error()))
			return
		}
		// Tell connected clients the callbook data for this callsign is ready.
		s.notify("callsign", callsign)
	}()
}
