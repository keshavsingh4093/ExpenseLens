package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AdminUsersCollection stores admin-portal accounts.
const AdminUsersCollection = "admin_users"

// Admin roles. Superadmins may also delete users and manage admin accounts.
const (
	AdminRoleAdmin      = "admin"
	AdminRoleSuperadmin = "superadmin"
)

// ValidAdminRole reports whether role is one of the known roles.
func ValidAdminRole(role string) bool {
	return role == AdminRoleAdmin || role == AdminRoleSuperadmin
}

// AdminUser is an admin-portal account (separate from end users).
type AdminUser struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string        `bson:"email" json:"email"`
	Name         string        `bson:"name" json:"name"`
	PasswordHash string        `bson:"passwordHash" json:"-"`
	Role         string        `bson:"role" json:"role"` // superadmin | admin
	Disabled     bool          `bson:"disabled,omitempty" json:"disabled,omitempty"`
	LastLoginAt  *time.Time    `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time     `bson:"updatedAt" json:"updatedAt"`
}

// PublicAdmin is the API view of an admin.
type PublicAdmin struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Disabled    bool       `json:"disabled,omitempty"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ToPublic converts an AdminUser to its API view.
func (a *AdminUser) ToPublic() PublicAdmin {
	return PublicAdmin{ID: a.ID.Hex(), Email: a.Email, Name: a.Name, Role: a.Role, Disabled: a.Disabled, LastLoginAt: a.LastLoginAt, CreatedAt: a.CreatedAt}
}
