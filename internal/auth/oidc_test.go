package auth

import (
	"context"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestOIDCAuthCodeURLIncludesPKCES256(t *testing.T) {
	p := OIDCProvider{
		oauth: oauth2.Config{
			Endpoint: oauth2.Endpoint{AuthURL: "https://idp.example.com/auth"},
		},
		requirePKCE:         true,
		pkceChallengeMethod: "S256",
	}
	got := p.AuthCodeURL("state123", "nonce123", "verifier123")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := u.Query()
	if q.Get("state") != "state123" {
		t.Fatalf("state mismatch: %q", q.Get("state"))
	}
	if q.Get("nonce") != "nonce123" {
		t.Fatalf("nonce mismatch: %q", q.Get("nonce"))
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method mismatch: %q", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("expected code_challenge")
	}
}

func TestOIDCAuthCodeURLIncludesPKCEPlain(t *testing.T) {
	p := OIDCProvider{
		oauth: oauth2.Config{
			Endpoint: oauth2.Endpoint{AuthURL: "https://idp.example.com/auth"},
		},
		requirePKCE:         true,
		pkceChallengeMethod: "plain",
	}
	got := p.AuthCodeURL("state123", "nonce123", "verifier123")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "plain" {
		t.Fatalf("code_challenge_method mismatch: %q", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") != "verifier123" {
		t.Fatalf("code_challenge mismatch: %q", q.Get("code_challenge"))
	}
}

func TestOIDCExchangeRejectsMissingPKCEVerifier(t *testing.T) {
	p := OIDCProvider{
		requirePKCE: true,
	}
	if _, err := p.Exchange(context.Background(), "code123", "nonce123", ""); err == nil {
		t.Fatal("expected missing pkce verifier error")
	}
}
