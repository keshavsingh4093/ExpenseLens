// Package admins is the admin portal's account-management logic: list,
// create, rename/re-role, disable/enable and set password for admin-portal
// accounts. Every mutation is superadmin-only (enforced by the route) and
// guarded here against locking the portal out: an admin cannot disable or
// demote themselves, and the last active superadmin cannot be disabled or
// demoted.
package admins

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/adminuser"

	"smartreceipt/backend/admin/internal/services/adminauth"
)

var (
	// ErrSelf is returned when an admin tries to disable or demote themselves.
	ErrSelf = gohttp.ErrConflict("self_target", "you cannot disable or demote your own account")
	// ErrLastSuperadmin protects the last active superadmin.
	ErrLastSuperadmin = gohttp.ErrConflict("last_superadmin", "this is the last active superadmin")
	// ErrEmailTaken maps the unique-index violation.
	ErrEmailTaken = gohttp.ErrConflict("admin_exists", "an admin with that email already exists")
)

// Service manages admin-portal accounts.
type Service struct {
	admins *adminuser.Service
	auth   *adminauth.Service
}

// New returns the service.
func New() *Service { return &Service{admins: adminuser.New(), auth: adminauth.New()} }

// List pages admins.
func (s *Service) List(ctx context.Context, f adminuser.ListFilter) ([]models.AdminUser, int64, error) {
	return s.admins.List(ctx, f)
}

// Create adds an admin account. The password is validated and hashed by
// adminauth.CreateAdmin (shared with the create-admin bootstrap script).
func (s *Service) Create(ctx context.Context, email, name, password, role string) (*models.AdminUser, error) {
	if role == "" {
		role = models.AdminRoleAdmin
	}
	a, err := s.auth.CreateAdmin(ctx, email, strings.TrimSpace(name), password, role)
	if err != nil {
		return nil, mapErr(err)
	}
	return a, nil
}

// Update renames and/or re-roles an admin. Demoting yourself or the last
// active superadmin is refused.
func (s *Service) Update(ctx context.Context, actor, id bson.ObjectID, name, role *string) (*models.AdminUser, error) {
	a, err := s.admins.FindByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	set := bson.M{}
	if name != nil && strings.TrimSpace(*name) != "" && strings.TrimSpace(*name) != a.Name {
		set["name"] = strings.TrimSpace(*name)
	}
	if role != nil && *role != a.Role {
		if !models.ValidAdminRole(*role) {
			return nil, gohttp.ErrValidation(map[string]string{"role": "must be admin or superadmin"})
		}
		if *role != models.AdminRoleSuperadmin { // demotion
			if err := s.guardSuperadmin(ctx, actor, a); err != nil {
				return nil, err
			}
		}
		set["role"] = *role
	}
	if len(set) == 0 {
		return a, nil
	}
	if err := s.admins.Update(ctx, id, set); err != nil {
		return nil, mapErr(err)
	}
	return s.admins.FindByID(ctx, id)
}

// SetDisabled disables or enables an admin. Disabling revokes every refresh
// token; the bearer guard rejects the current access token on the next
// request. Disabling yourself or the last active superadmin is refused.
func (s *Service) SetDisabled(ctx context.Context, actor, id bson.ObjectID, disabled bool) (*models.AdminUser, error) {
	a, err := s.admins.FindByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if disabled && !a.Disabled {
		if err := s.guardSuperadmin(ctx, actor, a); err != nil {
			return nil, err
		}
	}
	if a.Disabled != disabled {
		if err := s.admins.SetDisabled(ctx, id, disabled); err != nil {
			return nil, mapErr(err)
		}
	}
	if disabled {
		if err := s.auth.RevokeSessions(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.admins.FindByID(ctx, id)
}

// ResetPassword sets a new password for an admin and ends their sessions.
func (s *Service) ResetPassword(ctx context.Context, id bson.ObjectID, password string) (*models.AdminUser, error) {
	a, err := s.admins.FindByID(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := s.auth.SetPassword(ctx, id, password); err != nil {
		return nil, mapErr(err)
	}
	return a, nil
}

// guardSuperadmin refuses to remove target's superadmin standing (by disable
// or demotion) when target is the actor or the last active superadmin.
func (s *Service) guardSuperadmin(ctx context.Context, actor bson.ObjectID, target *models.AdminUser) error {
	if target.ID == actor {
		return ErrSelf
	}
	if target.Role != models.AdminRoleSuperadmin || target.Disabled {
		return nil
	}
	n, err := s.admins.CountSuperadmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastSuperadmin
	}
	return nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, adminuser.ErrNotFound):
		return gohttp.ErrNotFound
	case errors.Is(err, adminuser.ErrEmailTaken):
		return ErrEmailTaken
	}
	return err
}
