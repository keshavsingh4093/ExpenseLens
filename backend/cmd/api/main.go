// Command api runs the ExpenseLens HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"expenselens/backend/internal/auth"
	"expenselens/backend/internal/config"
	"expenselens/backend/internal/db"
	"expenselens/backend/internal/mail"
	"expenselens/backend/internal/server"
	"expenselens/backend/internal/users"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogger(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer database.Close(context.Background())
	slog.Info("mongo connected", "db", cfg.MongoDB)

	if err := database.EnsureIndexes(ctx, users.Indexes(), auth.Indexes()); err != nil {
		return err
	}

	// Dependencies
	mailer := mail.New(cfg.Mail)
	userRepo := users.NewRepo(database)
	tokenRepo := auth.NewTokenRepo(database)
	authSvc := auth.NewService(cfg, userRepo, tokenRepo, mailer)
	providers, perrs := auth.NewProviders(ctx, cfg)
	for _, e := range perrs {
		slog.Warn("oauth provider disabled", "err", e)
	}
	slog.Info("oauth providers", "enabled", len(providers))

	// Routes
	router := server.New(cfg)
	api := router.API()
	auth.NewHandler(cfg, authSvc, userRepo, providers).Mount(api)
	api.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(cfg.JWTSecret))
		users.NewHandler(userRepo, auth.UserID, authSvc).Mount(r)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("api listening", "addr", srv.Addr, "env", cfg.Env)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("api stopped")
	return nil
}

func setupLogger(cfg *config.Config) {
	var h slog.Handler
	if cfg.LogJSON {
		h = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(h))
}
