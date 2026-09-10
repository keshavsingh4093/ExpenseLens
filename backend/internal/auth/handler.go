package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"expenselens/backend/internal/config"
	"expenselens/backend/internal/server"
	"expenselens/backend/internal/users"
)

const (
	refreshCookie = "el_refresh"
	stateCookie   = "el_oauth"
)

// Handler exposes auth routes.
type Handler struct {
	cfg       *config.Config
	svc       *Service
	users     *users.Repo
	providers map[string]*Provider
}

// NewHandler constructs the handler.
func NewHandler(cfg *config.Config, svc *Service, u *users.Repo, providers map[string]*Provider) *Handler {
	return &Handler{cfg: cfg, svc: svc, users: u, providers: providers}
}

// Mount registers routes under /auth.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.register)
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
		r.Post("/refresh", h.refresh)
		r.Post("/forgot-password", h.forgotPassword)
		r.Post("/reset-password", h.resetPassword)
		r.Get("/providers", h.listProviders)
		r.Get("/{provider}/start", h.oauthStart)
		r.Get("/{provider}/callback", h.oauthCallback)
		r.Post("/{provider}/callback", h.oauthCallback)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(h.cfg.JWTSecret))
			r.Get("/me", h.me)
			r.Post("/password", h.changePassword)
		})
	})
}

type sessionResponse struct {
	AccessToken string       `json:"accessToken"`
	User        users.Public `json:"user"`
}

func (h *Handler) writeSession(w http.ResponseWriter, s *Session) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: s.RefreshToken,
		Path: "/api/v1/auth", HttpOnly: true, Secure: h.secureCookies(),
		SameSite: http.SameSiteLaxMode, MaxAge: int(h.cfg.RefreshTokenTTL.Seconds()),
	})
	server.JSON(w, http.StatusOK, sessionResponse{AccessToken: s.AccessToken, User: s.User.ToPublic()})
}

func (h *Handler) clearRefresh(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Value: "", Path: "/api/v1/auth", HttpOnly: true, MaxAge: -1})
}

func (h *Handler) secureCookies() bool {
	return h.cfg.IsProd() || strings.HasPrefix(h.cfg.APIURL, "https://")
}

func clientInfo(r *http.Request) (string, string) { return r.UserAgent(), r.RemoteAddr }

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email" validate:"required,email,max=254"`
		Name     string `json:"name" validate:"required,min=1,max=80"`
		Password string `json:"password" validate:"required"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	u, err := h.svc.Register(r.Context(), in.Email, strings.TrimSpace(in.Name), in.Password)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	ua, ip := clientInfo(r)
	s, err := h.svc.IssueSession(r.Context(), u, ua, ip)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	h.writeSession(w, s)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	u, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	ua, ip := clientInfo(r)
	s, err := h.svc.IssueSession(r.Context(), u, ua, ip)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	h.writeSession(w, s)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookie)
	if err != nil || c.Value == "" {
		server.Error(w, r, server.ErrUnauthorized)
		return
	}
	ua, ip := clientInfo(r)
	s, err := h.svc.Refresh(r.Context(), c.Value, ua, ip)
	if err != nil {
		h.clearRefresh(w)
		server.Error(w, r, err)
		return
	}
	h.writeSession(w, s)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	h.clearRefresh(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.FindByID(r.Context(), UserID(r.Context()))
	if err != nil {
		server.Error(w, r, server.ErrUnauthorized)
		return
	}
	server.JSON(w, http.StatusOK, u.ToPublic())
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), in.Email); err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]string{"message": "if that email exists, a reset link has been sent"})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), in.Token, in.Password); err != nil {
		server.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword" validate:"required"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := h.svc.ChangePassword(r.Context(), UserID(r.Context()), in.CurrentPassword, in.NewPassword); err != nil {
		server.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listProviders(w http.ResponseWriter, _ *http.Request) {
	names := make([]string, 0, len(h.providers))
	for _, n := range []string{"google", "microsoft", "apple"} {
		if _, ok := h.providers[n]; ok {
			names = append(names, n)
		}
	}
	server.JSON(w, http.StatusOK, map[string]any{"providers": names})
}

// oauthStart sets a state+nonce cookie and redirects to the provider.
func (h *Handler) oauthStart(w http.ResponseWriter, r *http.Request) {
	p, ok := h.providers[chi.URLParam(r, "provider")]
	if !ok {
		server.Error(w, r, server.ErrNotFound)
		return
	}
	state, err := newOpaque()
	if err != nil {
		server.Error(w, r, err)
		return
	}
	nonce, err := newOpaque()
	if err != nil {
		server.Error(w, r, err)
		return
	}
	// Apple posts the callback cross-site, which needs SameSite=None (https only).
	sameSite := http.SameSiteLaxMode
	if p.ResponseMode == "form_post" && h.secureCookies() {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state + "." + nonce, Path: "/api/v1/auth",
		HttpOnly: true, Secure: h.secureCookies(), SameSite: sameSite, MaxAge: 600,
	})
	http.Redirect(w, r, p.AuthCodeURL(state, nonce), http.StatusFound)
}

// oauthCallback completes the flow and redirects to the SPA, which then calls
// /auth/refresh to obtain an access token from the cookie.
func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(reason string, err error) {
		if err != nil {
			slog.Warn("oauth callback failed", "provider", chi.URLParam(r, "provider"), "reason", reason, "err", err)
		}
		http.Redirect(w, r, h.cfg.AppURL+"/login?error="+url.QueryEscape(reason), http.StatusFound)
	}
	p, ok := h.providers[chi.URLParam(r, "provider")]
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
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: "/api/v1/auth", MaxAge: -1})
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
	u, err := h.svc.LoginWithIdentity(r.Context(), *id)
	if err != nil {
		var api *server.APIError
		if errors.As(err, &api) {
			fail(api.Code, nil)
		} else {
			fail("internal", err)
		}
		return
	}
	ua, ip := clientInfo(r)
	s, err := h.svc.IssueSession(r.Context(), u, ua, ip)
	if err != nil {
		fail("internal", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: s.RefreshToken,
		Path: "/api/v1/auth", HttpOnly: true, Secure: h.secureCookies(),
		SameSite: http.SameSiteLaxMode, MaxAge: int(h.cfg.RefreshTokenTTL.Seconds()),
		Expires: time.Now().Add(h.cfg.RefreshTokenTTL),
	})
	http.Redirect(w, r, h.cfg.AppURL+"/auth/callback", http.StatusFound)
}
