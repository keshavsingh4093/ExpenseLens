package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AdminAuditLogCollection records every mutating admin-portal action.
const AdminAuditLogCollection = "admin_audit_log"

// AdminAuditEntry is one admin action. Details is free-form, action-specific
// context (e.g. the email of a deleted user) and must never hold secrets.
type AdminAuditEntry struct {
	ID         bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	AdminID    bson.ObjectID  `bson:"adminId" json:"adminId"`
	AdminEmail string         `bson:"adminEmail" json:"adminEmail"`
	Action     string         `bson:"action" json:"action"`         // e.g. user.disable, category.create
	TargetType string         `bson:"targetType" json:"targetType"` // user | receipt | category | system_config
	TargetID   string         `bson:"targetId" json:"targetId"`
	Details    map[string]any `bson:"details,omitempty" json:"details,omitempty"`
	IP         string         `bson:"ip,omitempty" json:"ip,omitempty"`
	At         time.Time      `bson:"at" json:"at"`
}
