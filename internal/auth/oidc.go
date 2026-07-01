package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"netlog/internal/config"
)

// OIDCProvider wraps a configured single OIDC provider for the auth-code flow.
type OIDCProvider struct {
	issuer              string
	provider            *oidc.Provider
	verifier            *oidc.IDTokenVerifier
	oauth               oauth2.Config
	requirePKCE         bool
	pkceChallengeMethod string
}

// OIDCClaims are the identity claims we consume from a verified ID token.
type OIDCClaims struct {
	Issuer        string
	Subject       string
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// NewOIDCProvider discovers the provider and builds the OAuth2 config. Returns
// (nil, nil) when OIDC is disabled in config.
func NewOIDCProvider(ctx context.Context, cfg config.OIDC) (*OIDCProvider, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	endpoint := provider.Endpoint()
	switch cfg.TokenEndpointAuthMethod {
	case "client_secret_post":
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	case "client_secret_basic":
		endpoint.AuthStyle = oauth2.AuthStyleInHeader
	}
	return &OIDCProvider{
		issuer:              cfg.Issuer,
		provider:            provider,
		verifier:            provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		requirePKCE:         cfg.RequirePKCE,
		pkceChallengeMethod: cfg.PKCEChallengeMethod,
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     endpoint,
			Scopes:       scopes,
		},
	}, nil
}

// Issuer returns the provider's issuer URL.
func (p *OIDCProvider) Issuer() string { return p.issuer }

// RequiresPKCE reports whether PKCE is enabled for the auth-code flow.
func (p *OIDCProvider) RequiresPKCE() bool { return p.requirePKCE }

// AuthCodeURL builds the authorization redirect URL with CSRF state and replay
// nonce.
func (p *OIDCProvider) AuthCodeURL(state, nonce, pkceVerifier string) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce)}
	if p.requirePKCE {
		if p.pkceChallengeMethod == "plain" {
			opts = append(opts,
				oauth2.SetAuthURLParam("code_challenge_method", "plain"),
				oauth2.SetAuthURLParam("code_challenge", pkceVerifier),
			)
		} else {
			opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
		}
	}
	return p.oauth.AuthCodeURL(state, opts...)
}

// Exchange completes the auth-code flow: it swaps code for tokens, verifies the
// ID token (signature, audience, expiry, nonce), and returns the claims.
func (p *OIDCProvider) Exchange(ctx context.Context, code, expectedNonce, pkceVerifier string) (OIDCClaims, error) {
	opts := []oauth2.AuthCodeOption{}
	if p.requirePKCE {
		if pkceVerifier == "" {
			return OIDCClaims{}, errors.New("oidc pkce verifier missing")
		}
		opts = append(opts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := p.oauth.Exchange(ctx, code, opts...)
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("oidc token exchange: %w", err)
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok || rawID == "" {
		return OIDCClaims{}, errors.New("oidc response missing id_token")
	}
	idToken, err := p.verifier.Verify(ctx, rawID)
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("verify id_token: %w", err)
	}
	if expectedNonce != "" && idToken.Nonce != expectedNonce {
		return OIDCClaims{}, errors.New("oidc nonce mismatch")
	}
	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return OIDCClaims{}, fmt.Errorf("parse id_token claims: %w", err)
	}
	claims.Issuer = idToken.Issuer
	claims.Subject = idToken.Subject
	return claims, nil
}
