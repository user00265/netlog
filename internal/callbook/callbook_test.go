package callbook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"netlog/internal/db"
	"netlog/internal/dxcc"
	"netlog/internal/logging"
	"netlog/internal/store"
)

func testLogger() *slog.Logger {
	l, _ := logging.New(logging.Options{Level: "error", Format: "text", Output: io.Discard})
	return l
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	sqldb, err := db.Open(filepath.Join(t.TempDir(), "cb.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	if err := db.Migrate(sqldb); err != nil {
		t.Fatal(err)
	}
	return store.New(sqldb)
}

const qrzSessionXML = `<?xml version="1.0"?>
<QRZDatabase version="1.34" xmlns="http://www.qrz.com">
  <Session><Key>SESSIONKEY123</Key><Count>1</Count></Session>
</QRZDatabase>`

const qrzCallsignXML = `<?xml version="1.0"?>
<QRZDatabase version="1.34" xmlns="http://www.qrz.com">
  <Callsign>
    <call>W1AW</call><fname>Hiram</fname><name>Maxim</name><nickname>The Old Man</nickname>
    <addr1>225 Main St</addr1><addr2>Newington</addr2><state>CT</state><zip>06111</zip>
    <country>United States</country><dxcc>291</dxcc><grid>FN31pr</grid>
    <lat>41.714</lat><lon>-72.727</lon><cqzone>5</cqzone><ituzone>8</ituzone>
    <email>w1aw@arrl.org</email><lotw>1</lotw>
  </Callsign>
  <Session><Key>SESSIONKEY123</Key><Count>2</Count></Session>
</QRZDatabase>`

func TestQRZLoginAndLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		if r.URL.Query().Get("username") != "" {
			io.WriteString(w, qrzSessionXML)
			return
		}
		io.WriteString(w, qrzCallsignXML)
	}))
	defer srv.Close()

	q := NewQRZ(newHTTPClient(srv.Client(), testLogger()), "user", "pass", "NetLog/test")
	q.baseURL = srv.URL

	rec, err := q.Lookup(context.Background(), "W1AW")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if rec.Callsign != "W1AW" || rec.FirstName != "Hiram" || rec.LastName != "Maxim" {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.Nickname != "The Old Man" {
		t.Errorf("nickname = %q", rec.Nickname)
	}
	if rec.DXCC == nil || *rec.DXCC != 291 {
		t.Errorf("dxcc not parsed: %+v", rec.DXCC)
	}
	if rec.Latitude == nil || *rec.Latitude != 41.714 {
		t.Errorf("lat not parsed: %+v", rec.Latitude)
	}
	if rec.RawBody == "" {
		t.Error("raw body should be preserved")
	}
}

const qrzNotFoundXML = `<?xml version="1.0"?>
<QRZDatabase version="1.34" xmlns="http://www.qrz.com">
  <Session><Key>SESSIONKEY123</Key><Error>Not found: NOCALL</Error></Session>
</QRZDatabase>`

func TestQRZNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("username") != "" {
			io.WriteString(w, qrzSessionXML)
			return
		}
		io.WriteString(w, qrzNotFoundXML)
	}))
	defer srv.Close()

	q := NewQRZ(newHTTPClient(srv.Client(), testLogger()), "user", "pass", "x")
	q.baseURL = srv.URL
	if _, err := q.Lookup(context.Background(), "NOCALL"); !errors.Is(err, ErrCallsignNotFound) {
		t.Errorf("expected ErrCallsignNotFound, got %v", err)
	}
}

const hamqthSessionXML = `<?xml version="1.0"?>
<HamQTH version="2.7" xmlns="https://www.hamqth.com"><session><session_id>HQID999</session_id></session></HamQTH>`

const hamqthSearchXML = `<?xml version="1.0"?>
<HamQTH version="2.7" xmlns="https://www.hamqth.com">
  <search>
    <callsign>g3abc</callsign><nick>Bob</nick><adr_name>Robert Smith</adr_name>
    <country>England</country><adif>223</adif><grid>IO91</grid><cq>14</cq><itu>27</itu>
    <continent>EU</continent><email>g3abc@example.com</email><web>http://g3abc.example</web>
  </search>
</HamQTH>`

func TestHamQTHLoginAndLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("u") != "" {
			io.WriteString(w, hamqthSessionXML)
			return
		}
		io.WriteString(w, hamqthSearchXML)
	}))
	defer srv.Close()

	h := NewHamQTH(newHTTPClient(srv.Client(), testLogger()), "user", "pass", "NetLog")
	h.baseURL = srv.URL

	rec, err := h.Lookup(context.Background(), "G3ABC")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if rec.Callsign != "G3ABC" || rec.FirstName != "Robert Smith" || rec.Nickname != "Bob" {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.DXCC == nil || *rec.DXCC != 223 {
		t.Errorf("adif not parsed: %+v", rec.DXCC)
	}
}

// fakeProvider lets us drive resolver behavior deterministically.
type fakeProvider struct {
	name string
	rec  *Record
	err  error
}

func (f fakeProvider) Name() string { return f.name }
func (f fakeProvider) Lookup(context.Context, string) (*Record, error) {
	return f.rec, f.err
}

const ctySample = `<clublog date='2026-06-01T00:00:00+00:00'>
<entities>
	<entity><adif>1</adif><name>CANADA</name><prefix>VE</prefix><deleted>false</deleted><cqz>5</cqz><cont>NA</cont><long>-80</long><lat>45</lat></entity>
	<entity><adif>291</adif><name>UNITED STATES OF AMERICA</name><prefix>K</prefix><deleted>false</deleted><cqz>5</cqz><cont>NA</cont><long>-90</long><lat>37</lat></entity>
</entities>
<exceptions></exceptions>
<prefixes>
	<prefix record='1'><call>W</call><entity>UNITED STATES OF AMERICA</entity><adif>291</adif><cqz>5</cqz><cont>NA</cont><long>-90</long><lat>37</lat></prefix>
	<prefix record='2'><call>VE</call><entity>CANADA</entity><adif>1</adif><cqz>5</cqz><cont>NA</cont><long>-80</long><lat>45</lat></prefix>
</prefixes>
<zone_exceptions></zone_exceptions>
</clublog>`

func loadDXCC(t *testing.T) *dxcc.DB {
	t.Helper()
	db, err := dxcc.Load(strings.NewReader(ctySample))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestResolverPrimaryFallbackAndCache(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	dx := loadDXCC(t)

	// Primary returns not-found; fallback answers.
	primary := fakeProvider{name: "qrz", err: ErrCallsignNotFound}
	fallback := fakeProvider{name: "hamqth", rec: &Record{Source: ProviderHamQTH, Callsign: "W1AW", FirstName: "Hiram", RawBody: "<x/>"}}

	r := &Resolver{providers: []Provider{primary, fallback}, store: st, dxcc: dx, logger: testLogger()}

	data, err := r.Lookup(ctx, "w1aw", false)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if data.Source != ProviderHamQTH || data.FirstName != "Hiram" {
		t.Errorf("expected fallback record, got %+v", data)
	}
	// DXCC enrichment added the flag.
	if data.FlagISO2 != "us" {
		t.Errorf("expected flag us from DXCC enrichment, got %q", data.FlagISO2)
	}

	// Now cached: a non-refresh lookup should not need providers (use failing
	// providers to prove the cache is consulted).
	r2 := &Resolver{providers: []Provider{fakeProvider{name: "qrz", err: errors.New("boom")}}, store: st, dxcc: dx, logger: testLogger()}
	cached, err := r2.Lookup(ctx, "W1AW", false)
	if err != nil {
		t.Fatalf("cached Lookup: %v", err)
	}
	if cached.FirstName != "Hiram" {
		t.Errorf("expected cached record, got %+v", cached)
	}
}

func TestResolverFlagOnlyWhenNoCallbook(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	dx := loadDXCC(t)

	// No providers answer, but DXCC resolves the prefix → flag-only record.
	r := &Resolver{providers: []Provider{fakeProvider{name: "qrz", err: ErrCallsignNotFound}}, store: st, dxcc: dx, logger: testLogger()}
	data, err := r.Lookup(ctx, "VE3XYZ", false)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if data.FlagISO2 != "ca" {
		t.Errorf("expected ca flag, got %q", data.FlagISO2)
	}
	if data.DXCC == nil || *data.DXCC != 1 {
		t.Errorf("expected dxcc 1, got %v", data.DXCC)
	}
}
