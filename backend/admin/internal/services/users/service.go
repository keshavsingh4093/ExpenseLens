// Package users is the admin portal's user-management logic: disable,
// enable, delete (cascade) and password-reset links.
package users

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/mail"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/storage"
	"smartreceipt/backend/common/services/auth"
	"smartreceipt/backend/common/services/receipt"
	usersvc "smartreceipt/backend/common/services/user"

	"smartreceipt/backend/admin/internal/config"
)

var store storage.Provider

// Configure installs the storage provider used to delete receipt files.
func Configure(s storage.Provider) { store = s }

// Service manages end-user accounts.
type Service struct {
	users    *usersvc.Service
	receipts *receipt.Service
}

// New returns the service.
func New() *Service { return &Service{users: usersvc.New(), receipts: receipt.New()} }

// core builds the common auth service with the admin's config. Access
// tokens are never issued from here; only the reset-token and disable paths
// are used. Mail is only used by the emailed reset-link variant.
func (s *Service) core() *auth.Service {
	c := config.AppConfig
	return auth.New(auth.Config{JWTSecret: c.JWTSecret, AppURL: c.AppURL}, s.users, mail.New(c.Mail), nil)
}

// Get loads a user with their receipt count.
func (s *Service) Get(ctx context.Context, id bson.ObjectID) (*models.User, int64, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	n, _ := s.receipts.Count(ctx, bson.M{"userId": id})
	return u, n, nil
}

// List pages users.
func (s *Service) List(ctx context.Context, f usersvc.ListFilter) ([]models.User, int64, error) {
	return s.users.List(ctx, f)
}

// SetDisabled disables or enables an account. Disabling revokes refresh
// tokens; the bearer guard rejects the current access token immediately.
func (s *Service) SetDisabled(ctx context.Context, id bson.ObjectID, disabled bool) (*models.User, error) {
	if err := s.core().SetDisabled(ctx, id, disabled); err != nil {
		return nil, mapErr(err)
	}
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return u, nil
}

// Delete removes the account and everything it owns (shared cascade).
// Returns the deleted user's email for the audit trail.
func (s *Service) Delete(ctx context.Context, id bson.ObjectID) (string, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return "", mapErr(err)
	}
	if err := usersvc.DeleteCascade(ctx, id, store); err != nil {
		return "", err
	}
	return u.Email, nil
}

// ResetLink issues a one-hour password reset link for the user.
func (s *Service) ResetLink(ctx context.Context, id bson.ObjectID) (string, time.Time, error) {
	link, exp, err := s.core().ResetLink(ctx, id)
	if err != nil {
		return "", time.Time{}, mapErr(err)
	}
	return link, exp, nil
}

// EmailResetLink issues a one-hour reset link and emails it to the user
// (same wording as the self-service forgot-password mail). The link is not
// returned to the caller.
func (s *Service) EmailResetLink(ctx context.Context, id bson.ObjectID) (time.Time, error) {
	exp, err := s.core().EmailResetLink(ctx, id)
	if err != nil {
		return time.Time{}, mapErr(err)
	}
	return exp, nil
}

func mapErr(err error) error {
	if errors.Is(err, usersvc.ErrNotFound) {
		return gohttp.ErrNotFound
	}
	return err
}
