// Package adminauth issues and refreshes admin sessions using the common
// password hashing and opaque token store.
package adminauth

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/adminuser"
	"smartreceipt/backend/common/services/auth"

	"smartreceipt/backend/admin/internal/config"
	"smartreceipt/backend/admin/internal/pkg/middleware"
)

// CookiePath scopes the admin refresh cookie.
const CookiePath = "/openapi/admin/auth"

// Session is what a successful admin login returns.
type Session struct {
	Admin        *models.AdminUser
	AccessToken  string
	RefreshToken string
}

// Service is the admin session service.
type Service struct {
	admins *adminuser.Service
	tokens auth.TokenStore
}

// New returns the service.
func New() *Service { return &Service{admins: adminuser.New()} }

// Login verifies email + password.
func (s *Service) Login(ctx context.Context, email, password string) (*models.AdminUser, error) {
	a, err := s.admins.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, adminuser.ErrNotFound) {
			_ = auth.VerifyPassword("$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", password)
			return nil, auth.ErrBadCredentials
		}
		return nil, err
	}
	if !auth.VerifyPassword(a.PasswordHash, password) {
		return nil, auth.ErrBadCredentials
	}
	if a.Disabled {
		return nil, auth.ErrAccountDisabled
	}
	now := time.Now().UTC()
	_ = s.admins.Update(ctx, a.ID, bson.M{"lastLoginAt": now})
	a.LastLoginAt = &now
	return a, nil
}

// IssueSession mints an access token and a refresh token.
func (s *Service) IssueSession(ctx context.Context, a *models.AdminUser, ua, ip string) (*Session, error) {
	c := config.AppConfig
	now := time.Now()
	access, err := auth.SignToken(c.JWTSecret, middleware.ClaimsFor(a, now, now.Add(c.AccessTokenTTL)))
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.Issue(ctx, models.TokenAdminRefresh, a.ID, c.RefreshTokenTTL, ua, ip)
	if err != nil {
		return nil, err
	}
	return &Session{Admin: a, AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh rotates a refresh token.
func (s *Service) Refresh(ctx context.Context, raw, ua, ip string) (*Session, error) {
	id, err := s.tokens.Consume(ctx, models.TokenAdminRefresh, raw, auth.RefreshGrace)
	if err != nil {
		if errors.Is(err, auth.ErrTokenInvalid) {
			return nil, gohttp.ErrUnauthorized
		}
		return nil, err
	}
	a, err := s.admins.FindByID(ctx, id)
	if err != nil || a.Disabled {
		return nil, gohttp.ErrUnauthorized
	}
	return s.IssueSession(ctx, a, ua, ip)
}

// Logout revokes one refresh token.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.tokens.Revoke(ctx, models.TokenAdminRefresh, raw)
}

// Me loads the admin for the current claims.
func (s *Service) Me(ctx context.Context, id bson.ObjectID) (*models.AdminUser, error) {
	a, err := s.admins.FindByID(ctx, id)
	if err != nil {
		return nil, gohttp.ErrUnauthorized
	}
	return a, nil
}

// CreateAdmin hashes the password and inserts an admin (tooling).
func (s *Service) CreateAdmin(ctx context.Context, email, name, password, role string) (*models.AdminUser, error) {
	if err := auth.ValidatePassword(password); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	a := &models.AdminUser{Email: email, Name: name, PasswordHash: hash, Role: role}
	if err := s.admins.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// SecureCookies reports whether cookies must carry the Secure flag.
func SecureCookies() bool { return config.AppConfig.IsProd() }
