package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// SystemConfigCollection holds dynamic configuration keys.
const SystemConfigCollection = "system_config"

// SystemConfig is one dynamic key/value. Sensitive values are masked in
// admin listings.
type SystemConfig struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Key         string        `bson:"key" json:"key"`
	Value       string        `bson:"value" json:"value"`
	Description string        `bson:"description,omitempty" json:"description"`
	Category    string        `bson:"category,omitempty" json:"category"`
	Sensitive   bool          `bson:"sensitive" json:"sensitive"`
	CreatedAt   time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt" json:"updatedAt"`
}
