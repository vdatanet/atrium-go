// Command atrium serves the Jellyfin API, and manages this installation's
// accounts.
//
// It is wiring and nothing else (architecture 3): everything it does is in
// internal/app, because "if something there is worth testing, it is in the
// wrong place".
//
//	atrium --data-dir /var/lib/atrium
//	atrium user add --data-dir /var/lib/atrium --name Ada
//	atrium --help
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/vdatanet/atrium-go/internal/app"
)

func main() {
	ctx, stop := app.ShutdownContext(context.Background())
	defer stop()

	args := os.Args[1:]

	// The whole dispatch, and deliberately the whole of it: one branch on the
	// first argument (002 plan 3). Anything richer here — a subcommand table, a
	// usage line, a second word to look at — would be a branch a test wants to
	// reach in the one package no test can reach into.
	//
	// A first argument that is not a subcommand falls through to the server,
	// where ParseConfig refuses it by name ("unexpected argument"). That is the
	// regression this shape is most likely to introduce and the reason it is
	// written as a single equality rather than as "does it start with a dash":
	// `atrium --data-dir …` has to keep serving.
	var err error
	if len(args) > 0 && args[0] == app.UserCommand {
		err = app.RunUser(ctx, args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr)
	} else {
		err = app.Run(ctx, args, os.Getenv, os.Stderr)
	}

	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		// The usage text has already been written, and asking for it is not
		// a failure.
		return
	default:
		fmt.Fprintln(os.Stderr, "atrium:", err)
		os.Exit(1)
	}
}
