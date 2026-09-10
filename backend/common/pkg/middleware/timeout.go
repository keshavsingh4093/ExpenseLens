package middleware

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// guardedWriter serialises writes so a timed-out request and the still
// running handler cannot both write to the response.
type guardedWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	wroteHeader bool
	timedOut    bool
	discard     http.Header
}

func (g *guardedWriter) Header() http.Header {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.timedOut {
		if g.discard == nil {
			g.discard = http.Header{}
		}
		return g.discard
	}
	return g.ResponseWriter.Header()
}

func (g *guardedWriter) WriteHeader(code int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.timedOut || g.wroteHeader {
		return
	}
	g.wroteHeader = true
	g.ResponseWriter.WriteHeader(code)
}

func (g *guardedWriter) Write(b []byte) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.timedOut {
		return len(b), nil
	}
	if !g.wroteHeader {
		g.wroteHeader = true
		g.ResponseWriter.WriteHeader(http.StatusOK)
	}
	return g.ResponseWriter.Write(b)
}

func (g *guardedWriter) writeTimeout() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.timedOut = true
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	g.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	g.ResponseWriter.WriteHeader(http.StatusRequestTimeout)
	_ = json.NewEncoder(g.ResponseWriter).Encode(map[string]any{"success": false, "error": "request timeout", "code": "timeout"})
}

func (g *guardedWriter) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *guardedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := g.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}

// WithTimeout bounds a request to max seconds. On timeout the client gets a
// 408 envelope and the handler's context is cancelled.
func WithTimeout(maxSeconds int64) func(http.Handler) http.Handler {
	if maxSeconds <= 0 {
		maxSeconds = 30
	}
	timeout := time.Duration(maxSeconds) * time.Second
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			gw := &guardedWriter{ResponseWriter: w}
			done := make(chan struct{})
			panicCh := make(chan any, 1)
			go func() {
				defer func() {
					if rec := recover(); rec != nil {
						panicCh <- rec
					}
					close(done)
				}()
				next.ServeHTTP(gw, r.WithContext(ctx))
			}()
			select {
			case <-done:
				select {
				case rec := <-panicCh:
					panic(rec) // re-raise on the request goroutine so RecoveryAndClean handles it
				default:
				}
			case <-ctx.Done():
				gw.writeTimeout()
			}
		})
	}
}
