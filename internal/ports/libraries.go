package ports

import (
	"context"

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

// LibraryStore is what the domain needs of the store to configure a library:
// 003 plan §5's precious contract, and the only way anything in this project
// writes the two tables `0003_libraries.sql` creates.
//
// It is narrow because it is declared by its consumer. 003 serves no route, so
// every caller of this interface is `internal/app`'s `atrium library`
// subcommand (003 plan §6.7) or a scan reading which libraries there are —
// which is why there is no method here that a client's request would reach.
//
// # The refusal spec §3.6 asks for is an absence rather than an error
//
// A library's collection type, its case sensitivity and its identity are fixed
// at creation. Changing one re-derives every identifier beneath it and nothing
// stores the old ones to undo with, so the change has no undo and the
// specification refuses it.
//
// **The way it is refused is that there is nothing to call.** There is no
// `SetCollectionType`, no `SetCaseSensitive` and no method that takes a whole
// [Library] except [LibraryStore.CreateLibrary] — so a caller that wanted to
// change one does not get an error, it fails to compile. An error would be a
// refusal implemented at run time, which is a weaker thing and a different
// one: it can be reached, logged, retried and eventually worked around, and
// every one of those is a path to the rewrite that has no undo.
//
// [RenameLibrary] and [ReplaceRoots] are the two edits spec §3.6 makes free,
// and they are two verbs rather than one `SetLibrary` for the same reason:
// one verb that took a record would carry the frozen columns along with the
// editable ones, and *"the collection type is frozen"* would become a rule the
// method has to remember rather than one the signature cannot express.
type LibraryStore interface {
	// CreateLibrary writes a new library and its roots, whole.
	//
	// It takes the record rather than a name and a handful of options, which
	// is [UserStore.CreateUser]'s reason applied again: what the store writes
	// is what it was handed, so there is no second place where a default is
	// decided — including the allocated identifier, which spec §3.6 makes the
	// one identifier in this feature that is *not* derived and which is
	// therefore the domain's to allocate rather than this method's to invent.
	//
	// The roots are written in the order they appear in [Library.Roots], and
	// that position is the `ordinal` a `ScannedItem.RootOrdinal` indexes.
	//
	// A name whose fold is already taken is refused by the unique index on
	// `name_folded` rather than by a check here, so the subcommand's
	// assumption — one library per folded name — is the database's rule and
	// not a convention somebody could forget to keep.
	CreateLibrary(ctx context.Context, library Library) error

	// Libraries returns every configured library, in a stated order.
	//
	// The order is the store's to make deterministic, because architecture §2
	// forbids one that derives from anything but stable input. It matters
	// twice here: a scan of every library walks them in this order, and the
	// `CollectionFolder` rows a scan writes are what 005 will list.
	Libraries(ctx context.Context) ([]Library, error)

	// LibraryByFoldedName finds the library whose folded name is folded. It is
	// how the subcommand addresses a library, because an operator has the name
	// and never the allocated identifier.
	LibraryByFoldedName(ctx context.Context, folded string) (Library, bool, error)

	// RenameLibrary replaces a library's name and the folded spelling
	// uniqueness is enforced over.
	//
	// Both are parameters rather than one, because the fold is the domain's
	// rule and not the store's: a store that folded on the way in would be a
	// second implementation of it, and the day the two disagreed the row would
	// be unreachable by the name it was created with.
	//
	// Renaming is free — spec §3.6 makes a library's identity allocated
	// precisely so that it is — and it changes no identifier under the
	// library. Removing a library and adding another with the same name is a
	// different act with a different cost, which is why they are different
	// verbs.
	RenameLibrary(ctx context.Context, id, name, folded string) error

	// ReplaceRoots replaces a library's roots with roots, in that order.
	//
	// It replaces rather than appends, and the ordinal is the position in the
	// slice. The order decides nothing about an item's identity — spec §3.6
	// derives that from the path relative to the root — but a list that moved
	// between two reads would move every `RootOrdinal` with it, and an item
	// whose recorded root ordinal no longer names the root its path is
	// relative to cannot be opened.
	ReplaceRoots(ctx context.Context, id string, roots []string) error

	// RemoveLibrary deletes a library and the roots configured under it.
	//
	// It does not remove the items scanned under it: those live in the derived
	// half, which holds no reference into this one (architecture §6), and they
	// are removed by [ItemStore.RemoveItems] or by the next rebuild. Spec
	// §3.6's consequence is what makes this the expensive verb — a library
	// declared again with the same name and the same roots is not the same
	// library, and every item beneath it gets a new identifier.
	RemoveLibrary(ctx context.Context, id string) error
}
