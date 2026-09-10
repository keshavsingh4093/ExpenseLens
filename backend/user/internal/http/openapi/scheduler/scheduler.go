// Package scheduler exposes POST /openapi/scheduler/run, protected by the
// same CONFIG_RELOAD_SECRET header as config reload, so ops (and the API
// smoke) can force the daily job instead of waiting for the hourly tick.
package scheduler

import (
	"crypto/subtle"
	"net/http"
	"time"

	gohttp "smartreceipt/backend/common/pkg/http"

	appconfig "smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/http/openapi/configreload"
	"smartreceipt/backend/user/internal/services/scheduler"
)

// GetRouteSpecs returns the run route.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{{Method: "POST", Path: "/openapi/scheduler/run", Handler: handleRun, TimeOut: 600}}
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	secret := appconfig.AppConfig.ConfigReloadSecret
	if secret == "" {
		gohttp.NotFound(w, "route not found")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(configreload.HeaderName)), []byte(secret)) != 1 {
		gohttp.Unauthorized(w, "invalid secret")
		return
	}
	gohttp.Success(w, scheduler.RunDaily(r.Context(), time.Now().UTC(), true))
}
