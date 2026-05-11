package runner

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Runner interface {
	Run(ctx context.Context, ready chan<- struct{}) error
}

type RunFunc func(ctx context.Context, ready chan<- struct{}) error

func (f RunFunc) Run(ctx context.Context, ready chan<- struct{}) error {
	return f(ctx, ready)
}

type Member struct {
	Name   string
	Runner Runner
}

type memberState struct {
	cancel context.CancelFunc
	done   chan error
}

// cancelAndDrain cancels members [0..upTo] in reverse order and waits for each to exit.
func cancelAndDrain(states []memberState, upTo int) {
	for j := upTo; j >= 0; j-- {
		states[j].cancel()
		<-states[j].done
	}
}

func StartOrdered(ctx context.Context, members []Member, opts ...StartOption) error {
	cfg := startConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	states := make([]memberState, len(members))
	anyDone := make(chan int, len(members))

	for i, m := range members {
		// Member contexts are intentionally derived from Background, not sigCtx, so that
		// LIFO shutdown can cancel them individually in sequence rather than all at once.
		memberCtx, cancel := context.WithCancel(context.Background()) //nolint:godre // see above
		states[i] = memberState{cancel: cancel, done: make(chan error, 1)}

		ready := make(chan struct{})
		go func(idx int, r Runner) {
			states[idx].done <- r.Run(memberCtx, ready)
			anyDone <- idx
		}(i, m.Runner)

		select {
		case <-ready:
		case idx := <-anyDone:
			cancel()
			cancelAndDrain(states, i-1)
			return fmt.Errorf("runner %q exited before ready: %w", members[idx].Name, <-states[idx].done)
		case <-sigCtx.Done():
			cancel()
			cancelAndDrain(states, i-1)
			return sigCtx.Err()
		}
	}

	if cfg.onReady != nil {
		cfg.onReady()
	}

	// Block until any runner exits or signal/context cancellation
	var exitedIdx int
	select {
	case exitedIdx = <-anyDone:
	case <-sigCtx.Done():
		// Signal received — shut down in LIFO
		var errs []error
		for i := len(members) - 1; i >= 0; i-- {
			states[i].cancel()
			if err := <-states[i].done; err != nil && !errors.Is(err, context.Canceled) {
				errs = append(errs, fmt.Errorf("%s: %w", members[i].Name, err))
			}
		}
		return errors.Join(errs...)
	}

	stop()

	// A runner exited on its own — shut down remaining in LIFO
	errs := make([]error, len(members))
	errs[exitedIdx] = <-states[exitedIdx].done

	for i := len(members) - 1; i >= 0; i-- {
		if i == exitedIdx {
			continue
		}
		states[i].cancel()
		errs[i] = <-states[i].done
	}

	var joinErrs []error
	for i, err := range errs {
		if err != nil && !errors.Is(err, context.Canceled) {
			joinErrs = append(joinErrs, fmt.Errorf("%s: %w", members[i].Name, err))
		}
	}
	return errors.Join(joinErrs...)
}

type startConfig struct {
	onReady func()
}

type StartOption func(*startConfig)

func WithOnReady(f func()) StartOption {
	return func(c *startConfig) {
		c.onReady = f
	}
}

type httpServerRunner struct {
	server *http.Server
	addr   string
	mu     sync.Mutex
}

func HTTPServer(addr string, handler http.Handler, tlsConfig *tls.Config) Runner {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return &httpServerRunner{server: srv}
}

func (r *httpServerRunner) Addr() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.addr
}

func (r *httpServerRunner) Run(ctx context.Context, ready chan<- struct{}) error {
	ln, err := net.Listen("tcp", r.server.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.server.Addr, err)
	}

	r.mu.Lock()
	r.addr = ln.Addr().String()
	r.mu.Unlock()

	var wg sync.WaitGroup
	wg.Go(func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = r.server.Shutdown(shutdownCtx)
	})

	close(ready)

	var serveErr error
	if r.server.TLSConfig != nil {
		serveErr = r.server.ServeTLS(ln, "", "")
	} else {
		serveErr = r.server.Serve(ln)
	}

	wg.Wait()

	if errors.Is(serveErr, http.ErrServerClosed) {
		return nil
	}
	return serveErr
}
