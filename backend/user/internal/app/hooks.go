package app

import (
	"context"
	"strings"

	"smartreceipt/backend/common/pkg/models"

	"smartreceipt/backend/user/internal/services/budget"
	"smartreceipt/backend/user/internal/services/notification"
	"smartreceipt/backend/user/internal/services/receipt"
)

// domainHooks wires cross-domain reactions to receipt events: in-app
// notifications when extraction finishes and budget alerts on confirm.
func domainHooks() receipt.Hooks {
	notify := notification.New()
	budgets := budget.New()
	return receipt.Hooks{
		OnExtracted: func(ctx context.Context, rc *models.Receipt) {
			data := map[string]any{"receiptId": rc.ID.Hex(), "status": string(rc.Status)}
			if rc.Status == models.ReceiptFailed {
				msg := "We could not read this receipt. Open it to retry or fill in the details manually."
				if rc.Extracted != nil && rc.Extracted.Error != "" {
					msg = "Extraction failed: " + rc.Extracted.Error
				}
				notify.Notify(ctx, rc.UserID, models.NotifyExtractionFailed, "Receipt extraction failed", msg, data, "")
				return
			}
			merchant := strings.TrimSpace(rc.Merchant)
			if merchant == "" {
				merchant = "Your receipt"
			}
			body := merchant + " is ready for review"
			if rc.Total > 0 {
				body += " (" + budget.Money(rc.Total, rc.Currency) + ")"
			}
			notify.Notify(ctx, rc.UserID, models.NotifyExtractionDone, "Receipt ready for review", body+".", data, "")
		},
		OnConfirmed: func(ctx context.Context, rc *models.Receipt) {
			if len(rc.Date) >= 7 {
				budgets.CheckAlerts(ctx, rc.UserID, rc.Date[:7])
			}
		},
	}
}
