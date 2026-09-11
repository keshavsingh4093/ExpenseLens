// Package auth implements password hashing, access tokens, opaque
// refresh/reset tokens, OIDC providers and the session lifecycle shared by
// the user and admin portals. HTTP handlers live in each portal.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/mail"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/user"
)

// Config tunes the session service.
type Config struct {
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	AppURL     string // SPA origin, used in emailed links
}

// ClaimsFactory builds the access-token claims for a user. Portals supply
// their own claims type so EnsureAuth can parse it back.
type ClaimsFactory func(u *models.User, issuedAt, expiresAt time.Time) jwt.Claims

// Service holds auth dependencies.
type Service struct {
	cfg    Config
	users  *user.Service
	tokens TokenStore
	mail   mail.Sender
	claims ClaimsFactory
}

// New constructs the service.
func New(cfg Config, users *user.Service, m mail.Sender, claims ClaimsFactory) *Service {
	return &Service{cfg: cfg, users: users, mail: m, claims: claims}
}

// Session is what a successful login returns.
type Session struct {
	User         *models.User
	AccessToken  string
	RefreshToken string
}

// RefreshGrace is how long a just-rotated refresh token stays accepted.
const RefreshGrace = 30 * time.Second

// ErrBadCredentials is the generic login failure.
var ErrBadCredentials = &gohttp.APIError{Status: 401, Code: "invalid_credentials", Message: "email or password is incorrect"}

// ErrAccountDisabled is returned for disabled accounts.
var ErrAccountDisabled = &gohttp.APIError{Status: 403, Code: "account_disabled", Message: "this account has been disabled"}

// Register creates a password account.
func (s *Service) Register(ctx context.Context, email, name, password string) (*models.User, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, gohttp.ErrValidation(map[string]string{"password": err.Error()})
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &models.User{Email: email, Name: name, PasswordHash: hash, Preferences: models.DefaultPreferences()}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, user.ErrEmailTaken) {
			return nil, gohttp.ErrConflict("email_taken", "an account with this email already exists")
		}
		return nil, err
	}
	return u, nil
}

// Login verifies email + password.
func (s *Service) Login(ctx context.Context, email, password string) (*models.User, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// burn time so missing users are indistinguishable from bad passwords
			_ = VerifyPassword("$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", password)
			return nil, ErrBadCredentials
		}
		return nil, err
	}
	if !u.HasPassword() || !VerifyPassword(u.PasswordHash, password) {
		return nil, ErrBadCredentials
	}
	if u.Disabled {
		return nil, ErrAccountDisabled
	}
	return u, nil
}

// IssueSession mints an access token and a refresh token for u.
func (s *Service) IssueSession(ctx context.Context, u *models.User, ua, ip string) (*Session, error) {
	now := time.Now()
	access, err := SignToken(s.cfg.JWTSecret, s.claims(u, now, now.Add(s.cfg.AccessTTL)))
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.Issue(ctx, models.TokenRefresh, u.ID, s.cfg.RefreshTTL, ua, ip)
	if err != nil {
		return nil, err
	}
	return &Session{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh rotates a refresh token and returns a new session.
func (s *Service) Refresh(ctx context.Context, raw, ua, ip string) (*Session, error) {
	uid, err := s.tokens.Consume(ctx, models.TokenRefresh, raw, RefreshGrace)
	if err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			return nil, gohttp.ErrUnauthorized
		}
		return nil, err
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return nil, gohttp.ErrUnauthorized
	}
	if u.Disabled {
		return nil, ErrAccountDisabled
	}
	return s.IssueSession(ctx, u, ua, ip)
}

// Logout revokes one refresh token.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.tokens.Revoke(ctx, models.TokenRefresh, raw)
}

// ForgotPassword emails a reset link. Always succeeds to avoid user enumeration.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil
	}
	raw, err := s.tokens.Issue(ctx, models.TokenReset, u.ID, time.Hour, "", "")
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.AppURL, url.QueryEscape(raw))
	return s.mail.Send(ctx, resetMessage(u, link))
}

// resetMessage is the password-reset email; ForgotPassword (user portal) and
// EmailResetLink (admin portal) send the same wording.
func resetMessage(u *models.User, link string) mail.Message {
	return mail.Message{
		To:      u.Email,
		Subject: "Reset your ExpenseLens password",
		Body:    "Hi " + u.Name + ",\n\nUse this link to set a new password (valid for 1 hour):\n" + link + "\n\nIf you did not request this, ignore this email.",
	}
}

// ResetPassword consumes a reset token and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, raw, password string) error {
	if err := ValidatePassword(password); err != nil {
		return gohttp.ErrValidation(map[string]string{"password": err.Error()})
	}
	uid, err := s.tokens.Consume(ctx, models.TokenReset, raw, 0)
	if err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			return gohttp.ErrBadRequest("reset link is invalid or has expired")
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
	return s.tokens.RevokeAll(ctx, models.TokenRefresh, uid)
}

