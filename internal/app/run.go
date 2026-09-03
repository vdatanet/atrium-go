package app

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/vdatanet/atrium-go/internal/build"
	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/surface"
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

	// Before anything reads or writes in it. The identity file is read next
	// and the store is opened after that, and both would fail on a directory
	// that is not there with a message about their own file rather than about
	// the directory.
	if err := EnsureDataDirectory(cfg.DataDirectory); err != nil {
		return err
	}

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

	// Also before the listener: a store that cannot be opened, or a migration
	// that fails, is a refusal to start (plan 7). Opening it applies every
	// pending migration, so this is where "migrations are applied at start"
	// (plan 4) actually happens.
	//
	// Deliberately not ctx. ctx is the shutdown signal, and a start is not
	// something this process abandons half-way: a signal arriving during the
	// start would turn a clean stop into a failure to start, which is what the
	// exit status and the log would then say. The identity above takes no
	// context at all for the same reason. Open still takes one so that a caller
	// with a long migration and a real deadline has somewhere to put it.
	store, err := sqlite.Open(context.WithoutCancel(ctx), cfg.DataDirectory)
	if err != nil {
		return err
	}
	defer store.Close()

	// What a start did to the schema, at debug because on every start after
	// the first it did nothing. The versions are per half, because a rescan
	// rebuilds one of them without touching the other (ADR-0003).
	logger.Debug("store opened",
		"path", store.Path(),
		"precious-migrations-applied", store.AppliedMigrations(sqlite.Precious),
		"derived-migrations-applied", store.AppliedMigrations(sqlite.Derived),
	)

	// Also before the listener, and for the same reason: the three stages that
	// read the route table refuse a table they cannot fold, and plan 7 makes
	// that a failure to start rather than a route that quietly never matches.
	//
	// No routes are registered yet — T16-T18 write the four handlers — so
	// every path the table names is answered by the router's own refusal. The
	// pipeline is nonetheless the whole pipeline: the gate, the two headers,
	// both canonicalisers and the refusal shapes are what this binary serves.
	pipeline, err := httpapi.NewPipeline(surface.V1(), httpapi.V1QuerySpellings(), nil)
	if err != nil {
		return err
	}

	server, err := NewServer(cfg, logger, pipeline)
	if err != nil {
		return err
	}

	// The gate is shut from the moment it is built (plan 6.8), and this is the
	// point at which the start has finished: the data directory exists, the
	// identity is readable, the store is open and migrated, and the listener
	// is bound. Opening it any earlier would make the gate answer a request
	// that arrived while one of those was still in doubt, which is exactly the
	// window spec 3.5 describes; opening it later than this — after Serve,
	// which blocks until the process stops — would never happen at all.
	//
	// A request can still arrive between the bind and here, because NewServer
	// binds. It waits in the accept queue and is answered once Serve reaches
	// it, by which time the gate is open. A request that arrives while a
	// *slower* start is still running gets the 503, which is the point.
	pipeline.Gate().MarkReady()
	// No address on this line: Serve writes "listening" with it next, and the
	// interesting fact here is the state change rather than where it happened.
	logger.Info("ready")

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
