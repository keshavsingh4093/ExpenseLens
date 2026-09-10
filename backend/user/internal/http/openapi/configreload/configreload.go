// Package configreload exposes POST /openapi/config/reload, protected by the
// CONFIG_RELOAD_SECRET header, so the admin portal can refresh this
// service's dynamic config after editing system_config.
package configreload

import (
	"crypto/subtle"
	"net/http"

	"smartreceipt/backend/common/pkg/config"
	gohttp "smartreceipt/backend/common/pkg/http"

	appconfig "smartreceipt/backend/user/internal/config"
)

// HeaderName carries the shared secret.
const HeaderName = "X-Config-Reload-Secret"

// GetRouteSpecs returns the reload route.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{{Method: "POST", Path: "/openapi/config/reload", Handler: handleReload}}
}

func handleReload(w http.ResponseWriter, r *http.Request) {
	secret := appconfig.AppConfig.ConfigReloadSecret
	if secret == "" {
		gohttp.NotFound(w, "route not found")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(HeaderName)), []byte(secret)) != 1 {
		gohttp.Forbidden(w, "invalid reload secret")
		return
	}
	if err := config.ReloadDynamicConfig(r.Context()); err != nil {
		gohttp.InternalError(w, "reload failed")
		return
	}
	if err := appconfig.Reload(); err != nil {
		gohttp.InternalError(w, "reload failed")
		return
	}
	gohttp.Success(w, map[string]string{"status": "reloaded"})
}
