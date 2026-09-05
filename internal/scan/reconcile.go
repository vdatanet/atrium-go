package scan

import (
	"fmt"
	"sort"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// Reconciliation is what one call to [Reconcile] decided: the batch a store is
// asked to write, the identifiers it is asked to remove, and the three lists a
// scan summary reports (003 §3.8).
//
// Every list is sorted. [Reconciliation.Write] is in `library.SortItems`'s
// order — root ordinal, then path, then identifier — and the four identifier
// lists are in identifier order, which is the only order they have.
//
// **Added, Updated and Unchanged partition the desired set**, and Added plus
// Updated is exactly what Write holds. They are carried as identifiers rather
// than as a count because a summary an operator reads names what moved, and
// because a test that asserts a count of one cannot tell *which* one.
type Reconciliation struct {
	// Write is every item to insert or update, in the order a store should
	// apply it. It is the whole record and not a delta: 003 plan §4.2's
	// `items` row is small, and a partial update is a second shape for a
	// store to get wrong.
	Write []ports.ScannedItem

	// Remove is the identifiers of items that are gone. **It holds only
	// file-backed items**; see [Reconciliation.Retained].
	Remove []string

	// Added is an item the previous scan had no row for.
	Added []string

	// Updated is an item whose record or whose file signal moved, and whose
	// identifier therefore did **not** — the identifier is a function of the
	// path and the path did not move (003 plan §6.4).
	Updated []string

	// Unchanged is an item this reconciliation believed. It is empty on a
	// full re-examination of a library whose items all have files.
	Unchanged []string

	// Retained is the identifier of a container the removal pass declined to
	// remove: a previous row with no file behind it that the desired set no
	// longer holds — a series whose every episode was deleted, an album whose
	// every track was.
	//
	// It is a list rather than an absence on purpose. [behaviours §5.2] is the
	// rule *"mark file-backed items removed and leave the containers above
	// them"*, and a test that asserted only that such a row is missing from
	// [Reconciliation.Remove] would pass on a build that removed nothing at
	// all. Naming what was kept is the control.
	//
	// **A retained container is not written either.** It is not in the desired
	// set, so there is no record to write; the row a previous scan stored
	// stays exactly as it is.
	//
	// [behaviours §5.2]: ../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed
	Retained []string
}

// IdentifierMismatchError is one path whose stored identifier and whose
// freshly derived identifier disagree.
//
// It fails the library's whole scan and nothing is written (003 plan §6.4).
// The two can only differ if `library.DeriveID`'s derivation changed or if the
// library's `case_sensitive` moved, both of which are supposed to be
// impossible — which is exactly why it is worth one comparison per row to find
// out, and why the answer is an error rather than a repair. Rewriting the row
// with the new identifier is the silent discard Principle VII exists to
// prevent: every favourite and every resume position keyed on the old string
// would stop naming anything, and no scan would report it.
//
// A **rename** is not this. Identity is path-derived, so a renamed file is a
// previous row at one path and a desired item at another (003 §3.8's table),
// which is a removal and an addition and never this error.
type IdentifierMismatchError struct {
	// Root is the library root ordinal the path is relative to.
	Root int

	// Path is the root-relative path both identifiers were derived from.
	Path string

	// Stored is the identifier the previous scan wrote.
	Stored string

	// Derived is the identifier this scan computed for the same path.
	Derived string
}

func (e *IdentifierMismatchError) Error() string {
	return fmt.Sprintf(
		"scan: root %d: %q: the stored identifier %s and the derived identifier %s disagree, so this scan writes nothing",
		e.Root, e.Path, e.Stored, e.Derived)
}

// Reconcile decides what one library's scan changes, and it is where every
// removal in this project is decided.
//
// previous is what the store holds for the library; desired is what
// `library.Resolve` made of every root's reading of it. **It takes no store,
// no filesystem and no clock** (003 plan §5), so what a scan does to a library
// is decidable from two values a test can write down.
//
// # The four rows of 003 plan §6.4's table
//
//	A path with no previous row              Add; ancestors come with it
//	Size or modification time moved          Update, keeping the identifier
//	Both unchanged                           Believe it, unless full
//	A previous row with no path in the set   Remove, subject to plan §6.5
//
// *"Ancestors are created as needed"* needs no code here: desired is the whole
// library, so a new episode's series and season are already in it — new if they
// are new, and untouched if they are not.
//
// # Three things that table does not say
//
// **A record that moved is an update even when the file signal did not**, and
// this is not an optimisation of the table but a case it does not cover. A
// container has no file, so a series that was renamed or an album whose parent
// artist changed has no signal to move and would otherwise never be written.
// The same holds one level down: an album's parent is derived from its album
// artist across all of its tracks (003 §3.5), so adding one track can change a
// *sibling* album's record while every byte of its own files stands still.
// Both halves of the comparison are therefore made — the record, and the files.
//
// **The comparison excludes `ports.ScannedItem.SortTitle`**, which is the one
// field with no column behind it (003 plan §5). A row read back from a store
// carries the empty string there whatever was resolved, so comparing it would
// report every item updated on every scan the moment 004 supplies one.
//
// **The identifier is re-derived and compared, never assumed.** A desired item
// and a previous row are matched by identifier, which is the only key that
// survives a case-folding library and a path that changed case on disk. The
// **path** is matched too, and a path whose stored identifier disagrees with
// the derived one is an [IdentifierMismatchError] that fails the whole
// reconciliation.
//
// # What full changes, and what it does not
//
// full is spec §3.8's re-examination and it changes exactly one thing: whether
// an unchanged signal is believed. Every other row — what is added, what is
// updated because something moved, what is removed, what is retained — is the
// same under both. Spec §3.8 says *"the default is the fast one, the full one
// is always available"*, and a full scan that also changed a removal decision
// would make that untrue in the dangerous direction.
//
// It applies to an item that **has a file**. An item with none has no signal to
// disbelieve, and forcing every artist, series and season to be rewritten would
// make a full scan report a library's containers as updated every time.
//
// # What it does not prove
//
// Nothing here decides whether a container with nothing under it is **offered**
// to a client. This function establishes only that the row is kept, which is
// the half that makes the other half 005's — 003 plan §8.3's sixth row, and it
// is still open.
func Reconcile(previous, desired []ports.ScannedItem, full bool) (Reconciliation, error) {
	previousByID := make(map[string]ports.ScannedItem, len(previous))
	previousByPath := make(map[pathKey]ports.ScannedItem, len(previous))
	for _, item := range previous {
		previousByID[item.ID] = item
		if item.Path != "" {
			previousByPath[pathKey{Root: item.RootOrdinal, Path: item.Path}] = item
		}
	}

	present := make(map[string]struct{}, len(desired))
	for _, item := range desired {
		present[item.ID] = struct{}{}
	}

	var result Reconciliation
	for _, item := range desired {
		if item.Path != "" {
			key := pathKey{Root: item.RootOrdinal, Path: item.Path}
			if stored, ok := previousByPath[key]; ok && stored.ID != item.ID {
				return Reconciliation{}, &IdentifierMismatchError{
					Root:    item.RootOrdinal,
					Path:    item.Path,
					Stored:  stored.ID,
					Derived: item.ID,
				}
			}
		}

		stored, ok := previousByID[item.ID]
		switch {
		case !ok:
			result.Added = append(result.Added, item.ID)
			result.Write = append(result.Write, item)
		case recordMoved(stored, item), signalMoved(stored.Files, item.Files), full && len(item.Files) > 0:
			result.Updated = append(result.Updated, item.ID)
			result.Write = append(result.Write, item)
		default:
			result.Unchanged = append(result.Unchanged, item.ID)
		}
	}

	for _, item := range previous {
		if _, ok := present[item.ID]; ok {
			continue
		}
		if len(item.Files) == 0 {
			// behaviours §5.2, and plan §6.5's closing paragraph: none of the
			// three guards watches a directory that emptied inside a healthy
			// root, so the container is kept rather than judged gone for good.
			result.Retained = append(result.Retained, item.ID)
			continue
		}
		result.Remove = append(result.Remove, item.ID)
	}

	library.SortItems(result.Write)
	sortIdentifiers(result.Remove)
	sortIdentifiers(result.Added)
	sortIdentifiers(result.Updated)
	sortIdentifiers(result.Unchanged)
	sortIdentifiers(result.Retained)
	return result, nil
}

// pathKey is a path within a library: the root ordinal and the root-relative
// path. Two roots of one library can both hold `The Matrix (1999).mkv` and they
// are two items (003 §3.6 keys on the root ordinal), so the ordinal is half the
// key rather than a field beside it.
type pathKey struct {
	Root int
	Path string
}

// recordMoved reports whether anything a store keeps about an item differs
// between the row it holds and the item this scan resolved.
//
// The fields are 003 plan §4.2's `items` columns, one for one and named rather
// than compared reflectively, so that a column added to the record without a
// line here is a review comment rather than a silent behaviour. `SortTitle` is
// deliberately absent — see [Reconcile].
//
// [ports.ScannedItem.Files] is not compared here either: the files are the
// change *signal* and [signalMoved] is where they are read, because 003 plan
// §6.4 states the signal as `(size, modification time)` per file and a
// comparison that folded the two together could not be varied independently.
func recordMoved(previous, desired ports.ScannedItem) bool {
	switch {
	case previous.LibraryID != desired.LibraryID,
		previous.ParentID != desired.ParentID,
		previous.Type != desired.Type,
		previous.Name != desired.Name,
		previous.SortKey != desired.SortKey,
		previous.Path != desired.Path,
		previous.RootOrdinal != desired.RootOrdinal,
		previous.Unplaceable != desired.Unplaceable,
		!sameNumber(previous.IndexNumber, desired.IndexNumber),
		!sameNumber(previous.ParentIndexNumber, desired.ParentIndexNumber),
		!sameNumber(previous.IndexNumberEnd, desired.IndexNumberEnd),
		!sameNumber(previous.ProductionYear, desired.ProductionYear),
		!sameDate(previous.PremiereDate, desired.PremiereDate):
		return true
	}
	return false
}

// signalMoved reports whether 003 plan §6.4's change signal moved.
//
// The signal is `(size, modification time)` **per file**, and the two halves
// are read apart because the failures they catch are different: a re-encode
// keeps the length and moves the time, and a restore from a backup keeps the
// time and moves the length. A comparison reading only one of the two passes
// every test that varies both.
//
// The ordinal and the path are compared as well, because a multi-part film's
// parts are one item's files and a part that was replaced by a differently
// named file of the same length at the same instant is a change to the item
// (003 §3.3).
//
// **Both modification times have been through `units.At`**, which rounds to a
// whole tick. That is the property this comparison rests on: a filesystem may
// report a resolution finer or coarser than a tick, and comparing a freshly
// stated instant against a stored tick would report every file changed on the
// first rescan of every installation on such a filesystem. The walk converts
// once, on the way in, and a store keeps the converted value.
//
// The comparison is [units.Time.Equal] and not `==`. Measured at T9, the two
// agree on every value this project can build — `units.At` strips the monotonic
// reading and sets the location to UTC, so two Times naming one instant have
// one representation, and the mutation that swaps `==` in survives the whole
// suite. It is Equal anyway because that equivalence is a property of the
// constructor rather than of this comparison, and a `time.Time` compared with
// `==` is a comparison of representations that is wrong the day one arrives by
// another road.
func signalMoved(previous, desired []ports.ScannedFile) bool {
	if len(previous) != len(desired) {
		return true
	}
	for i := range previous {
		switch {
		case previous[i].Ordinal != desired[i].Ordinal,
			previous[i].Path != desired[i].Path,
			previous[i].Size != desired[i].Size,
			!previous[i].ModifiedAt.Equal(desired[i].ModifiedAt):
			return true
		}
	}
	return false
}

// sameNumber compares two optional integers, where absent and zero are
// different answers: season 0 is `Specials` and a season with no number at all
// is not it (003 §3.4).
func sameNumber(previous, desired *int) bool {
	if previous == nil || desired == nil {
		return previous == nil && desired == nil
	}
	return *previous == *desired
}

// sameDate compares two optional dates. It uses [units.Time.Equal] rather than
// `==`, because two Times that serialise to the same bytes are the same instant
// and a struct comparison is a comparison of representations.
func sameDate(previous, desired *units.Time) bool {
	if previous == nil || desired == nil {
		return previous == nil && desired == nil
	}
	return previous.Equal(*desired)
}

// sortIdentifiers puts a list of identifiers in the one order they have.
//
// Principle VII, at the layer the plan is produced rather than at the layer it
// is applied: two runs over the same two sets must hand a store the same lists,
// and both loops above walk their input in the caller's order.
func sortIdentifiers(ids []string) {
	sort.Strings(ids)
}
