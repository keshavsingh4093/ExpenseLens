// Package audit records mutating admin actions. Every admin route that
// changes state calls Log after the change succeeds; failures are logged but
// never fail the request.
package audit

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/adminaudit"

	"smartreceipt/backend/admin/internal/pkg/middleware"
)

var store = adminaudit.New()

// Target types.
const (
	TargetUser         = "user"
	TargetReceipt      = "receipt"
	TargetCategory     = "category"
	TargetSystemConfig = "system_config"
)

// Log writes one audit entry for the admin on r's context.
func Log(r *http.Request, action, targetType, targetID string, details map[string]any) {
	c, ok := middleware.GetAdminFromContext(r.Context())
	if !ok {
		return
	}
	adminID, _ := bson.ObjectIDFromHex(c.AdminID)
	e := &models.AdminAuditEntry{
		AdminID: adminID, AdminEmail: c.Email, Action: action, TargetType: targetType, TargetID: targetID,
		Details: details, IP: clientIP(r), At: time.Now().UTC(),
	}
	// Use a detached context so a cancelled request still records what it did.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Insert(ctx, e); err != nil {
		slog.Error("audit log write failed", "action", action, "target", targetID, "err", err)
	}
}

// List pages the log.
func List(ctx context.Context, f adminaudit.ListFilter) ([]models.AdminAuditEntry, int64, error) {
	return store.List(ctx, f)
}

// Actions lists distinct action names.
func Actions(ctx context.Context) ([]string, error) { return store.Actions(ctx) }

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
