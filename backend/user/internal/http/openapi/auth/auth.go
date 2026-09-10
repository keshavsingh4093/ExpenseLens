// Package auth exposes the session routes under /openapi/auth: register,
// login, refresh, logout, password reset and social login.
package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/validate"
	commonauth "smartreceipt/backend/common/services/auth"

	"smartreceipt/backend/user/internal/config"
	authsvc "smartreceipt/backend/user/internal/services/auth"
)

const (
	refreshCookie = "el_refresh"
	stateCookie   = "el_oauth"
)

var svc = authsvc.New()

// GetRouteSpecs returns the auth routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "POST", Path: "/openapi/auth/register", Handler: handleRegister},
		{Method: "POST", Path: "/openapi/auth/login", Handler: handleLogin},
		{Method: "POST", Path: "/openapi/auth/logout", Handler: handleLogout},
		{Method: "POST", Path: "/openapi/auth/refresh", Handler: handleRefresh},
		{Method: "POST", Path: "/openapi/auth/forgot-password", Handler: handleForgotPassword},
		{Method: "POST", Path: "/openapi/auth/reset-password", Handler: handleResetPassword},
		{Method: "GET", Path: "/openapi/auth/providers", Handler: handleProviders},
		{Method: "GET", Path: "/openapi/auth/{provider}/start", Handler: handleOAuthStart},
		{Method: "GET", Path: "/openapi/auth/{provider}/callback", Handler: handleOAuthCallback},
		{Method: "POST", Path: "/openapi/auth/{provider}/callback", Handler: handleOAuthCallback},
	}
}

type sessionResponse struct {
	AccessToken string            `json:"accessToken"`
	User        models.PublicUser `json:"user"`
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: token,
		Path: authsvc.CookiePath, HttpOnly: true, Secure: authsvc.SecureCookies(),
		SameSite: http.SameSiteLaxMode, MaxAge: int(config.AppConfig.RefreshTokenTTL.Seconds()),
		Expires: time.Now().Add(config.AppConfig.RefreshTokenTTL),
	})
}

func writeSession(w http.ResponseWriter, s *commonauth.Session) {
	setRefreshCookie(w, s.RefreshToken)
	gohttp.Success(w, sessionResponse{AccessToken: s.AccessToken, User: s.User.ToPublic()})
}

func clearRefresh(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Value: "", Path: authsvc.CookiePath, HttpOnly: true, MaxAge: -1})
}

func clientInfo(r *http.Request) (string, string) { return r.UserAgent(), r.RemoteAddr }

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email" validate:"required,email,max=254"`
		Name     string `json:"name" validate:"required,min=1,max=80"`
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
	core := svc.Core()
	u, err := core.Register(r.Context(), in.Email, strings.TrimSpace(in.Name), in.Password)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	ua, ip := clientInfo(r)
	s, err := core.IssueSession(r.Context(), u, ua, ip)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	writeSession(w, s)
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
	core := svc.Core()
	u, err := core.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	ua, ip := clientInfo(r)
	s, err := core.IssueSession(r.Context(), u, ua, ip)
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
	ua, ip := clientInfo(r)
	s, err := svc.Core().Refresh(r.Context(), c.Value, ua, ip)
	if err != nil {
		clearRefresh(w)
		gohttp.WriteError(w, r, err)
		return
	}
	writeSession(w, s)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil {
		_ = svc.Core().Logout(r.Context(), c.Value)
	}
	clearRefresh(w)
	gohttp.NoContent(w)
}

func handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := svc.Core().ForgotPassword(r.Context(), in.Email); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]string{"message": "if that email exists, a reset link has been sent"})
}

func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token" validate:"required"`
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
	if err := svc.Core().ResetPassword(r.Context(), in.Token, in.Password); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.NoContent(w)
}

func handleProviders(w http.ResponseWriter, _ *http.Request) {
	gohttp.Success(w, map[string]any{"providers": svc.ProviderNames()})
}

// handleOAuthStart sets a state+nonce cookie and redirects to the provider.
func handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	p, ok := svc.Provider(chi.URLParam(r, "provider"))
	if !ok {
		gohttp.WriteError(w, r, gohttp.ErrNotFound)
		return
	}
	state, err := commonauth.NewOpaque()
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	nonce, err := commonauth.NewOpaque()
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	// Apple posts the callback cross-site, which needs SameSite=None (https only).
	sameSite := http.SameSiteLaxMode
	if p.ResponseMode == "form_post" && authsvc.SecureCookies() {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state + "." + nonce, Path: authsvc.CookiePath,
		HttpOnly: true, Secure: authsvc.SecureCookies(), SameSite: sameSite, MaxAge: 600,
	})
	http.Redirect(w, r, p.AuthCodeURL(state, nonce), http.StatusFound)
}

// handleOAuthCallback completes the flow and redirects to the SPA, which then
// calls /openapi/auth/refresh to obtain an access token from the cookie.
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	appURL := config.AppConfig.AppURL
	fail := func(reason string, err error) {
		if err != nil {
			slog.Warn("oauth callback failed", "provider", chi.URLParam(r, "provider"), "reason", reason, "err", err)
		}
		http.Redirect(w, r, appURL+"/login?error="+url.QueryEscape(reason), http.StatusFound)
	}
	p, ok := svc.Provider(chi.URLParam(r, "provider"))
	if !ok {
		fail("unknown_provider", nil)
		return
	}
	if err := r.ParseForm(); err != nil {
		fail("bad_request", err)
		return
	}
	if e := r.Form.Get("error"); e != "" {
		fail(e, nil)
		return
	}
	c, err := r.Cookie(stateCookie)
	if err != nil {
		fail("state_missing", err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: authsvc.CookiePath, MaxAge: -1})
	state, nonce, _ := strings.Cut(c.Value, ".")
	if state == "" || r.Form.Get("state") != state {
		fail("state_mismatch", nil)
		return
	}
	id, err := p.Exchange(r.Context(), r.Form.Get("code"), nonce, r.Form.Get("user"))
	if err != nil {
		fail("exchange_failed", err)
		return
	}
	core := svc.Core()
	u, err := core.LoginWithIdentity(r.Context(), *id)
	if err != nil {
		var api *gohttp.APIError
		if errors.As(err, &api) {
			fail(api.Code, nil)
		} else {
			fail("internal", err)
		}
		return
	}
	ua, ip := clientInfo(r)
	s, err := core.IssueSession(r.Context(), u, ua, ip)
	if err != nil {
		fail("internal", err)
		return
	}
	setRefreshCookie(w, s.RefreshToken)
	http.Redirect(w, r, appURL+"/auth/callback", http.StatusFound)
}
