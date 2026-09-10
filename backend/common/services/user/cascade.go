package user

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/storage"
	"smartreceipt/backend/common/services/budget"
	"smartreceipt/backend/common/services/category"
	"smartreceipt/backend/common/services/notification"
	"smartreceipt/backend/common/services/receipt"
	"smartreceipt/backend/common/services/warranty"
)

// DeleteCascade removes everything a user owns and then the user document:
// receipts (and their stored files via store), categories, budgets,
// warranties, notifications and every auth token (refresh + reset). It is
// shared by the user portal (DELETE /api/me) and the admin portal
// (DELETE /api/admin/users/{id}). store may be nil when files must be kept.
func DeleteCascade(ctx context.Context, uid bson.ObjectID, store storage.Provider) error {
	receipts := receipt.New()
	keys, err := receipts.AllImageKeys(ctx, uid)
	if err != nil {
		return err
	}
	if err := receipts.DeleteAllForUser(ctx, uid); err != nil {
		return err
	}
	if store != nil {
		for _, k := range keys {
			if err := store.Delete(ctx, k); err != nil {
				slog.Warn("cascade: delete file", "key", k, "err", err)
			}
		}
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
	// Tokens of every kind: the subject no longer exists, so drop the rows
	// rather than marking them used.
	if _, err := database.GetDBConnection().Collection(models.AuthTokensCollection).
		DeleteMany(ctx, bson.M{"userId": uid}); err != nil {
		return err
	}
	return New().Delete(ctx, uid)
}
