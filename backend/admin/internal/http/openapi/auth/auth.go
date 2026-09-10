// Package auth exposes admin login/refresh/logout under /openapi/admin.
package auth

import (
	"net/http"
	"time"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/admin/internal/config"
	"smartreceipt/backend/admin/internal/services/adminauth"
)

const refreshCookie = "el_admin_refresh"

var svc = adminauth.New()

// GetRouteSpecs returns the admin auth routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "POST", Path: "/openapi/admin/login", Handler: handleLogin},
		{Method: "POST", Path: "/openapi/admin/auth/refresh", Handler: handleRefresh},
		{Method: "POST", Path: "/openapi/admin/auth/logout", Handler: handleLogout},
	}
}

type sessionResponse struct {
	AccessToken string             `json:"accessToken"`
	Admin       models.PublicAdmin `json:"admin"`
}

func writeSession(w http.ResponseWriter, s *adminauth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: s.RefreshToken,
		Path: adminauth.CookiePath, HttpOnly: true, Secure: adminauth.SecureCookies(),
		SameSite: http.SameSiteLaxMode, MaxAge: int(config.AppConfig.RefreshTokenTTL.Seconds()),
		Expires: time.Now().Add(config.AppConfig.RefreshTokenTTL),
	})
	gohttp.Success(w, sessionResponse{AccessToken: s.AccessToken, Admin: s.Admin.ToPublic()})
}

func clearRefresh(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Value: "", Path: adminauth.CookiePath, HttpOnly: true, MaxAge: -1})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	a, err := svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	s, err := svc.IssueSession(r.Context(), a, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	writeSession(w, s)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookie)
	if err != nil || c.Value == "" {
		gohttp.WriteError(w, r, gohttp.ErrUnauthorized)
		return
	}
	s, err := svc.Refresh(r.Context(), c.Value, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		clearRefresh(w)
		gohttp.WriteError(w, r, err)
		return
	}
	writeSession(w, s)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil {
		_ = svc.Logout(r.Context(), c.Value)
	}
	clearRefresh(w)
	gohttp.NoContent(w)
}
