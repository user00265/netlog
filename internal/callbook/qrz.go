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

// qrzBaseURL pins the QRZ XML spec version in the path, per the QRZ spec.
const qrzBaseURL = "https://xmldata.qrz.com/xml/1.34/"

// QRZ is a QRZ XML callbook provider. It caches the session key and transparently
// re-authenticates when QRZ reports the session has timed out.
type QRZ struct {
	http     *httpClient
	baseURL  string
	username string
	password string
	agent    string

	mu         sync.Mutex
	sessionKey string
}

// NewQRZ builds a QRZ provider.
func NewQRZ(http *httpClient, username, password, agent string) *QRZ {
	if agent == "" {
		agent = "NetLog/0.1"
	}
	return &QRZ{http: http, baseURL: qrzBaseURL, username: username, password: password, agent: agent}
}

// Name returns the provider key.
func (q *QRZ) Name() string { return ProviderQRZ }

// qrzDatabase models the QRZ XML response. Field tags use local element names;
// Go matches them regardless of the document's default namespace.
type qrzDatabase struct {
	XMLName  xml.Name     `xml:"QRZDatabase"`
	Session  qrzSession   `xml:"Session"`
	Callsign *qrzCallsign `xml:"Callsign"`
}

type qrzSession struct {
	Key   string `xml:"Key"`
	Error string `xml:"Error"`
	Count string `xml:"Count"`
}

type qrzCallsign struct {
	Call     string `xml:"call"`
	Fname    string `xml:"fname"`
	Name     string `xml:"name"`
	Nickname string `xml:"nickname"`
	Addr1    string `xml:"addr1"`
	Addr2    string `xml:"addr2"`
	State    string `xml:"state"`
	Zip      string `xml:"zip"`
	Country  string `xml:"country"`
	Lat      string `xml:"lat"`
	Lon      string `xml:"lon"`
	Grid     string `xml:"grid"`
	County   string `xml:"county"`
	Land     string `xml:"land"`
	DXCC     string `xml:"dxcc"`
	Email    string `xml:"email"`
	URL      string `xml:"url"`
	Class    string `xml:"class"`
	QSLMgr   string `xml:"qslmgr"`
	LoTW     string `xml:"lotw"`
	EQSL     string `xml:"eqsl"`
	CQZone   string `xml:"cqzone"`
	ITUZone  string `xml:"ituzone"`
	IOTA     string `xml:"iota"`
}

// Lookup resolves a callsign, authenticating first and retrying once on session
// timeout.
func (q *QRZ) Lookup(ctx context.Context, callsign string) (*Record, error) {
	rec, err := q.lookupOnce(ctx, callsign)
	if errors.Is(err, errSessionExpired) {
		// Force a fresh session and try one more time.
		q.mu.Lock()
		q.sessionKey = ""
		q.mu.Unlock()
		rec, err = q.lookupOnce(ctx, callsign)
	}
	return rec, err
}

// errSessionExpired signals that the cached session was rejected and a retry
// with a fresh key is warranted.
var errSessionExpired = errors.New("qrz session expired")

func (q *QRZ) lookupOnce(ctx context.Context, callsign string) (*Record, error) {
	key, err := q.ensureSession(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{"s": {key}, "callsign": {callsign}, "agent": {q.agent}}
	body, err := q.http.get(ctx, q.baseURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var db qrzDatabase
	if err := xml.Unmarshal(body, &db); err != nil {
		return nil, fmt.Errorf("parse qrz response: %w", err)
	}
	if e := strings.ToLower(db.Session.Error); e != "" {
		switch {
		case strings.Contains(e, "session") || strings.Contains(e, "timeout") || strings.Contains(e, "invalid"):
			return nil, errSessionExpired
		case strings.Contains(e, "not found"):
			return nil, ErrCallsignNotFound
		default:
			return nil, fmt.Errorf("qrz error: %s", db.Session.Error)
		}
	}
	if db.Callsign == nil {
		return nil, ErrCallsignNotFound
	}
	return qrzToRecord(db.Callsign, string(body)), nil
}

// ensureSession returns a cached session key, logging in if necessary.
func (q *QRZ) ensureSession(ctx context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.sessionKey != "" {
		return q.sessionKey, nil
	}
	params := url.Values{"username": {q.username}, "password": {q.password}, "agent": {q.agent}}
	body, err := q.http.get(ctx, q.baseURL+"?"+params.Encode())
	if err != nil {
		return "", err
	}
	var db qrzDatabase
	if err := xml.Unmarshal(body, &db); err != nil {
		return "", fmt.Errorf("parse qrz session: %w", err)
	}
	if db.Session.Key == "" {
		if db.Session.Error != "" {
			return "", fmt.Errorf("qrz login failed: %s", db.Session.Error)
		}
		return "", errors.New("qrz login failed: no session key")
	}
	q.sessionKey = db.Session.Key
	return q.sessionKey, nil
}

func qrzToRecord(c *qrzCallsign, raw string) *Record {
	return &Record{
		Source:    ProviderQRZ,
		Callsign:  strings.ToUpper(c.Call),
		FirstName: c.Fname,
		LastName:  c.Name,
		Nickname:  c.Nickname,
		// QRZ uses addr1 = street and addr2 = city; there is no second street
		// line, so City (not Address2) is populated from addr2.
		Address1:   c.Addr1,
		City:       c.Addr2,
		State:      c.State,
		Zip:        c.Zip,
		Country:    c.Country,
		DXCC:       intPtr(c.DXCC),
		Grid:       c.Grid,
		Latitude:   floatPtr(c.Lat),
		Longitude:  floatPtr(c.Lon),
		CQZone:     intPtr(c.CQZone),
		ITUZone:    intPtr(c.ITUZone),
		IOTA:       c.IOTA,
		Email:      c.Email,
		Website:    c.URL,
		QSLManager: c.QSLMgr,
		LoTW:       c.LoTW,
		EQSL:       c.EQSL,
		RawBody:    raw,
	}
}
