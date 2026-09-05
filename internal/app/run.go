package app

import (
	"context"
	"io"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/vdatanet/atrium-go/internal/build"
	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/system"
	"github.com/vdatanet/atrium-go/internal/users"
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
	// the first it did nothing. The two halves are reported differently
	// because 003 made them different things: the precious half applied a
	// tail of migrations, and the derived half was either at this build's
	// generation or was dropped and recreated whole (ADR-0003, 003 plan 6.8).
	//
	// derived-rebuilt is the one line here that is not only an observation. A
	// rebuild leaves every library with no items, so a true here is a full
	// scan of every library owed — enqueued after the server begins serving,
	// never inside the start.
	logger.Debug("store opened",
		"path", store.Path(),
		"precious-migrations-applied", store.AppliedMigrations(sqlite.Precious),
		"derived-rebuilt", store.DerivedRebuilt(),
	)

	// listeningPort is what /System/Info reports as WebSocketPortNumber, and
	// it is filled in below, once the server has bound.
	//
	// The handlers are built before the pipeline, the pipeline before the
	// server, and the server is what binds — so the port cannot be a value
	// here, and with --bind-address naming port 0 it is not even a value this
	// process has chosen. It is atomic because the goroutines that read it are
	// the ones serving requests; nothing serves before Serve, so the store
	// below already happens before every read, and the atomic says so rather
	// than leaving a reader to work it out.
	var listeningPort atomic.Int64

	// The handlers, before the pipeline that registers them. The address
	// configuration is the zero value because 001 gives an operator no way to
	// set any of it — no published URL, no derive flag, no bound-address list —
	// and system.LocalAddress then answers from the request itself, which
	// plan 6.6 states as the deliberate answer for an installation with none of
	// the three. The flags that fill it in are the feature that adds them.
	//
	// The clock and the authenticator, which three of the four handlers below
	// share. One authenticator rather than one per handler: 002 plan 5 makes
	// resolving a credential one question with one answer, and two readers of
	// one credential can disagree.
	clock := SystemClock()
	authenticator, err := httpapi.NewTokenAuthenticator(httpapi.TokenAuthenticatorConfig{
		Sessions: store,
		Accounts: store,
		Clock:    clock,
	})
	if err != nil {
		return err
	}

	// ~~No Authenticator, and that is the honest value rather than a gap: this
	// build has issued no token and knows no user~~ **002 filled it.** The port
	// 001 declared is the real one now, so /System/Info answers 200 to a token
	// this server issued and 401 to a request carrying none, once setup is
	// complete (001 spec 3.2, 002 AC-14).
	systemHandler, err := httpapi.NewSystemHandler(httpapi.SystemHandlerConfig{
		InstallationID: installationID,
		Installations:  store,
		Addresses:      system.AddressConfig{},
		Paths:          system.PathsFor(cfg.DataDirectory),
		HTTPPort:       func() int { return int(listeningPort.Load()) },
		Authenticator:  authenticator,
	})
	if err != nil {
		return err
	}

	usersHandler, err := httpapi.NewUsersHandler(httpapi.UsersHandlerConfig{
		InstallationID: installationID,
		Login:          users.NewLogin(store, clock),
		Accounts:       store,
		Sessions:       store,
		Clock:          clock,
		Authenticator:  authenticator,
	})
	if err != nil {
		return err
	}

	sessionsHandler, err := httpapi.NewSessionsHandler(httpapi.SessionsHandlerConfig{
		Sessions:      store,
		Accounts:      store,
		Authenticator: authenticator,
		Clock:         clock,
	})
	if err != nil {
		return err
	}

	routes, err := httpapi.Routes(surface.V1(), httpapi.Handlers{
		System:   systemHandler,
		Users:    usersHandler,
		Sessions: sessionsHandler,
	})
	if err != nil {
		return err
	}

	// Also before the listener, and for the same reason: the three stages that
	// read the route table refuse a table they cannot fold, and plan 7 makes
	// that a failure to start rather than a route that quietly never matches.
	//
	// ~~All four rows of 001~~ **All eleven rows of 001 and 002** are
	// registered now, so the paths this server answers are exactly the ones
	// surface.yaml gives those two features; every other path the table names
	// is answered by the router's own refusal. The
	// pipeline is the whole pipeline: the gate, the two headers, both
	// canonicalisers and the refusal shapes are what this binary serves.
	pipeline, err := httpapi.NewPipeline(surface.V1(), httpapi.V1QuerySpellings(), routes)
	if err != nil {
		return err
	}

	server, err := NewServer(cfg, logger, pipeline)
	if err != nil {
		return err
	}

	// Now that there is a listener, the port /System/Info reports is knowable.
	// Before the gate opens, so that no request can be answered with the zero
	// this started as.
	if address, ok := server.Addr().(*net.TCPAddr); ok {
		listeningPort.Store(int64(address.Port))
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

	// After the gate opens, and never inside the start: a full scan of every
	// library would hold the gate shut for minutes on the one start that most
	// needs to come up (003 plan §6.8, and sqlite's own ensureDerivedGeneration
	// says it is leaving this here). The deferred wait is registered *after*
	// the store's deferred close, so it runs before it — a scan still writing
	// must not meet a closed database.
	scans := startScheduledScans(ctx, store, logger, cfg.ScanInterval, store.DerivedRebuilt())
	defer scans.wait()

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
