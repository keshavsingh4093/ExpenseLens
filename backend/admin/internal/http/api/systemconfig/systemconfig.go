// Package systemconfig exposes /api/admin/system-config: list, upsert,
// delete, and reload (this service + best-effort user backend).
package systemconfig

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/logger"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/validate"
	svcpkg "smartreceipt/backend/common/services/systemconfig"

	"smartreceipt/backend/admin/internal/config"
)

var svc = svcpkg.New()

var keyRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

// GetRouteSpecs returns the system config routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/system-config", Handler: handleList},
		{Method: "PUT", Path: "/api/admin/system-config/{key}", Handler: handleUpsert},
		{Method: "DELETE", Path: "/api/admin/system-config/{key}", Handler: handleDelete},
		{Method: "POST", Path: "/api/admin/system-config/reload", Handler: handleReload},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	out, err := svc.List(r.Context(), r.URL.Query().Get("category"), false)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"configs": out})
}

func handleUpsert(w http.ResponseWriter, r *http.Request) {
	key := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "key")))
	if !keyRe.MatchString(key) {
		gohttp.ValidationError(w, map[string]string{"key": "must be UPPER_SNAKE_CASE"})
		return
	}
	var in struct {
		Value       string `json:"value"`
		Description string `json:"description" validate:"max=500"`
		Category    string `json:"category" validate:"max=40"`
		Sensitive   bool   `json:"sensitive"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	// A masked value posted back unchanged must not overwrite the secret.
	if in.Sensitive && strings.HasPrefix(in.Value, "********") {
		if cur, err := svc.Get(r.Context(), key); err == nil && svcpkg.Mask(cur.Value) == in.Value {
			in.Value = cur.Value
		}
	}
	c, err := svc.Upsert(r.Context(), models.SystemConfig{
		Key: key, Value: in.Value, Description: in.Description, Category: in.Category, Sensitive: in.Sensitive,
	})
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if c.Sensitive && c.Value != "" {
		c.Value = svcpkg.Mask(c.Value)
	}
	go notifyUserBackend()
	gohttp.Success(w, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "key")))
	if err := svc.Delete(r.Context(), key); err != nil {
		if errors.Is(err, svcpkg.ErrNotFound) {
			err = gohttp.ErrNotFound
		}
		gohttp.WriteError(w, r, err)
		return
	}
	go notifyUserBackend()
	gohttp.NoContent(w)
}

func handleReload(w http.ResponseWriter, r *http.Request) {
	if err := svc.Reload(r.Context()); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	_ = config.Reload()
	userReloaded := notifyUserBackend()
	gohttp.Success(w, map[string]any{"status": "reloaded", "userBackendReloaded": userReloaded})
}

// notifyUserBackend asks the user API to refresh its dynamic config. It is
// best effort and only runs when CONFIG_RELOAD_SECRET is configured.
func notifyUserBackend() bool {
	c := config.AppConfig
	if c.ConfigReloadSecret == "" || c.UserAPIURL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.UserAPIURL+"/openapi/config/reload", nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Config-Reload-Secret", c.ConfigReloadSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn(ctx, "user backend config reload failed", "err", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
