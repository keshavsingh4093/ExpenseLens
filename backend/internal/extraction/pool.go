package extraction

import (
	"context"
	"log/slog"
	"sync"
)

// Pool runs background jobs with bounded concurrency. Receipts submit an
// extraction closure per upload; a queue can replace this later without
// changing callers.
type Pool struct {
	jobs chan func(context.Context)
	wg   sync.WaitGroup
	ctx  context.Context
	stop context.CancelFunc
}

// NewPool starts n workers.
func NewPool(n int) *Pool {
	if n < 1 {
		n = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{jobs: make(chan func(context.Context), 256), ctx: ctx, stop: cancel}
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job := <-p.jobs:
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("extraction job panic", "err", r)
					}
				}()
				job(p.ctx)
			}()
		}
	}
}

// Submit enqueues a job; it never blocks the caller for long.
func (p *Pool) Submit(job func(context.Context)) {
	select {
	case p.jobs <- job:
	default:
		// queue full: run inline in a goroutine rather than drop
		go job(p.ctx)
	}
}

// Shutdown stops accepting work and waits for in-flight jobs (bounded by ctx).
func (p *Pool) Shutdown(ctx context.Context) {
	p.stop()
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
