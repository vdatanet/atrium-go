package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/vdatanet/atrium-go/internal/build"
)

// readHeaderTimeout bounds a client that opens a connection and never finishes
// sending its request head. It is not a limit on a body or on a response: 008
// streams both, and a write deadline would cut a transcode off mid-delivery.
const readHeaderTimeout = 20 * time.Second

// idleTimeout bounds a kept-alive connection nobody is using, so that a client
// that disappears does not hold a connection until the process stops.
const idleTimeout = 2 * time.Minute

// Server is this process's HTTP server.
//
// It is bound when it is built rather than when it is served, so that the
// address it actually listens on is knowable before the first request. That is
// what lets a caller ask for port 0 — a test, today; an operator who wants the
// operating system to choose, tomorrow — and still learn where to send one.
type Server struct {
	http     *http.Server
	listener net.Listener
	log      *slog.Logger
	drain    time.Duration

	// running counts the goroutines this server started. Serve returns only
	// once it is empty, which is the "waits for the process group" half of
	// architecture 5's graceful shutdown: a stop that returns while work is
	// still running is how a process leaves something behind.
	running sync.WaitGroup
}

// NewServer binds cfg.BindAddress and returns a server that hands every request
// to handler. The listener is open when this returns, so a caller that fails
// afterwards must stop the server rather than drop it.
func NewServer(cfg Config, logger *slog.Logger, handler http.Handler) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.BindAddress)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", cfg.BindAddress, err)
	}

	drain := cfg.ShutdownTimeout
	if drain <= 0 {
		drain = DefaultShutdownTimeout
	}

	return &Server{
		http: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
			// net/http logs its own failures through a log.Logger, so the
			// process would otherwise have two log formats on one stream.
			ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		listener: listener,
		log:      logger,
		drain:    drain,
	}, nil
}

// Addr is the address the server is actually listening on, which is not the
// configured one when the configured one named port 0.
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

// Serve accepts connections until ctx is cancelled, then stops accepting,
// drains the requests already in flight and returns.
//
// It returns only once every goroutine it started has returned. A drain that
// outlives cfg.ShutdownTimeout is reported rather than waited out: the point of
// a graceful stop is that it is bounded, and a stop nobody can rely on
// finishing is the shape that ends with a process killed mid-write.
func (s *Server) Serve(ctx context.Context) error {
	served := make(chan error, 1)
	s.running.Add(1)
	go func() {
		defer s.running.Done()
		served <- s.http.Serve(s.listener)
	}()

	s.log.Info("listening", "address", s.Addr().String(), "version", build.Version())

	select {
	case err := <-served:
		// Serve returned on its own, which means it failed: a stop goes
		// through the other branch.
		s.running.Wait()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	s.log.Info("stopping", "drain", s.drain)

	// ctx is already cancelled, so the drain is given a deadline of its own
	// rather than one that has already passed.
	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.drain)
	defer cancel()

	err := s.http.Shutdown(drainCtx)
	s.running.Wait()
	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	s.log.Info("stopped")
	return nil
}
