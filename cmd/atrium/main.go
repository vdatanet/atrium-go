// Command atrium serves the Jellyfin API.
//
// It is wiring and nothing else (architecture 3): everything it does is in
// internal/app, because "if something there is worth testing, it is in the
// wrong place".
//
//	atrium --data-dir /var/lib/atrium
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

	err := app.Run(ctx, os.Args[1:], os.Getenv, os.Stderr)
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
