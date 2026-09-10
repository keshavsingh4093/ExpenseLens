// Package user exposes /api/me: profile, preferences, password, deletion.
package user

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"
	usersvc "smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/http/api/reports"
	"smartreceipt/backend/user/internal/pkg/middleware"
	authsvc "smartreceipt/backend/user/internal/services/auth"
	"smartreceipt/backend/user/internal/services/receipt"
	"smartreceipt/backend/user/internal/services/report"
)

var (
	users    = usersvc.New()
	auth     = authsvc.New()
	exporter = report.New()
	receipts = receipt.New()
)

// GetRouteSpecs returns the /api/me routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/me", Handler: handleGet},
		{Method: "PATCH", Path: "/api/me", Handler: handlePatch},
		{Method: "PATCH", Path: "/api/me/preferences", Handler: handlePatchPreferences},
		{Method: "POST", Path: "/api/me/password", Handler: handleChangePassword},
		{Method: "DELETE", Path: "/api/me", Handler: handleDelete},
		{Method: "GET", Path: "/api/me/export", Handler: handleExport, TimeOut: 120},
	}
}

// handleExport downloads everything the user owns as JSON.
func handleExport(w http.ResponseWriter, r *http.Request) {
	out, err := exporter.AllData(r.Context(), middleware.UserID(r.Context()), receipts.Decorate)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	reports.Attachment(w, "expenselens-export-"+time.Now().UTC().Format("2006-01-02")+".json", "application/json", body)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	u, err := users.FindByID(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		gohttp.WriteError(w, r, gohttp.ErrUnauthorized)
		return
	}
	gohttp.Success(w, u.ToPublic())
}

func handlePatch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      *string `json:"name" validate:"omitempty,min=1,max=80"`
		AvatarURL *string `json:"avatarUrl" validate:"omitempty,url,max=2048"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	set := bson.M{}
	if in.Name != nil {
		set["name"] = strings.TrimSpace(*in.Name)
	}
	if in.AvatarURL != nil {
		set["avatarUrl"] = *in.AvatarURL
	}
	if len(set) == 0 {
		gohttp.BadRequest(w, "nothing to update")
		return
	}
	if err := users.Update(r.Context(), middleware.UserID(r.Context()), set); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	handleGet(w, r)
}

func handlePatchPreferences(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Currency      *string `json:"currency" validate:"omitempty,len=3,alpha"`
		Language      *string `json:"language" validate:"omitempty,min=2,max=8"`
		Theme         *string `json:"theme" validate:"omitempty,oneof=light dark system"`
		Timezone      *string `json:"timezone" validate:"omitempty,min=1,max=64"`
		Notifications *struct {
			BudgetAlerts      *bool `json:"budgetAlerts"`
			WarrantyReminders *bool `json:"warrantyReminders"`
			WeeklySummary     *bool `json:"weeklySummary"`
		} `json:"notifications"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	set := bson.M{}
	if in.Currency != nil {
		set["preferences.currency"] = strings.ToUpper(*in.Currency)
	}
	if in.Language != nil {
		set["preferences.language"] = *in.Language
	}
	if in.Theme != nil {
		set["preferences.theme"] = *in.Theme
	}
	if in.Timezone != nil {
		set["preferences.timezone"] = *in.Timezone
	}
	if n := in.Notifications; n != nil {
		if n.BudgetAlerts != nil {
			set["preferences.notifications.budgetAlerts"] = *n.BudgetAlerts
		}
		if n.WarrantyReminders != nil {
			set["preferences.notifications.warrantyReminders"] = *n.WarrantyReminders
		}
		if n.WeeklySummary != nil {
			set["preferences.notifications.weeklySummary"] = *n.WeeklySummary
		}
	}
	if len(set) == 0 {
		gohttp.BadRequest(w, "nothing to update")
		return
	}
	if err := users.Update(r.Context(), middleware.UserID(r.Context()), set); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	handleGet(w, r)
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword" validate:"required"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := auth.Core().ChangePassword(r.Context(), middleware.UserID(r.Context()), in.CurrentPassword, in.NewPassword); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.NoContent(w)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := auth.DeleteAccount(r.Context(), middleware.UserID(r.Context())); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.NoContent(w)
}
