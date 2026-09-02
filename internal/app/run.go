package app

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vdatanet/atrium-go/internal/build"
	"github.com/vdatanet/atrium-go/internal/system"
)

// Run is the whole process: configuration, logging, the server, and its stop.
// args excludes the program name, getenv reads the environment, and stderr
// receives both the usage text and the log.
//
// It returns when ctx is cancelled and the drain has finished, or with the
// first failure that stops the server coming up. flag.ErrHelp comes back as it
// is, with the usage already written, so that the caller can exit successfully
// on a request for help.
func Run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	cfg, err := ParseConfig(args, getenv, stderr)
	if err != nil {
		return err
	}

	logger := NewLogger(stderr, cfg.LogLevel)

	// Before the listener, because a refusal here is a refusal to start. An
	// identity file that cannot be read is answered by stopping and naming it
	// (plan 7): generating a fresh identity over it would make every client
	// treat this as a new server and re-authenticate, which is the same
	// failure, silently and expensively.
	installationID, err := system.InstallationID(cfg.DataDirectory)
	if err != nil {
		return err
	}

	logger.Info("starting",
		"version", build.Version(),
		"data-directory", cfg.DataDirectory,
		"installation-id", installationID,
	)

	server, err := NewServer(cfg, logger, NoRoutes())
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// ShutdownContext derives a context that is cancelled by SIGINT or SIGTERM: the
// signal a terminal sends and the one a service manager sends. The returned
// stop function releases the handler, after which a second signal terminates
// the process the way it would have the first time.
//
// This is in the entry layer rather than in main so that the reaction to a
// signal is something a test can provoke. architecture 5 puts graceful shutdown
// here for two reasons that outlive this task — a child process that outlives
// its parent accumulates until the machine dies, and the ignored-parameter
// tally is only complete at the moment the server stops.
func ShutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// NoRoutes is the handler this binary serves until the request pipeline is
// assembled. Nothing is routed yet, and a path matching no route is answered
// "404, empty body, no Content-Type" (behaviours 1.11).
//
// It is a placeholder rather than a decision: the refusal shapes belong to the
// router, are computed from the route table, and arrive with the pipeline. What
// this must not do is answer something the reference never sends — the
// standard library's own NotFoundHandler writes a text/plain body, which is a
// shape no reference server produces.
func NoRoutes() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
}
