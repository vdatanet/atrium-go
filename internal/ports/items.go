package ports

import (
	"context"

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

// ScanBatch is one committed step of a scan: the items to write, and the
// renewal of the claim that says the scanner is still alive.
//
// The two travel together because 003 plan §6.9 makes them one transaction.
// The claim is renewed on every committed batch rather than by a timer beside
// the scan, which is what makes `staleAfter` defensible: it has to exceed the
// time between two batches and not the time a whole scan takes, so it is a
// number somebody can argue for rather than a guess about how big a library
// is. A renewal that committed separately from the batch would be a scanner
// that could report progress it did not make, or hold a claim over work it had
// abandoned.
type ScanBatch struct {
	// LibraryID is the library being scanned, and the row whose claim this
	// batch renews.
	LibraryID string

	// Items are the additions and updates this batch writes, ancestors before
	// what hangs from them. Removals are never batched: 003 plan §6.9 computes
	// them once from the complete reading and applies them in the final
	// transaction, so a scan that dies half way has added and updated some
	// items and removed none.
	Items []ScannedItem

	// ClaimedBy names the process holding the claim, and it is carried on the
	// batch rather than remembered by the store so that the renewal can be
	// conditional on it. A batch from a scanner whose claim has already been
	// broken and taken by another must not renew the claim it no longer holds.
	ClaimedBy string

	// At is when this batch was committed, which is what the renewed claim
	// records. It comes from [Clock] for architecture §2's reason: a store
	// that called the wall clock itself would hold a value no test could hold
	// still.
	At units.Time
}

// ItemStore is what the domain needs of the store to hold a scan's output:
// 003 plan §5's derived contract.
//
// Everything it reaches is the derived half — the half ADR-0003 lets a rescan
// drop and rebuild — and that is what [ItemStore.RebuildDerived] belongs to
// this interface rather than to a migration runner for. A forward-only lineage
// can only apply the steps after the one it recorded; it cannot express *drop
// and rebuild*, and a schema edited in place is invisible to it (003 plan §6.8).
//
// No method here names a library by anything but its identifier string. The
// derived half holds no reference into the precious one, because SQLite's
// foreign keys are on and a real constraint from a derived table to a precious
// one would refuse the drop this interface performs (architecture §6).
type ItemStore interface {
	// ItemsForLibrary returns every item recorded under a library, in a stated
	// order, with each item's files in ordinal order.
	//
	// The order is stated for a reason one layer below where Principle VII is
	// usually enforced: a scan compares this set against the one it just
	// derived, so a store answering in storage order would make a scan's
	// answer depend on the order rows happened to be inserted in.
	ItemsForLibrary(ctx context.Context, libraryID string) ([]ScannedItem, error)

	// ApplyScanBatch writes one batch and renews its claim, in one
	// transaction (003 plan §6.9).
	ApplyScanBatch(ctx context.Context, batch ScanBatch) error

	// RemoveItems deletes the items named by ids, and the file rows beneath
	// them.
	//
	// It takes identifiers rather than a library, because 003 plan §6.4
	// decides every removal in this project in one pure function and this is
	// where that decision is applied. A method that removed *by library* would
	// be the over-broad delete spec §3.8's guards exist to prevent, with the
	// guards on the other side of the call.
	RemoveItems(ctx context.Context, ids []string) error

	// ClaimScan takes the scanning claim on a library at at, breaking one
	// older than staleAfter, and reports whether it won and which claimant it
	// displaced or lost to.
	//
	// It returns a boolean rather than an error for a library already being
	// scanned, because two scanners over one store is a state this feature
	// creates on purpose — an operator may run `atrium library scan` against a
	// data directory a server is serving from (003 plan §6.7) — and
	// *"somebody else is scanning"* is an outcome the caller reports, not a
	// fault.
	//
	// The claimant travels beside the boolean because 003 plan §7 asks for two
	// messages this call is the only place that can supply: a refusal saying
	// who holds the claim, and a log line naming the process whose claim was
	// broken on age. Neither is recoverable afterwards — the row now names the
	// winner — and a caller that read the row first would be naming a claimant
	// it had not necessarily displaced. It is empty when there was none, which
	// is the first scan of a library and every scan after a rebuild.
	//
	// *(Amended at 003 T12, which wrote the implementation. The declaration
	// returned `(bool, error)` and had no way to answer the two rows of plan
	// §7 that name a claimant.)*
	ClaimScan(ctx context.Context, libraryID, by string, at units.Time, staleAfter units.Ticks) (bool, string, error)

	// ReleaseScan drops the claim and records what the scan did: when it
	// finished, whether it was a full re-examination, and 003 §3.8's summary
	// as a document.
	ReleaseScan(ctx context.Context, libraryID string, at units.Time, summary []byte, full bool) error

	// RebuildDerived drops every object of the derived schema and creates it
	// again (003 plan §6.8).
	//
	// It leaves the precious half untouched, which is ADR-0003's central claim
	// and the one thing a wrong drop destroys for good. Afterwards no library
	// has a `scan_state` row, which is the same state a library that has never
	// been scanned is in — one half, one answer.
	RebuildDerived(ctx context.Context) error
}
