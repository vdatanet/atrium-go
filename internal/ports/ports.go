// Package ports holds the interfaces the domain declares and something outside
// it implements.
//
// architecture 2 puts this at the bottom of the diagram: it may import nothing
// of ours but internal/units, and everything above it may import it. The unit
// types are the exception because architecture 2 makes them one — "a leaf,
// imported by both" — and plan 5 writes this package's contracts in terms of
// them. That direction is what lets
// ADR-0003 be argued after a feature is planned rather than before — a plan
// writes against the interface it needs, and the store that eventually arrives
// implements an interface that already existed instead of rewriting the code
// that would otherwise have named a database.
//
// So nothing here says SQL, transaction or row. An interface that leaked one
// would have decided the store for every caller, which is exactly the decision
// this package exists to keep open.
package ports

import (
	"context"

	"github.com/vdatanet/atrium-go/internal/units"
)

// Installation is what the domain knows about this installation, and it is
// deliberately not what the store holds.
//
// The store records when setup was completed; spec 4 makes the observable
// StartupWizardCompleted, a boolean, and plan 4 derives it as
// "setup_completed_at IS NOT NULL". Handing the domain the instant instead
// would put a value on this side of the boundary that no response carries and
// nothing decides.
//
// The identity is absent on purpose. It is a file beside the store rather than
// a row in it, because AC-4 asks that it survive "a rebuild of the store from
// empty" — plan 4 argues it, and internal/system reads it.
type Installation struct {
	// Name is the operator-chosen friendly name, reported as ServerName.
	// A fresh installation carries "atrium" (spec 3.1).
	Name string

	// SetupCompleted reports whether initial configuration is finished. It is
	// what StartupWizardCompleted answers (spec 3.1, spec 4).
	SetupCompleted bool
}

// InstallationStore is what the domain needs of the store to answer the two
// /System/Info responses and to let an operator rename the server (plan 5).
//
// It is narrow because it is declared by its consumer: the store implements
// this, not the other way round, so a method nothing calls is a method nothing
// should have to implement.
//
// It takes units.Time, which is the one import in this package that is not the
// standard library. architecture 2's table says a port may import "nothing of
// ours", and its own prose says why the unit types are the exception: they are
// "a leaf, imported by both", because behaviours 1.3 puts ticks in storage as
// well as on the wire "so no conversion can be forgotten at a boundary". A port
// that took a bare integer would be exactly that forgotten conversion, and
// plan 5 writes the contract with the type rather than the number.
type InstallationStore interface {
	// Installation returns this installation's name and setup state.
	Installation(ctx context.Context) (Installation, error)

	// SetServerName replaces the friendly name.
	SetServerName(ctx context.Context, name string) error

	// MarkSetupComplete records the instant initial configuration finished,
	// which is what makes StartupWizardCompleted true (spec 4, plan 4).
	//
	// It takes the instant rather than deriving one, because architecture 2
	// makes the wall clock a port: a store that called time.Now would put a
	// value in a golden body that no test could hold still.
	MarkSetupComplete(ctx context.Context, at units.Time) error
}

// Clock is the wall clock, as a port.
//
// 001's plan 5 declared it and 001 never wrote it, because 001 had no caller:
// nothing it served recorded an instant. 002 is the first — provisioning the
// first account records when setup completed (002 plan 6.8), a login stamps
// last_login_at, and a session carries three dates of its own.
//
// It is an interface rather than a call to time.Now for the reason
// architecture 2 gives: "a clock the tests replace is what keeps a golden body
// stable between two runs". Every date this server sends is compared byte for
// byte at L3, and wall-clock is one of the allowlist's four declared derivation
// classes — so the differences a clock creates are enumerated, and a value that
// moved because a test could not hold it still would be a difference nobody
// declared.
//
// It returns units.Time and not time.Time, which is the same rule
// InstallationStore's MarkSetupComplete follows: a units.Time is already UTC
// and already rounded to a whole tick, so no caller can hold an instant the
// wire cannot spell (behaviours 1.2, 1.3).
type Clock interface {
	// Now is this instant.
	Now() units.Time
}
