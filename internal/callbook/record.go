package callbook

import (
	"strings"

	"netlog/internal/models"
	"netlog/internal/validate"
)

// maxRawBody caps how much of a provider's raw payload we persist. Real callbook
// records are a few KB; this bounds storage against a hostile/oversized response
// (the HTTP client already caps the read at 4 MiB).
const maxRawBody = 64 * 1024

// sanitize cleans and length-caps every field that came from the external
// callbook before it is stored or rendered. Per the project's distrust of all
// external data, nothing from QRZ/HamQTH is persisted verbatim except a bounded
// copy of the raw payload (kept for forensics, never re-parsed as trusted).
func (r *Record) sanitize() {
	r.Callsign = validate.NormalizeCallsign(r.Callsign)
	r.FirstName = validate.CleanLine(r.FirstName, 120)
	r.LastName = validate.CleanLine(r.LastName, 120)
	r.Nickname = validate.CleanLine(r.Nickname, 120)
	r.Address1 = validate.CleanLine(r.Address1, 200)
	r.Address2 = validate.CleanLine(r.Address2, 200)
	r.City = validate.CleanLine(r.City, 120)
	r.State = validate.CleanLine(r.State, 120)
	r.Zip = validate.CleanLine(r.Zip, 32)
	r.Country = validate.CleanLine(r.Country, 120)
	r.Grid = validate.CleanLine(r.Grid, 12)
	r.IOTA = validate.CleanLine(r.IOTA, 16)
	r.Continent = validate.CleanLine(r.Continent, 4)
	r.QSLManager = validate.CleanLine(r.QSLManager, 120)
	r.LoTW = validate.CleanLine(r.LoTW, 8)
	r.EQSL = validate.CleanLine(r.EQSL, 8)
	r.Website = validate.SafeURL(r.Website)
	if email := strings.TrimSpace(r.Email); validate.Email(email) {
		r.Email = validate.Truncate(email, 254)
	} else {
		r.Email = ""
	}
	r.RawBody = validate.Truncate(r.RawBody, maxRawBody)
}

// Provider keys.
const (
	ProviderQRZ    = "qrz"
	ProviderHamQTH = "hamqth"
)

// Record is the normalized result of a callbook lookup. Raw holds every field
// the provider returned (as strings) so nothing is lost even if we don't model
// it; it is persisted as the provider's raw JSON payload.
type Record struct {
	Source     string
	Callsign   string
	FirstName  string
	LastName   string
	Nickname   string
	Address1   string
	Address2   string
	City       string
	State      string
	Zip        string
	Country    string
	DXCC       *int
	Grid       string
	Latitude   *float64
	Longitude  *float64
	CQZone     *int
	ITUZone    *int
	IOTA       string
	Continent  string
	Email      string
	Website    string
	QSLManager string
	LoTW       string
	EQSL       string
	// RawBody is the provider's original response payload, preserved verbatim so
	// no returned data is ever lost.
	RawBody string
}

// RawPtr returns the raw payload for storage, or nil when empty.
func (r *Record) RawPtr() *string {
	if r.RawBody == "" {
		return nil
	}
	return &r.RawBody
}

// toCallsignData projects the record onto the persisted model. DXCC/flag fields
// are filled separately by the resolver from the cty.xml dataset.
func (r *Record) toCallsignData() models.CallsignData {
	return models.CallsignData{
		Callsign:   r.Callsign,
		FirstName:  r.FirstName,
		LastName:   r.LastName,
		Nickname:   r.Nickname,
		Address1:   r.Address1,
		Address2:   r.Address2,
		City:       r.City,
		State:      r.State,
		Zip:        r.Zip,
		Country:    r.Country,
		DXCC:       r.DXCC,
		Grid:       r.Grid,
		Latitude:   r.Latitude,
		Longitude:  r.Longitude,
		CQZone:     r.CQZone,
		ITUZone:    r.ITUZone,
		IOTA:       r.IOTA,
		Continent:  r.Continent,
		Email:      r.Email,
		Website:    r.Website,
		QSLManager: r.QSLManager,
		LoTW:       r.LoTW,
		EQSL:       r.EQSL,
		Source:     r.Source,
	}
}
