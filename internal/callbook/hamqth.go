package callbook

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

const hamqthBaseURL = "https://www.hamqth.com/xml.php"

// HamQTH is a HamQTH callbook provider. Sessions last one hour; an expired
// session triggers a transparent re-login.
type HamQTH struct {
	http     *httpClient
	baseURL  string
	username string
	password string
	program  string

	mu        sync.Mutex
	sessionID string
}

// NewHamQTH builds a HamQTH provider.
func NewHamQTH(http *httpClient, username, password, program string) *HamQTH {
	if program == "" {
		program = "NetLog"
	}
	return &HamQTH{http: http, baseURL: hamqthBaseURL, username: username, password: password, program: program}
}

// Name returns the provider key.
func (h *HamQTH) Name() string { return ProviderHamQTH }

type hamqthDocument struct {
	XMLName xml.Name      `xml:"HamQTH"`
	Session hamqthSession `xml:"session"`
	Search  *hamqthSearch `xml:"search"`
}

type hamqthSession struct {
	SessionID string `xml:"session_id"`
	Error     string `xml:"error"`
}

type hamqthSearch struct {
	Callsign   string `xml:"callsign"`
	Nick       string `xml:"nick"`
	QTH        string `xml:"qth"`
	Country    string `xml:"country"`
	ADIF       string `xml:"adif"`
	ITU        string `xml:"itu"`
	CQ         string `xml:"cq"`
	Grid       string `xml:"grid"`
	AdrName    string `xml:"adr_name"`
	Street1    string `xml:"adr_street1"`
	Street2    string `xml:"adr_street2"`
	City       string `xml:"adr_city"`
	Zip        string `xml:"adr_zip"`
	AdrCountry string `xml:"adr_country"`
	USState    string `xml:"us_state"`
	USCounty   string `xml:"us_county"`
	IOTA       string `xml:"iota"`
	QSLVia     string `xml:"qsl_via"`
	LoTW       string `xml:"lotw"`
	EQSL       string `xml:"eqsl"`
	Email      string `xml:"email"`
	Web        string `xml:"web"`
	Latitude   string `xml:"latitude"`
	Longitude  string `xml:"longitude"`
	Continent  string `xml:"continent"`
}

// Lookup resolves a callsign, retrying once if the session has expired.
func (h *HamQTH) Lookup(ctx context.Context, callsign string) (*Record, error) {
	rec, err := h.lookupOnce(ctx, callsign)
	if errors.Is(err, errSessionExpired) {
		h.mu.Lock()
		h.sessionID = ""
		h.mu.Unlock()
		rec, err = h.lookupOnce(ctx, callsign)
	}
	return rec, err
}

func (h *HamQTH) lookupOnce(ctx context.Context, callsign string) (*Record, error) {
	id, err := h.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	params := url.Values{"id": {id}, "callsign": {callsign}, "prg": {h.program}}
	body, err := h.http.get(ctx, h.baseURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var doc hamqthDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse hamqth response: %w", err)
	}
	if e := strings.ToLower(doc.Session.Error); e != "" {
		switch {
		case strings.Contains(e, "session"):
			return nil, errSessionExpired
		case strings.Contains(e, "not found"):
			return nil, ErrCallsignNotFound
		default:
			return nil, fmt.Errorf("hamqth error: %s", doc.Session.Error)
		}
	}
	if doc.Search == nil {
		return nil, ErrCallsignNotFound
	}
	return hamqthToRecord(doc.Search, string(body)), nil
}

func (h *HamQTH) ensureSession(ctx context.Context) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessionID != "" {
		return h.sessionID, nil
	}
	params := url.Values{"u": {h.username}, "p": {h.password}}
	body, err := h.http.get(ctx, h.baseURL+"?"+params.Encode())
	if err != nil {
		return "", err
	}
	var doc hamqthDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("parse hamqth session: %w", err)
	}
	if doc.Session.SessionID == "" {
		if doc.Session.Error != "" {
			return "", fmt.Errorf("hamqth login failed: %s", doc.Session.Error)
		}
		return "", errors.New("hamqth login failed: no session id")
	}
	h.sessionID = doc.Session.SessionID
	return h.sessionID, nil
}

func hamqthToRecord(s *hamqthSearch, raw string) *Record {
	country := s.Country
	if country == "" {
		country = s.AdrCountry
	}
	return &Record{
		Source:     ProviderHamQTH,
		Callsign:   strings.ToUpper(s.Callsign),
		FirstName:  s.AdrName, // HamQTH provides a single name field
		Nickname:   s.Nick,
		Address1:   strings.TrimSpace(s.Street1 + " " + s.Street2),
		City:       s.City,
		State:      s.USState,
		Zip:        s.Zip,
		Country:    country,
		DXCC:       intPtr(s.ADIF),
		Grid:       s.Grid,
		Latitude:   floatPtr(s.Latitude),
		Longitude:  floatPtr(s.Longitude),
		CQZone:     intPtr(s.CQ),
		ITUZone:    intPtr(s.ITU),
		IOTA:       s.IOTA,
		Continent:  s.Continent,
		Email:      s.Email,
		Website:    s.Web,
		QSLManager: s.QSLVia,
		LoTW:       s.LoTW,
		EQSL:       s.EQSL,
		RawBody:    raw,
	}
}
