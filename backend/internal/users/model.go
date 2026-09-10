// Package users owns the user document, its repository, and profile endpoints.
package users

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is the persisted user document.
type User struct {
	ID            bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	Email         string         `bson:"email" json:"email"`
	EmailVerified bool           `bson:"emailVerified" json:"emailVerified"`
	PasswordHash  string         `bson:"passwordHash,omitempty" json:"-"`
	Name          string         `bson:"name" json:"name"`
	AvatarURL     string         `bson:"avatarUrl,omitempty" json:"avatarUrl,omitempty"`
	Providers     []ProviderLink `bson:"providers" json:"providers"`
	Preferences   Preferences    `bson:"preferences" json:"preferences"`
	CreatedAt     time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time      `bson:"updatedAt" json:"updatedAt"`
}

// ProviderLink records a linked social login.
type ProviderLink struct {
	Provider       string    `bson:"provider" json:"provider"` // google | microsoft | apple
	ProviderUserID string    `bson:"providerUserId" json:"-"`
	LinkedAt       time.Time `bson:"linkedAt" json:"linkedAt"`
}

// Preferences are user-tunable settings.
type Preferences struct {
	Currency      string        `bson:"currency" json:"currency"`
	Language      string        `bson:"language" json:"language"`
	Theme         string        `bson:"theme" json:"theme"` // light | dark | system
	Timezone      string        `bson:"timezone" json:"timezone"`
	Notifications Notifications `bson:"notifications" json:"notifications"`
}

// Notifications are per-channel toggles.
type Notifications struct {
	BudgetAlerts      bool `bson:"budgetAlerts" json:"budgetAlerts"`
	WarrantyReminders bool `bson:"warrantyReminders" json:"warrantyReminders"`
	WeeklySummary     bool `bson:"weeklySummary" json:"weeklySummary"`
}

// DefaultPreferences returns INR / Asia-Kolkata defaults.
func DefaultPreferences() Preferences {
	return Preferences{
		Currency: "INR",
		Language: "en",
		Theme:    "system",
		Timezone: "Asia/Kolkata",
		Notifications: Notifications{
			BudgetAlerts:      true,
			WarrantyReminders: true,
			WeeklySummary:     false,
		},
	}
}

// HasPassword reports whether the user can log in with a password.
func (u *User) HasPassword() bool { return u.PasswordHash != "" }

// Public is the API-safe view of the user.
type Public struct {
	ID            string      `json:"id"`
	Email         string      `json:"email"`
	EmailVerified bool        `json:"emailVerified"`
	Name          string      `json:"name"`
	AvatarURL     string      `json:"avatarUrl,omitempty"`
	HasPassword   bool        `json:"hasPassword"`
	Providers     []string    `json:"providers"`
	Preferences   Preferences `json:"preferences"`
	CreatedAt     time.Time   `json:"createdAt"`
}

// ToPublic converts a User to its API view.
func (u *User) ToPublic() Public {
	provs := make([]string, 0, len(u.Providers))
	for _, p := range u.Providers {
		provs = append(provs, p.Provider)
	}
	return Public{
		ID:            u.ID.Hex(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		HasPassword:   u.HasPassword(),
		Providers:     provs,
		Preferences:   u.Preferences,
		CreatedAt:     u.CreatedAt,
	}
}
