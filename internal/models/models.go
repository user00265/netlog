// Package models holds NetLog's domain types shared across packages.
package models

// Role values for RBAC.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Net status values.
const (
	NetPending = "pending" // created, not yet started
	NetOpen    = "open"    // live; check-ins editable
	NetClosed  = "closed"  // read-only
)

// User is an account. PasswordHash is empty for OIDC-only accounts.
type User struct {
	ID           string `json:"id"`
	Callsign     string `json:"callsign"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	// Timezone is the operator's IANA timezone for local-time display; empty
	// means the client should fall back to the browser's timezone.
	Timezone string `json:"timezone"`
	// TimeFormat is "24h" or "12h".
	TimeFormat string `json:"timeFormat"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// IsAdmin reports whether the user has the admin role.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// Session is a server-side session; the cookie carries only its ID.
type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

// OIDCIdentity links an external OIDC subject to a local user.
type OIDCIdentity struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Issuer    string `json:"issuer"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"createdAt"`
}

// Net is a directed net. start_at/end_at are immutable once set.
type Net struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	NetDate   string  `json:"netDate"`
	NCSUserID string  `json:"ncsUserId"`
	Status    string  `json:"status"`
	StartAt   *string `json:"startAt"`
	EndAt     *string `json:"endAt"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	DeletedAt *string `json:"deletedAt,omitempty"`
	// NCSCallsign is a denormalized, non-persisted convenience field populated on
	// reads/sync so clients can display the NCS without a separate user lookup.
	NCSCallsign string `json:"ncsCallsign,omitempty"`
}

// NetWithMeta is a net plus the count of (non-deleted) check-ins for list views.
type NetWithMeta struct {
	Net
	CheckInCount int `json:"checkInCount"`
}

// CheckIn is a single check-in to a net.
type CheckIn struct {
	ID          string  `json:"id"`
	NetID       string  `json:"netId"`
	Callsign    string  `json:"callsign"`
	Nickname    string  `json:"nickname"`
	HasTraffic  bool    `json:"hasTraffic"`
	ShortTime   bool    `json:"shortTime"`
	Notes       string  `json:"notes"`
	Seq         int     `json:"seq"`
	CheckedInAt string  `json:"checkedInAt"`
	CreatedBy   *string `json:"createdBy,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

// CallsignData is the cached callbook + DXCC information for a callsign. Raw
// provider payloads are preserved so no looked-up data is ever discarded.
type CallsignData struct {
	Callsign     string   `json:"callsign"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Nickname     string   `json:"nickname"`
	Address1     string   `json:"address1"`
	Address2     string   `json:"address2"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	Zip          string   `json:"zip"`
	Country      string   `json:"country"`
	DXCC         *int     `json:"dxcc"`
	Grid         string   `json:"grid"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	CQZone       *int     `json:"cqZone"`
	ITUZone      *int     `json:"ituZone"`
	IOTA         string   `json:"iota"`
	Continent    string   `json:"continent"`
	Email        string   `json:"email"`
	Website      string   `json:"website"`
	QSLManager   string   `json:"qslManager"`
	LoTW         string   `json:"lotw"`
	EQSL         string   `json:"eqsl"`
	FlagISO2     string   `json:"flagIso2"`
	Source       string   `json:"source"`
	LastLookupAt *string  `json:"lastLookupAt"`
}
