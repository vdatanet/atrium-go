package ports

import (
	"github.com/vdatanet/atrium-go/internal/units"
)

// Library is a configured library: 003 plan §5's precious record, and the one
// value every resolution and every identifier in this feature is relative to.
//
// It lives here for the same reason [ScannedItem] does — a port method
// returning a domain type would invert architecture §2's arrow — and it is
// declared now, ahead of the `LibraryStore` that will read and write it,
// because `library.Resolve` takes one and a resolver cannot be written against
// a record that does not exist. The store and its interface arrive with the
// migration that backs them.
//
// # Three of these fields are frozen at creation, and that is a rule rather
// # than a convention
//
// [Library.CollectionType], [Library.CaseSensitive] and [Library.ID] decide
// every identifier under the library (003 §3.6). Changing one re-derives all of
// them, and nothing stores the old ones to undo with — so there is deliberately
// no method that writes any of the three after the library exists, and the way
// the change is refused is that there is nothing to call.
type Library struct {
	// ID is the library's own identity, allocated when it is declared and
	// kept — never derived from its name or its roots, so that renaming a
	// library or moving its roots costs nothing (003 §3.6). It is an input to
	// every identifier beneath it, which is why deleting a library and
	// declaring another with the same name is not the same library.
	ID string

	// Name is what an operator called it, and it is the name of the library's
	// own `CollectionFolder` item.
	Name string

	// NameFolded is Name lowercased, and it is what the uniqueness of a
	// library's name is enforced over: two libraries whose names differ only
	// in case are one name.
	NameFolded string

	// CollectionType is one of `library.Movies`, `library.Shows` or
	// `library.Music`, carried as a string because this package may not import
	// the domain. It selects which resolution rules apply and is not a hint
	// (003 §3.1).
	CollectionType string

	// CaseSensitive says whether two paths differing only in capitalisation
	// are two items. It defaults to false — paths are compared without regard
	// to case — and it is a property of this library rather than of the server
	// (003 §3.6, and [U-44] records that the reference's own default is
	// unmeasured).
	//
	// [U-44]: ../../docs/compatibility/reference-target.md
	CaseSensitive bool

	// Roots are the configured root directories, in ordinal order. The order
	// is what `ScannedItem.RootOrdinal` indexes, so it is stored and read back
	// rather than left to a query's default ordering.
	Roots []string

	// CreatedAt is when the library was declared.
	CreatedAt units.Time
}