// ChangePassword verifies the current password (if any) and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, uid bson.ObjectID, current, next string) error {
	if err := ValidatePassword(next); err != nil {
		return gohttp.ErrValidation(map[string]string{"newPassword": err.Error()})
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return gohttp.ErrUnauthorized
	}
	if u.HasPassword() && !VerifyPassword(u.PasswordHash, current) {
		return gohttp.ErrValidation(map[string]string{"currentPassword": "is incorrect"})
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
func (s *Service) LoginWithIdentity(ctx context.Context, id Identity) (*models.User, error) {
	if u, err := s.users.FindByProvider(ctx, id.Provider, id.ProviderUserID); err == nil {
		if u.Disabled {
			return nil, ErrAccountDisabled
		}
		return u, nil
	} else if !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}
	if id.Email == "" {
		return nil, gohttp.ErrBadRequest("the provider did not share an email address")
	}
	link := models.ProviderLink{Provider: id.Provider, ProviderUserID: id.ProviderUserID}
	if u, err := s.users.FindByEmail(ctx, id.Email); err == nil {
		if !id.EmailVerified {
			return nil, gohttp.ErrConflict("email_unverified", "this email belongs to an existing account; sign in with your password first")
		}
		if u.Disabled {
			return nil, ErrAccountDisabled
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
	} else if !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}
	name := id.Name
	if name == "" {
		name = id.Email
	}
	u := &models.User{
		Email: id.Email, EmailVerified: id.EmailVerified, Name: name, AvatarURL: id.AvatarURL,
		Providers:   []models.ProviderLink{{Provider: link.Provider, ProviderUserID: link.ProviderUserID, LinkedAt: time.Now().UTC()}},
		Preferences: models.DefaultPreferences(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// EnsureActive rejects requests from users that were disabled or deleted
// after their access token was issued. Wired into the bearer middleware via
// commonmw.SetAuthGuard so handlers never check this themselves.
func (s *Service) EnsureActive(ctx context.Context, uid bson.ObjectID) error {
	disabled, err := s.users.IsDisabled(ctx, uid)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return gohttp.ErrUnauthorized
		}
		return err
	}
	if disabled {
		return ErrAccountDisabled
	}
	return nil
}

// SetDisabled flips the account flag; disabling also revokes every live
// refresh token so the session ends at the next refresh (admin portal).
func (s *Service) SetDisabled(ctx context.Context, uid bson.ObjectID, disabled bool) error {
	if err := s.users.Update(ctx, uid, bson.M{"disabled": disabled}); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return gohttp.ErrNotFound
		}
		return err
	}
	if disabled {
		return s.tokens.RevokeAll(ctx, models.TokenRefresh, uid)
	}
	return nil
}

// ResetLink issues a password-reset token for uid and returns the SPA URL
// that consumes it (admin portal hands it to the user out of band; nothing
// is emailed).
func (s *Service) ResetLink(ctx context.Context, uid bson.ObjectID) (string, time.Time, error) {
	_, link, exp, err := s.issueResetLink(ctx, uid)
	return link, exp, err
}

// EmailResetLink issues a password-reset token for uid and emails the link
// to the user with the same wording as ForgotPassword (admin portal "send"
// variant). The link itself is never returned.
func (s *Service) EmailResetLink(ctx context.Context, uid bson.ObjectID) (time.Time, error) {
	u, link, exp, err := s.issueResetLink(ctx, uid)
	if err != nil {
		return time.Time{}, err
	}
	if err := s.mail.Send(ctx, resetMessage(u, link)); err != nil {
		return time.Time{}, err
	}
	return exp, nil
}

func (s *Service) issueResetLink(ctx context.Context, uid bson.ObjectID) (*models.User, string, time.Time, error) {
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, "", time.Time{}, gohttp.ErrNotFound
		}
		return nil, "", time.Time{}, err
	}
	const ttl = time.Hour
	raw, err := s.tokens.Issue(ctx, models.TokenReset, uid, ttl, "", "")
	if err != nil {
		return nil, "", time.Time{}, err
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.AppURL, url.QueryEscape(raw))
	return u, link, time.Now().UTC().Add(ttl), nil
}

// DeleteAccount removes the user and revokes their tokens. Domain data is
// cascaded by the caller before this (see user.DeleteCascade, which also
// covers this step).
func (s *Service) DeleteAccount(ctx context.Context, uid bson.ObjectID) error {
	if err := s.tokens.RevokeAll(ctx, models.TokenRefresh, uid); err != nil {
		return err
	}
	return s.users.Delete(ctx, uid)
}

// Tokens exposes the opaque token store (admin sessions reuse it).
func (s *Service) Tokens() TokenStore { return s.tokens }
