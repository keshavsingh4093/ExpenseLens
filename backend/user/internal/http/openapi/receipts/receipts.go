// Package receipts exposes POST /openapi/receipts/{id}/reextract, protected
// by the CONFIG_RELOAD_SECRET header. The admin portal cannot reach this
// service's worker pool, so its "retry extraction" action calls this route.
package receipts

import (
	"crypto/subtle"
	"net/http"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"

	appconfig "smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/http/openapi/configreload"
	"smartreceipt/backend/user/internal/services/receipt"
)

var svc = receipt.New()

// GetRouteSpecs returns the ops receipt routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "POST", Path: "/openapi/receipts/{id}/reextract", Handler: handleReextract, TimeOut: 120},
	}
}

func handleReextract(w http.ResponseWriter, r *http.Request) {
	secret := appconfig.AppConfig.ConfigReloadSecret
	if secret == "" {
		gohttp.NotFound(w, "route not found")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(configreload.HeaderName)), []byte(secret)) != 1 {
		gohttp.Unauthorized(w, "invalid secret")
		return
	}
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.ReextractByID(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Accepted(w, map[string]any{"id": rc.ID.Hex(), "status": rc.Status})
}
