// Package auth wires the common auth service to the user portal's config,
// claims and social providers, and owns the account-deletion cascade.
package auth

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/mail"
	"smartreceipt/backend/common/pkg/models"
	commonauth "smartreceipt/backend/common/services/auth"
	"smartreceipt/backend/common/services/category"
	"smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/budget"
	"smartreceipt/backend/user/internal/services/notification"
	"smartreceipt/backend/user/internal/services/receipt"
	"smartreceipt/backend/user/internal/services/warranty"
)

// CookiePath is where the refresh and OAuth-state cookies are scoped.
const CookiePath = "/openapi/auth"

var providers = map[string]*commonauth.Provider{}

// InitProviders performs OIDC discovery for every configured social login.
// Failures disable that provider and are logged, never fatal.
func InitProviders(ctx context.Context) {
	base := config.AppConfig.APIURL + CookiePath
	p, errs := commonauth.NewProviders(ctx, config.AppConfig.OAuth, base)
	for _, e := range errs {
		slog.Warn("oauth provider disabled", "err", e)
	}
	providers = p
	slog.Info("oauth providers", "enabled", len(providers))
}

// Service is the user portal's auth facade.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Core builds the common auth service from the live config.
func (s *Service) Core() *commonauth.Service {
	c := config.AppConfig
	return commonauth.New(commonauth.Config{
		JWTSecret: c.JWTSecret, AccessTTL: c.AccessTokenTTL, RefreshTTL: c.RefreshTokenTTL, AppURL: c.AppURL,
	}, user.New(), mail.New(c.Mail), middleware.ClaimsFor)
}

// Provider returns a configured social login by name.
func (s *Service) Provider(name string) (*commonauth.Provider, bool) {
	p, ok := providers[name]
	return p, ok
}

// ProviderNames lists enabled social logins in display order.
func (s *Service) ProviderNames() []string {
	names := make([]string, 0, len(providers))
	for _, n := range []string{"google", "microsoft", "apple"} {
		if _, ok := providers[n]; ok {
			names = append(names, n)
		}
	}
	return names
}

// DeleteAccount cascades an account deletion across domains, then removes
// the user and their tokens.
func (s *Service) DeleteAccount(ctx context.Context, uid bson.ObjectID) error {
	if err := receipt.New().DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	if err := category.New().DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	if err := budget.New().DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	if err := warranty.New().DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	if err := notification.New().DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	return s.Core().DeleteAccount(ctx, uid)
}

// SecureCookies reports whether cookies must carry the Secure flag.
func SecureCookies() bool {
	c := config.AppConfig
	return c.IsProd() || len(c.APIURL) > 8 && c.APIURL[:8] == "https://"
}

var _ = models.TokenRefresh
