package http

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

	"smartreceipt/backend/common/pkg/middleware"
)

// ServerConfig configures NewServer.
type ServerConfig struct {
	Port int
	// CORS, when set, wraps the whole handler.
	CORS *middleware.CORSConfig
	// ShutdownTimeout bounds graceful shutdown (default 10s).
	ShutdownTimeout time.Duration
}

// Server is an http.Server with graceful shutdown on SIGINT/SIGTERM.
type Server struct {
	srv      *http.Server
	shutdown time.Duration
	// OnShutdown hooks run after the listener stops (worker pools etc).
	OnShutdown []func(ctx context.Context)
}

// NewServer wraps handler with CORS (if configured) and builds the server.
func NewServer(cfg ServerConfig, handler http.Handler) *Server {
	if cfg.CORS != nil {
		handler = middleware.CORS(cfg.CORS)(handler)
	}
	t := cfg.ShutdownTimeout
	if t <= 0 {
		t = 10 * time.Second
	}
	return &Server{
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
		shutdown: t,
	}
}

// Run serves until a termination signal arrives, then shuts down gracefully.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http listening", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	slog.Info("http shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdown)
	defer cancel()
	err := s.srv.Shutdown(shutdownCtx)
	for _, f := range s.OnShutdown {
		f(shutdownCtx)
	}
	slog.Info("http stopped")
	if err != nil {
		fmt.Fprintln(os.Stderr, "shutdown:", err)
	}
	return err
}
