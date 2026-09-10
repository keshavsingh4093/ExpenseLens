// Package auth implements registration, login, sessions, password reset, and
// social login. Handlers live in handler.go; this file is the business logic.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/config"
	"expenselens/backend/internal/mail"
	"expenselens/backend/internal/server"
	"expenselens/backend/internal/users"
)

// Service holds auth dependencies.
type Service struct {
	cfg    *config.Config
	users  *users.Repo
	tokens *TokenRepo
	mail   mail.Sender
}

// NewService constructs the auth service.
func NewService(cfg *config.Config, u *users.Repo, t *TokenRepo, m mail.Sender) *Service {
	return &Service{cfg: cfg, users: u, tokens: t, mail: m}
}

// Session is what a successful login returns.
type Session struct {
	User         *users.User
	AccessToken  string
	RefreshToken string
}

// refreshGrace is how long a just-rotated refresh token stays accepted.
const refreshGrace = 30 * time.Second

var errBadCredentials = &server.APIError{Status: 401, Code: "invalid_credentials", Message: "email or password is incorrect"}

// Register creates a password account.
func (s *Service) Register(ctx context.Context, email, name, password string) (*users.User, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, server.Validation(map[string]string{"password": err.Error()})
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &users.User{Email: email, Name: name, PasswordHash: hash, Preferences: users.DefaultPreferences()}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, users.ErrEmailTaken) {
			return nil, server.Conflict("email_taken", "an account with this email already exists")
		}
		return nil, err
	}
	return u, nil
}

// Login verifies email + password.
func (s *Service) Login(ctx context.Context, email, password string) (*users.User, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			// burn time so missing users are indistinguishable from bad passwords
			_ = VerifyPassword("$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", password)
			return nil, errBadCredentials
		}
		return nil, err
	}
	if !u.HasPassword() || !VerifyPassword(u.PasswordHash, password) {
		return nil, errBadCredentials
	}
	return u, nil
}

// IssueSession mints an access token and a refresh token for u.
func (s *Service) IssueSession(ctx context.Context, u *users.User, ua, ip string) (*Session, error) {
	access, err := SignAccessToken(s.cfg.JWTSecret, u.ID.Hex(), s.cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.issue(ctx, kindRefresh, u.ID, s.cfg.RefreshTokenTTL, ua, ip)
	if err != nil {
		return nil, err
	}
	return &Session{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh rotates a refresh token and returns a new session.
func (s *Service) Refresh(ctx context.Context, raw, ua, ip string) (*Session, error) {
	uid, err := s.tokens.consume(ctx, kindRefresh, raw, refreshGrace)
	if err != nil {
		if errors.Is(err, errTokenInvalid) {
			return nil, server.ErrUnauthorized
		}
		return nil, err
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return nil, server.ErrUnauthorized
	}
	return s.IssueSession(ctx, u, ua, ip)
}

// Logout revokes one refresh token.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.tokens.revoke(ctx, kindRefresh, raw)
}

// ForgotPassword emails a reset link. Always succeeds to avoid user enumeration.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil
	}
	raw, err := s.tokens.issue(ctx, kindReset, u.ID, time.Hour, "", "")
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.AppURL, url.QueryEscape(raw))
	return s.mail.Send(ctx, mail.Message{
		To:      u.Email,
		Subject: "Reset your ExpenseLens password",
		Body:    "Hi " + u.Name + ",\n\nUse this link to set a new password (valid for 1 hour):\n" + link + "\n\nIf you did not request this, ignore this email.",
	})
}

// ResetPassword consumes a reset token and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, raw, password string) error {
	if err := ValidatePassword(password); err != nil {
		return server.Validation(map[string]string{"password": err.Error()})
	}
	uid, err := s.tokens.consume(ctx, kindReset, raw, 0)
	if err != nil {
		if errors.Is(err, errTokenInvalid) {
			return server.BadRequest("reset link is invalid or has expired")
		}
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.users.Update(ctx, uid, bson.M{"passwordHash": hash}); err != nil {
		return err
	}
	return s.tokens.revokeAll(ctx, kindRefresh, uid)
}

// ChangePassword verifies the current password (if any) and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, uid bson.ObjectID, current, next string) error {
	if err := ValidatePassword(next); err != nil {
		return server.Validation(map[string]string{"newPassword": err.Error()})
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return server.ErrUnauthorized
	}
	if u.HasPassword() && !VerifyPassword(u.PasswordHash, current) {
		return server.Validation(map[string]string{"currentPassword": "is incorrect"})
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return s.users.Update(ctx, uid, bson.M{"passwordHash": hash})
}

// LoginWithIdentity finds or creates the user for a social identity.
// Resolution order: existing provider link → existing account with the same
// verified email (link it) → new account.
func (s *Service) LoginWithIdentity(ctx context.Context, id Identity) (*users.User, error) {
	if u, err := s.users.FindByProvider(ctx, id.Provider, id.ProviderUserID); err == nil {
		return u, nil
	} else if !errors.Is(err, users.ErrNotFound) {
		return nil, err
	}
	if id.Email == "" {
		return nil, server.BadRequest("the provider did not share an email address")
	}
	link := users.ProviderLink{Provider: id.Provider, ProviderUserID: id.ProviderUserID}
	if u, err := s.users.FindByEmail(ctx, id.Email); err == nil {
		if !id.EmailVerified {
			return nil, server.Conflict("email_unverified", "this email belongs to an existing account; sign in with your password first")
		}
		if err := s.users.LinkProvider(ctx, u.ID, link); err != nil {
			return nil, err
		}
		set := bson.M{"emailVerified": true}
		if u.AvatarURL == "" && id.AvatarURL != "" {
			set["avatarUrl"] = id.AvatarURL
		}
		_ = s.users.Update(ctx, u.ID, set)
		return s.users.FindByID(ctx, u.ID)
	} else if !errors.Is(err, users.ErrNotFound) {
		return nil, err
	}
	name := id.Name
	if name == "" {
		name = id.Email
	}
	u := &users.User{
		Email: id.Email, EmailVerified: id.EmailVerified, Name: name, AvatarURL: id.AvatarURL,
		Providers:   []users.ProviderLink{{Provider: link.Provider, ProviderUserID: link.ProviderUserID, LinkedAt: time.Now().UTC()}},
		Preferences: users.DefaultPreferences(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// DeleteAccount removes the user and all their tokens. Domain data cascade is
// added as those packages land (receipts, budgets, warranties, notifications).
func (s *Service) DeleteAccount(ctx context.Context, uid bson.ObjectID) error {
	if err := s.tokens.revokeAll(ctx, kindRefresh, uid); err != nil {
		return err
	}
	return s.users.Delete(ctx, uid)
}
