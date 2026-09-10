package report

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/services/budget"
	"smartreceipt/backend/user/internal/services/notification"
	"smartreceipt/backend/user/internal/services/warranty"
)

// Export is the all-data JSON download (Settings → Data & Privacy).
type Export struct {
	ExportedAt    time.Time             `json:"exportedAt"`
	User          models.PublicUser     `json:"user"`
	Categories    []models.Category     `json:"categories"`
	Receipts      []models.Receipt      `json:"receipts"`
	Budgets       []models.Budget       `json:"budgets"`
	Warranties    []models.Warranty     `json:"warranties"`
	Notifications []models.Notification `json:"notifications"`
}

// AllData gathers everything the user owns. decorate fills receipt image
// URLs (the receipt service owns the storage provider).
func (s *Service) AllData(ctx context.Context, uid bson.ObjectID, decorate func(*models.Receipt) *models.Receipt) (*Export, error) {
	u, err := user.New().FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := &Export{ExportedAt: time.Now().UTC(), User: u.ToPublic()}
	if out.Categories, err = s.cats.ListForUser(ctx, uid); err != nil {
		return nil, err
	}
	if out.Receipts, err = s.receipts.ListAllForUser(ctx, uid); err != nil {
		return nil, err
	}
	for i := range out.Receipts {
		decorate(&out.Receipts[i])
	}
	if out.Budgets, err = budget.New().ListAllForUser(ctx, uid); err != nil {
		return nil, err
	}
	if out.Warranties, err = warranty.New().ListAllForUser(ctx, uid); err != nil {
		return nil, err
	}
	if out.Notifications, err = notification.New().ListAllForUser(ctx, uid); err != nil {
		return nil, err
	}
	return out, nil
}
