package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"netlog/internal/auth"
	"netlog/internal/config"
	"netlog/internal/db"
	"netlog/internal/logging"
	"netlog/internal/store"
	"netlog/internal/sync"
)

type testEnv struct {
	srv    *httptest.Server
	client *http.Client
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	sqldb, err := db.Open(filepath.Join(t.TempDir(), "api.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	if err := db.Migrate(sqldb); err != nil {
		t.Fatal(err)
	}
	st := store.New(sqldb)
	logger, _ := logging.New(logging.Options{Level: "error", Format: "text", Output: io.Discard})
	cfg := config.Defaults()

	srv := New(Deps{
		Config:   &cfg,
		Logger:   logger,
		Store:    st,
		Auth:     auth.NewService(st),
		Sessions: auth.NewManager(st, false),
		Sync:     sync.NewService(st, nil, nil, logger),
		SPA:      fstest.MapFS{"index.html": {Data: []byte("<html></html>")}},
	})

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	return &testEnv{srv: ts, client: &http.Client{Jar: jar}}
}

func (e *testEnv) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

func TestFullFlow(t *testing.T) {
	e := newTestEnv(t)

	// Bootstrap: first admin needed.
	boot := decode[map[string]any](t, e.do(t, "GET", "/api/bootstrap", nil))
	if boot["needsFirstAdmin"] != true {
		t.Fatalf("expected needsFirstAdmin true, got %v", boot)
	}

	// Register the first admin.
	resp := e.do(t, "POST", "/api/register", map[string]string{
		"callsign": "KF0ACN", "firstName": "Sam", "lastName": "Resto",
		"email": "sam@example.com", "password": "supersecret1",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
	admin := decode[map[string]any](t, resp)
	if admin["role"] != "admin" {
		t.Fatalf("expected admin role, got %v", admin["role"])
	}

	// /api/me reflects the session.
	me := decode[map[string]any](t, e.do(t, "GET", "/api/me", nil))
	if me["callsign"] != "KF0ACN" {
		t.Fatalf("me callsign = %v", me["callsign"])
	}

	netID := "11111111-1111-1111-1111-111111111111"

	// Create a net (sync put).
	push := func(changes []sync.Change) sync.PushResponse {
		r := e.do(t, "POST", "/api/sync", sync.PushRequest{Changes: changes})
		if r.StatusCode != http.StatusOK {
			t.Fatalf("sync status = %d", r.StatusCode)
		}
		return decode[sync.PushResponse](t, r)
	}

	mk := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

	res := push([]sync.Change{{Entity: "net", Op: "put", ID: netID,
		Data: mk(map[string]any{"name": "Tuesday Net", "netDate": "2026-06-23"})}})
	if res.Results[0].Status != "applied" || res.Results[0].Net.Status != "pending" {
		t.Fatalf("create net result: %+v", res.Results[0])
	}

	// Open the net; server stamps start time.
	res = push([]sync.Change{{Entity: "net", Op: "put", ID: netID,
		Data: mk(map[string]any{"status": "open"})}})
	if res.Results[0].Net.Status != "open" || res.Results[0].Net.StartAt == nil {
		t.Fatalf("open net result: %+v", res.Results[0])
	}

	// Add a check-in.
	ciID := "22222222-2222-2222-2222-222222222222"
	res = push([]sync.Change{{Entity: "checkin", Op: "put", ID: ciID,
		Data: mk(map[string]any{"netId": netID, "callsign": "w1aw", "nickname": "Hiram", "hasTraffic": true})}})
	if res.Results[0].Status != "applied" {
		t.Fatalf("checkin result: %+v", res.Results[0])
	}
	if ci := res.Results[0].CheckIn; ci == nil || ci.Callsign != "W1AW" || ci.Seq != 1 {
		t.Fatalf("checkin not normalized/sequenced: %+v", res.Results[0].CheckIn)
	}

	// Net list shows count and open status.
	list := decode[[]map[string]any](t, e.do(t, "GET", "/api/nets", nil))
	if len(list) != 1 || list[0]["checkInCount"].(float64) != 1 || list[0]["ncsCallsign"] != "KF0ACN" {
		t.Fatalf("net list unexpected: %+v", list)
	}

	// Net detail includes the check-in.
	detail := decode[map[string]any](t, e.do(t, "GET", "/api/nets/"+netID, nil))
	if cis := detail["checkins"].([]any); len(cis) != 1 {
		t.Fatalf("expected 1 check-in in detail, got %d", len(cis))
	}

	// Pull since epoch returns the net and check-in.
	pull := decode[sync.PullResponse](t, e.do(t, "GET", "/api/sync?since=", nil))
	if len(pull.Nets) != 1 || len(pull.CheckIns) != 1 {
		t.Fatalf("pull unexpected: %d nets, %d checkins", len(pull.Nets), len(pull.CheckIns))
	}

	// Close the net; server stamps end time.
	res = push([]sync.Change{{Entity: "net", Op: "put", ID: netID,
		Data: mk(map[string]any{"status": "closed"})}})
	if res.Results[0].Net.Status != "closed" || res.Results[0].Net.EndAt == nil {
		t.Fatalf("close net result: %+v", res.Results[0])
	}

	// Editing a check-in on a closed net is rejected.
	res = push([]sync.Change{{Entity: "checkin", Op: "put", ID: ciID,
		Data: mk(map[string]any{"netId": netID, "callsign": "w1aw", "notes": "late edit"})}})
	if res.Results[0].Status != "conflict" {
		t.Fatalf("expected conflict editing closed net, got %+v", res.Results[0])
	}
}

func TestSyncInputHardening(t *testing.T) {
	e := newTestEnv(t)
	reg := e.do(t, "POST", "/api/register", map[string]string{
		"callsign": "KF0ACN", "firstName": "Sam", "lastName": "Resto",
		"email": "sam@example.com", "password": "supersecret1",
	})
	reg.Body.Close()

	mk := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	push := func(changes []sync.Change) sync.PushResponse {
		r := e.do(t, "POST", "/api/sync", sync.PushRequest{Changes: changes})
		return decode[sync.PushResponse](t, r)
	}

	// Control chars + extra whitespace in the net name are sanitized server-side.
	res := push([]sync.Change{{Entity: "net", Op: "put", ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Data: mk(map[string]any{"name": "  Bad\tName  ", "netDate": "2026-06-23"})}})
	if res.Results[0].Status != "applied" || res.Results[0].Net.Name != "Bad Name" {
		t.Fatalf("name not sanitized: %+v", res.Results[0])
	}

	// A malformed id is rejected.
	res = push([]sync.Change{{Entity: "net", Op: "put", ID: "bad id!",
		Data: mk(map[string]any{"name": "X", "netDate": "2026-06-23"})}})
	if res.Results[0].Status != "error" {
		t.Fatalf("expected error for bad id, got %+v", res.Results[0])
	}

	// A non-calendar date is rejected.
	res = push([]sync.Change{{Entity: "net", Op: "put", ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Data: mk(map[string]any{"name": "X", "netDate": "2026-13-40"})}})
	if res.Results[0].Status != "error" {
		t.Fatalf("expected error for bad date, got %+v", res.Results[0])
	}
}

func TestAuthRequired(t *testing.T) {
	e := newTestEnv(t)
	// Without a session, protected endpoints return 401.
	resp := e.do(t, "GET", "/api/nets", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRegistrationClosedAfterFirst(t *testing.T) {
	e := newTestEnv(t)
	first := e.do(t, "POST", "/api/register", map[string]string{
		"callsign": "W1AW", "firstName": "A", "lastName": "B", "email": "a@b.com", "password": "password12",
	})
	first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first register = %d", first.StatusCode)
	}
	second := e.do(t, "POST", "/api/register", map[string]string{
		"callsign": "W2XYZ", "firstName": "C", "lastName": "D", "email": "c@d.com", "password": "password12",
	})
	second.Body.Close()
	if second.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for second register, got %d", second.StatusCode)
	}
}
