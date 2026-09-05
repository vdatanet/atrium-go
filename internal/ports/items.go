package ports

import (
	"github.com/vdatanet/atrium-go/internal/units"
)

// ScannedItem is one item a scan derived: a film, a series, a season, an
// episode, an artist, an album, a track, or one of the container rows that back
// no file at all.
//
// It lives here rather than in `internal/library` for the reason
// [003 plan §5] gives, which is 002's own decision applied rather than retaken:
// a port method returning a domain type would make the bottom of
// [architecture §2]'s diagram import a package above it. The cost that decision
// carried in 002 — a policy crossing the boundary as bytes — has no analogue
// here, because a ScannedItem is already flat values and a [units.Time].
//
// The fields are 003 plan §4.2's `items` columns, one for one, so that a row
// read back and a row about to be written are the same shape and nothing has to
// be mapped between them. The one field that is **not** a column is
// [ScannedItem.SortTitle]; see its own comment.
//
// [003 plan §5]: ../../specs/003-library-configuration-and-scanning/plan.md#5-contracts
// [architecture §2]: ../../docs/architecture.md
type ScannedItem struct {
	// ID is the 32 lowercase hexadecimal characters `library.DeriveID`
	// produced. It is derived from stable inputs and never allocated
	// (003 §3.6, Principle VII).
	ID string

	// LibraryID is the library this item belongs to, as the library's
	// allocated identity and never its root path.
	LibraryID string

	// ParentID is the container's ID, and is empty for a library's own root
	// row — the `parent_id` column is NULL there.
	ParentID string

	// Type is one of the eight `library.Kind` values, carried as a string
	// because this package may not import the domain. A second spelling of a
	// type is a second identifier for every item of that type.
	Type string

	// Name is what the path or the tags said. 004 may replace it.
	Name string

	// SortKey is what every list orders by, and it is computed once at write
	// time by `library.SortKeyFor` rather than at read time
	// (003 §3.7, plan §6.6).
	SortKey string

	// SortTitle is an explicit sort title carried by metadata, and it is the
	// **input** to 003 §3.7.3 rather than anything stored: when it is not
	// empty it replaces the derivation entirely, for every type including the
	// three that override, and [ScannedItem.SortKey] is what the store then
	// holds.
	//
	// It is empty for everything this feature produces on its own. 003 has no
	// metadata reader — 004 does — so this is the seam that feature fills, and
	// it is declared here rather than at 004 because the derivation that
	// consumes it ships now and a derivation with no way to be reached is not
	// a derivation (003 plan §8.4, AC-15).
	//
	// Empty means absent. A sort title that is empty is not a sort title, and
	// there is no case in which an item wants to sort under the empty string.
	SortTitle string

	// Path is relative to the root, **exactly as the walk read it**. It is
	// empty for an inferred container that has no directory of its own.
	//
	// It is not the normalised key the identifier was derived from, and the
	// difference is the whole point of the field: `library.Normalise` folds
	// case in a case-insensitive library, and a path stored lower-cased cannot
	// be opened on a case-sensitive filesystem at all. The key is an input to
	// `library.DeriveID` and is stored nowhere; this is what something will
	// eventually open and what an operator will read in a log line.
	//
	// *(Corrected at 003 T5, which is the first task to write the field. The
	// first wording said "in the normalised form the identifier was derived
	// from", which is true of the key and wrong of the path, and no test in
	// the packages that had been written could tell the two apart because
	// every tree in them is already lower case.)*
	Path string

	// RootOrdinal says which of the library's roots Path is relative to.
	RootOrdinal int

	// IndexNumber, ParentIndexNumber and IndexNumberEnd are the episode,
	// season, track and disc numbers, and the second number of a multi-episode
	// file. They are pointers because absent and zero are different answers:
	// season **0** is `Specials` (003 §3.4) and a season with no number at all
	// is not it.
	IndexNumber       *int
	ParentIndexNumber *int
	IndexNumberEnd    *int

	// ProductionYear is the year 003 §3.3 strips out of a name.
	ProductionYear *int

	// PremiereDate is the date a date-named episode carries (003 §3.4).
	PremiereDate *units.Time

	// Unplaceable reports that the name said too little to place the item. It
	// is counted apart from a skip because 003 §3.8 requires the two be
	// reported apart: an operator told that a file was skipped goes looking
	// for something that is not missing.
	Unplaceable bool

	// Files are the files behind the item, in ordinal order. A multi-part film
	// is one item with more than one (003 §3.3).
	Files []ScannedFile
}

// ScannedFile is one file behind a [ScannedItem], and the pair
// (item, ordinal) identifies it.
type ScannedFile struct {
	// Ordinal is the part order for a multi-part film, and 0 for everything
	// else.
	Ordinal int

	// Path is relative to the root.
	Path string

	// Size is the file's size in bytes. It is observable, because a media
	// source carries `Size` (behaviours §2.17).
	Size int64

	// ModifiedAt is when the file last changed. It is observable nowhere and
	// is stored only as half of 003 plan §6.4's change signal.
	ModifiedAt units.Time
}
