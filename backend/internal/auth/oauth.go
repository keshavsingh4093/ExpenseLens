package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"expenselens/backend/internal/config"
)

// Identity is what a provider tells us about the signed-in person.
type Identity struct {
	Provider       string
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	AvatarURL      string
}

// Provider is one configured OIDC login.
type Provider struct {
	Name         string
	ResponseMode string // "" or "form_post" (Apple)
	cfg          oauth2.Config
	verifier     *oidc.IDTokenVerifier
	secret       func() (string, error) // dynamic client secret (Apple); nil = static
}

// NewProviders builds every provider that has credentials configured.
// Discovery failures for one provider disable it and are returned in errs.
func NewProviders(ctx context.Context, cfg *config.Config) (map[string]*Provider, []error) {
	out := map[string]*Provider{}
	var errs []error
	redirect := func(name string) string { return cfg.APIURL + "/api/v1/auth/" + name + "/callback" }

	if cfg.OAuth.GoogleClientID != "" {
		p, err := newOIDC(ctx, "google", "https://accounts.google.com", cfg.OAuth.GoogleClientID,
			cfg.OAuth.GoogleClientSecret, redirect("google"), []string{oidc.ScopeOpenID, "email", "profile"}, false)
		if err != nil {
			errs = append(errs, err)
		} else {
			out["google"] = p
		}
	}
	if cfg.OAuth.MicrosoftClientID != "" {
		issuer := "https://login.microsoftonline.com/" + cfg.OAuth.MicrosoftTenant + "/v2.0"
		// Multi-tenant ("common"/"organizations"/"consumers") discovery docs carry a
		// templated issuer, so skip the issuer check for those.
		skip := cfg.OAuth.MicrosoftTenant == "common" || cfg.OAuth.MicrosoftTenant == "organizations" || cfg.OAuth.MicrosoftTenant == "consumers"
		p, err := newOIDC(ctx, "microsoft", issuer, cfg.OAuth.MicrosoftClientID,
			cfg.OAuth.MicrosoftClientSecret, redirect("microsoft"), []string{oidc.ScopeOpenID, "email", "profile"}, skip)
		if err != nil {
			errs = append(errs, err)
		} else {
			out["microsoft"] = p
		}
	}
	if cfg.OAuth.AppleClientID != "" {
		p, err := newOIDC(ctx, "apple", "https://appleid.apple.com", cfg.OAuth.AppleClientID,
			"", redirect("apple"), []string{oidc.ScopeOpenID, "email", "name"}, false)
		if err != nil {
			errs = append(errs, err)
		} else {
			p.ResponseMode = "form_post"
			oc := cfg.OAuth
			p.secret = func() (string, error) {
				return appleClientSecret(oc.AppleTeamID, oc.AppleClientID, oc.AppleKeyID, oc.ApplePrivateKey)
			}
			out["apple"] = p
		}
	}
	return out, errs
}

func newOIDC(ctx context.Context, name, issuer, clientID, clientSecret, redirect string, scopes []string, skipIssuer bool) (*Provider, error) {
	if skipIssuer {
		ctx = oidc.InsecureIssuerURLContext(ctx, issuer)
	}
	prov, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for %s: %w", name, err)
	}
	return &Provider{
		Name: name,
		cfg: oauth2.Config{
			ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirect,
			Endpoint: prov.Endpoint(), Scopes: scopes,
		},
		verifier: prov.Verifier(&oidc.Config{ClientID: clientID, SkipIssuerCheck: skipIssuer}),
	}, nil
}

// AuthCodeURL builds the redirect to the provider.
func (p *Provider) AuthCodeURL(state, nonce string) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce)}
	if p.ResponseMode != "" {
		opts = append(opts, oauth2.SetAuthURLParam("response_mode", p.ResponseMode))
	}
	return p.cfg.AuthCodeURL(state, opts...)
}

// Exchange trades the code for tokens and returns the verified identity.
// appleUserJSON is the optional `user` form field Apple posts on first sign-in.
func (p *Provider) Exchange(ctx context.Context, code, nonce, appleUserJSON string) (*Identity, error) {
	cfg := p.cfg
	if p.secret != nil {
		s, err := p.secret()
		if err != nil {
			return nil, err
		}
		cfg.ClientSecret = s
	}
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%s exchange: %w", p.Name, err)
	}
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		return nil, errors.New(p.Name + ": no id_token in response")
	}
	idt, err := p.verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("%s id_token: %w", p.Name, err)
	}
	if idt.Nonce != nonce {
		return nil, errors.New(p.Name + ": nonce mismatch")
	}
	var c struct {
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"` // bool (Google) or "true" string (Apple)
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Preferred     string `json:"preferred_username"`
	}
	if err := idt.Claims(&c); err != nil {
		return nil, err
	}
	id := &Identity{Provider: p.Name, ProviderUserID: idt.Subject, Email: strings.ToLower(c.Email), Name: c.Name, AvatarURL: c.Picture}
	switch v := c.EmailVerified.(type) {
	case bool:
		id.EmailVerified = v
	case string:
		id.EmailVerified = v == "true"
	}
	// Microsoft and Apple don't emit email_verified; both only hand out
	// verified addresses on the openid email scope.
	if p.Name == "microsoft" || p.Name == "apple" {
		id.EmailVerified = id.Email != ""
	}
	if id.Name == "" && appleUserJSON != "" {
		var au struct {
			Name struct {
				First string `json:"firstName"`
				Last  string `json:"lastName"`
			} `json:"name"`
		}
		if json.Unmarshal([]byte(appleUserJSON), &au) == nil {
			id.Name = strings.TrimSpace(au.Name.First + " " + au.Name.Last)
		}
	}
	if id.Name == "" && c.Preferred != "" && !strings.Contains(c.Preferred, "@") {
		id.Name = c.Preferred
	}
	return id, nil
}

// appleClientSecret builds the ES256 JWT Apple requires as client_secret.
func appleClientSecret(teamID, clientID, keyID, pemKey string) (string, error) {
	pemKey = strings.ReplaceAll(pemKey, `\n`, "\n")
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return "", errors.New("apple: private key is not valid PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("apple: parse key: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return "", errors.New("apple: key is not ECDSA")
	}
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Issuer:    teamID,
		Subject:   clientID,
		Audience:  jwt.ClaimStrings{"https://appleid.apple.com"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
	})
	t.Header["kid"] = keyID
	return t.SignedString(ec)
}
