package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"netlog/internal/db"
	"netlog/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	sqldb, err := db.Open(filepath.Join(t.TempDir(), "auth.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	if err := db.Migrate(sqldb); err != nil {
		t.Fatal(err)
	}
	return store.New(sqldb)
}

func sampleInput() AccountInput {
	return AccountInput{
		Callsign:  "kf0acn",
		FirstName: "Sam",
		LastName:  "Resto",
		Email:     "sam@example.com",
		Password:  "supersecret1",
	}
}

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("hunter2hunter2")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("hunter2hunter2", hash)
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}
	ok, _ = VerifyPassword("wrong", hash)
	if ok {
		t.Error("expected mismatch for wrong password")
	}
	ok, _ = VerifyPassword("anything", "")
	if ok {
		t.Error("empty hash must never verify")
	}
}

func TestFirstAdminFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newTestStore(t))

	need, err := svc.NeedsFirstAdmin(ctx)
	if err != nil || !need {
		t.Fatalf("expected first-admin needed, got need=%v err=%v", need, err)
	}

	admin, err := svc.RegisterFirstAdmin(ctx, sampleInput())
	if err != nil {
		t.Fatalf("RegisterFirstAdmin: %v", err)
	}
	if !admin.IsAdmin() {
		t.Error("first account must be admin")
	}
	if admin.Callsign != "KF0ACN" {
		t.Errorf("callsign not normalized: %q", admin.Callsign)
	}

	need, _ = svc.NeedsFirstAdmin(ctx)
	if need {
		t.Error("first-admin should no longer be needed")
	}

	// Second self-registration is closed.
	in2 := sampleInput()
	in2.Callsign = "w1aw"
	in2.Email = "w1aw@example.com"
	if _, err := svc.RegisterFirstAdmin(ctx, in2); !errors.Is(err, ErrRegistrationClosed) {
		t.Errorf("expected ErrRegistrationClosed, got %v", err)
	}

	// Admin-created user is a normal user.
	user, err := svc.CreateUser(ctx, in2)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.IsAdmin() {
		t.Error("created user must not be admin")
	}

	// Duplicate callsign rejected.
	if _, err := svc.CreateUser(ctx, in2); !errors.Is(err, ErrCallsignTaken) {
		t.Errorf("expected ErrCallsignTaken, got %v", err)
	}

	// Duplicate email (different callsign) rejected.
	in3 := sampleInput()
	in3.Callsign = "n0call"
	in3.Email = "w1aw@example.com" // same as in2
	if _, err := svc.CreateUser(ctx, in3); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newTestStore(t))
	if _, err := svc.RegisterFirstAdmin(ctx, sampleInput()); err != nil {
		t.Fatal(err)
	}

	// Correct credentials (callsign case-insensitive).
	if _, err := svc.Authenticate(ctx, "KF0ACN", "supersecret1"); err != nil {
		t.Errorf("expected success, got %v", err)
	}
	// Wrong password.
	if _, err := svc.Authenticate(ctx, "KF0ACN", "nope"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
	// Unknown callsign yields the same error (no enumeration).
	if _, err := svc.Authenticate(ctx, "N0BODY", "whatever1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSessionIssueAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	svc := NewService(st)
	mgr := NewManager(st, false)

	admin, err := svc.RegisterFirstAdmin(ctx, sampleInput())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	if _, err := mgr.Issue(ctx, rec, admin.ID); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	got, err := mgr.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got.ID != admin.ID {
		t.Errorf("got user %q, want %q", got.ID, admin.ID)
	}

	// No cookie => ErrNoSession.
	if _, err := mgr.Authenticate(httptest.NewRequest(http.MethodGet, "/", nil)); !errors.Is(err, ErrNoSession) {
		t.Errorf("expected ErrNoSession, got %v", err)
	}
}
