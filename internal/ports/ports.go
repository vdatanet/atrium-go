// Package ports holds the interfaces the domain declares and something outside
// it implements.
//
// architecture 2 puts this at the bottom of the diagram: it may import nothing
// of ours, and everything above it may import it. That direction is what lets
// ADR-0003 be argued after a feature is planned rather than before — a plan
// writes against the interface it needs, and the store that eventually arrives
// implements an interface that already existed instead of rewriting the code
// that would otherwise have named a database.
//
// So nothing here says SQL, transaction or row. An interface that leaked one
// would have decided the store for every caller, which is exactly the decision
// this package exists to keep open.
package ports

import "context"

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
// plan 5 names a third method, MarkSetupComplete, which records the instant
// setup finished. It is not here yet: its parameter is a date, dates are ticks
// project-wide (ADR-0003, behaviours 1.3), and the tick type is internal/units
// — which T4 delivers. Declaring it now would mean either inventing a second
// spelling for a tick or giving the port a plain integer, and a port that takes
// a bare number is how the unit gets lost. The column it writes exists, and the
// method lands with the type it needs.
type InstallationStore interface {
	// Installation returns this installation's name and setup state.
	Installation(ctx context.Context) (Installation, error)

	// SetServerName replaces the friendly name.
	SetServerName(ctx context.Context, name string) error
}
