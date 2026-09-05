package scan

// The reconciliation's tests sit at the `scan` level of 003 tasks' three: a Go
// test beside the package, asserting about the function that produced the
// answer and nothing downstream of it.
//
// **Both sides of almost every case here are built by `library.Resolve`**
// rather than written out as literals. Reconcile compares whole records, so a
// hand-written `previous` row is a row whose fields agree with the resolver only
// as far as the person writing the test remembered — and a test in which every
// record differs from the resolver's would report every item updated and pass
// three of the four rows of the table for the wrong reason. Resolving both sides
// from readings makes the *entries* the only thing a case varies, which is what
// each case is about.
//
// **What none of these proves**: anything about what was stored, and anything a
// client would receive. Most sharply, 003 plan §8.3's sixth row —
// `TestTheRemovalPassMarksFileBackedItemsRemovedAndLeavesTheContainersAboveThem`
// establishes that an emptied series keeps its row and **nothing** about whether
// a client is offered it. That half is 005's entirely and this feature cannot
// reach it.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// aMoviesLibrary and aShowsLibrary are the two configured libraries these tests
// resolve against. The identity is fixed rather than generated because every
// identifier below is derived from it and a failure message naming a different
// digest on every run is unreadable.
func aMoviesLibrary() ports.Library {
	return ports.Library{
		ID:             "0123456789abcdef0123456789abcdef",
		Name:           "Movies",
		NameFolded:     "movies",
		CollectionType: string(library.Movies),
	}
}

func aShowsLibrary() ports.Library {
	return ports.Library{
		ID:             "fedcba9876543210fedcba9876543210",
		Name:           "Shows",
		NameFolded:     "shows",
		CollectionType: string(library.Shows),
	}
}

// anInstant is a fixed modification time, so that a case that does not vary the
// time does not vary it by accident.
func anInstant() units.Time {
	return units.At(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
}

// entry is one candidate file, with a size and a modification time a case can
// vary one at a time.
func entry(path string, size int64, at units.Time) library.Entry {
	return library.Entry{Path: path, Size: size, ModifiedAt: at}
}

// at is entry with the fixed instant, for the cases that vary neither half of
// the signal.
func at(path string, size int64) library.Entry {
	return entry(path, size, anInstant())
}

// resolved is what a scan of these entries would hand Reconcile.
func resolved(t *testing.T, lib ports.Library, entries ...library.Entry) []ports.ScannedItem {
	t.Helper()
	plan, err := library.Resolve(lib, []library.Reading{{Root: 0, Entries: entries}})
	if err != nil {
		t.Fatalf("resolving %d entries: %v", len(entries), err)
	}
	return plan.Items
}

// idOf finds the one item at path and returns its identifier, failing the test
// when there is not exactly one. A test that names an identifier by digest is
// unreadable; a test that names it by path says what it means.
func idOf(t *testing.T, items []ports.ScannedItem, path string) string {
	t.Helper()
	var found []string
	for _, item := range items {
		if item.Path == path {
			found = append(found, item.ID)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one item at %q, found %d", path, len(found))
	}
	return found[0]
}

// idOfNamed finds the one item called name, for the container rows that have no
// path of their own.
func idOfNamed(t *testing.T, items []ports.ScannedItem, kind library.Kind, name string) string {
	t.Helper()
	var found []string
	for _, item := range items {
		if item.Type == string(kind) && item.Name == name {
			found = append(found, item.ID)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one %s called %q, found %d", kind, name, len(found))
	}
	return found[0]
}

// writtenIDs is the identifiers of a batch, in the order the batch holds them.
func writtenIDs(items []ports.ScannedItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// sortedIDs is a list of identifiers in the order Reconcile returns them, so a
// test can state what it expects in the order it likes.
func sortedIDs(ids ...string) []string {
	out := slices.Clone(ids)
	slices.Sort(out)
	if out == nil {
		return []string{}
	}
	return out
}

// orEmpty is what a list looks like when it is empty, so that a comparison
// against a literal does not have to care whether a nil slice or an empty one
// came back.
func orEmpty(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

func assertIDs(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(orEmpty(got), orEmpty(want)) {
		t.Errorf("%s: got %v, want %v", what, orEmpty(got), orEmpty(want))
	}
}

// --- Row 1 of plan §6.4's table: a path with no previous row -----------------

func TestAPathWithNoPreviousRowIsAddedAndItsAncestorsComeWithIt(t *testing.T) {
	lib := aShowsLibrary()
	before := resolved(t, lib, at("The Series/Season 01/The Series - S01E01 - Pilot.mkv", 700))
	after := resolved(t, lib,
		at("The Series/Season 01/The Series - S01E01 - Pilot.mkv", 700),
		at("Another Series/Season 01/Another Series - S01E01 - Start.mkv", 800),
	)

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	// The episode is the file that appeared. The series and the season above it
	// have no file at all, so a reconciliation that only added what it had a
	// path for would write an episode whose parent does not exist.
	assertIDs(t, "added", result.Added, sortedIDs(
		idOf(t, after, "Another Series/Season 01/Another Series - S01E01 - Start.mkv"),
		idOf(t, after, "Another Series"),
		idOf(t, after, "Another Series/Season 01"),
	))
	assertIDs(t, "the batch", sortedIDs(writtenIDs(result.Write)...), result.Added)
	assertIDs(t, "removed", result.Remove, nil)
	assertIDs(t, "retained", result.Retained, nil)

	// And the control: everything that was already there is believed. Without
	// it, a build that wrote the whole desired set on every scan would pass the
	// assertions above.
	if len(result.Unchanged) != len(before) {
		t.Errorf("unchanged: got %d identifiers, want the %d the previous scan held", len(result.Unchanged), len(before))
	}
}

func TestTheBatchIsOrderedSoThatAnAncestorIsWrittenBeforeWhatHangsFromIt(t *testing.T) {
	lib := aShowsLibrary()
	after := resolved(t, lib, at("The Series/Season 01/The Series - S01E01 - Pilot.mkv", 700))

	result, err := Reconcile(nil, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	position := map[string]int{}
	for i, item := range result.Write {
		position[item.ID] = i
	}
	for _, item := range result.Write {
		if item.ParentID == "" {
			continue
		}
		parent, ok := position[item.ParentID]
		if !ok {
			t.Fatalf("%s (%s) names a parent that is not in the batch", item.Name, item.Type)
		}
		if parent > position[item.ID] {
			t.Errorf("%s (%s) is written at %d, before its parent at %d",
				item.Name, item.Type, position[item.ID], parent)
		}
	}
}

// --- Row 2: size and modification time, varied one at a time -----------------

// TestOnlyTheSizeMovingIsAnUpdateThatKeepsTheIdentifier is the restore that kept
// the time: a file put back from a backup with its recorded time of change
// restored alongside it, and a different number of bytes in it.
//
// It varies the size and **nothing else**, because a build reading only the
// modification time passes every case that varies both.
//
// It is also where spec §3.2's last exclusion lands — *"files being written,
// detected by size change between two passes"*. v1 does not refuse such a file:
// the size that moved is an **update** carrying the new number, and the
// assertions below say so by naming what is *not* in the answer as well as what
// is. A build that implemented the row as an exclusion would remove the item
// while the copy ran.
func TestOnlyTheSizeMovingIsAnUpdateThatKeepsTheIdentifier(t *testing.T) {
	lib := aMoviesLibrary()
	const path = "The Matrix (1999).mkv"
	before := resolved(t, lib, entry(path, 700, anInstant()))
	after := resolved(t, lib, entry(path, 900, anInstant()))

	// The control on the case itself: exactly one of the two halves moved.
	beforeFile := fileOf(t, before, path)
	afterFile := fileOf(t, after, path)
	if beforeFile.Size == afterFile.Size {
		t.Fatal("this case is supposed to move the size and it did not")
	}
	if !beforeFile.ModifiedAt.Equal(afterFile.ModifiedAt) {
		t.Fatal("this case is supposed to leave the modification time alone and it did not")
	}

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	id := idOf(t, after, path)
	assertIDs(t, "updated", result.Updated, []string{id})
	assertIDs(t, "added", result.Added, nil)
	assertIDs(t, "removed", result.Remove, nil)
	if got := idOf(t, before, path); got != id {
		t.Errorf("the identifier moved: %s became %s", got, id)
	}
	if got := fileOf(t, result.Write, path).Size; got != 900 {
		t.Errorf("the batch carries size %d, want the new 900 — a media source's Size travels (behaviours §2.17)", got)
	}
	// Spec §3.2's row, as what v1 does with it: the file is not refused and the
	// item is not withdrawn while it is being written.
	if len(result.Unchanged) != len(after)-1 {
		t.Errorf("unchanged: got %d identifiers, want the %d rows that are not the film", len(result.Unchanged), len(after)-1)
	}
	assertIDs(t, "retained", result.Retained, nil)
}

// TestOnlyTheModificationTimeMovingIsAnUpdateThatKeepsTheIdentifier is the
// re-encode that kept the length. It varies the modification time and nothing
// else, because a build reading only the size passes every case that varies
// both — and the failure it hides is not the one above.
func TestOnlyTheModificationTimeMovingIsAnUpdateThatKeepsTheIdentifier(t *testing.T) {
	lib := aMoviesLibrary()
	const path = "The Matrix (1999).mkv"
	later := units.At(anInstant().Instant().Add(time.Hour))
	before := resolved(t, lib, entry(path, 700, anInstant()))
	after := resolved(t, lib, entry(path, 700, later))

	beforeFile := fileOf(t, before, path)
	afterFile := fileOf(t, after, path)
	if beforeFile.Size != afterFile.Size {
		t.Fatal("this case is supposed to leave the size alone and it did not")
	}
	if beforeFile.ModifiedAt.Equal(afterFile.ModifiedAt) {
		t.Fatal("this case is supposed to move the modification time and it did not")
	}

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	id := idOf(t, after, path)
	assertIDs(t, "updated", result.Updated, []string{id})
	assertIDs(t, "added", result.Added, nil)
	assertIDs(t, "removed", result.Remove, nil)
	if got := idOf(t, before, path); got != id {
		t.Errorf("the identifier moved: %s became %s", got, id)
	}
}

// fileOf is the one file behind the one item at path.
func fileOf(t *testing.T, items []ports.ScannedItem, path string) ports.ScannedFile {
	t.Helper()
	for _, item := range items {
		if item.Path != path {
			continue
		}
		if len(item.Files) != 1 {
			t.Fatalf("the item at %q has %d files, want one", path, len(item.Files))
		}
		return item.Files[0]
	}
	t.Fatalf("no item at %q", path)
	return ports.ScannedFile{}
}

// TestAMultiPartFilmsPartsAreOneItemsFilesAndAChangeToAnyOfThemIsAnUpdate is
// row 2 for the shape that has more than one file behind it.
//
// A multi-part film is **one** item with two media sources (003 §3.3, AC-4), so
// a part that was renamed, one that disappeared and one that appeared are all
// updates to the same row and never a removal. The parts are compared by
// ordinal and by path as well as by the signal, because a part replaced by a
// differently named file of the same length at the same instant is a change to
// the item that neither half of the signal can see.
func TestAMultiPartFilmsPartsAreOneItemsFilesAndAChangeToAnyOfThemIsAnUpdate(t *testing.T) {
	lib := aMoviesLibrary()
	const (
		one   = "The Long Film (1998)/The Long Film (1998) - part1.mkv"
		two   = "The Long Film (1998)/The Long Film (1998) - part2.mkv"
		three = "The Long Film (1998)/The Long Film (1998) - part3.mkv"
	)
	both := resolved(t, lib, at(one, 700), at(two, 800))
	renamed := resolved(t, lib, at(one, 700), at(three, 800))
	alone := resolved(t, lib, at(one, 700))

	film := idOf(t, both, one)
	if idOf(t, renamed, one) != film || idOf(t, alone, one) != film {
		t.Fatal("this case needs all three readings to resolve to one item, and they did not")
	}

	for _, c := range []struct {
		name             string
		previous, wanted []ports.ScannedItem
	}{
		{"a part was renamed and nothing else moved", both, renamed},
		{"a part disappeared", both, alone},
		{"a part appeared", alone, both},
	} {
		t.Run(c.name, func(t *testing.T) {
			result, err := Reconcile(c.previous, c.wanted, false)
			if err != nil {
				t.Fatalf("reconciling: %v", err)
			}
			assertIDs(t, "updated", result.Updated, []string{film})
			assertIDs(t, "removed", result.Remove, nil)
			assertIDs(t, "added", result.Added, nil)
		})
	}

	// The control: the same two parts twice are believed, so the three updates
	// above came from the part that moved.
	unmoved, err := Reconcile(both, resolved(t, lib, at(one, 700), at(two, 800)), false)
	if err != nil {
		t.Fatalf("reconciling an unchanged pair of parts: %v", err)
	}
	assertIDs(t, "updated by an unchanged pair of parts", unmoved.Updated, nil)
}

// --- Row 3: both unchanged, and what full changes ----------------------------

func TestBothUnchangedIsBelieved(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700), at("The Long Film (2010)/The Long Film (2010).mkv", 800))
	after := resolved(t, lib, at("The Matrix (1999).mkv", 700), at("The Long Film (2010)/The Long Film (2010).mkv", 800))

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	if len(result.Write) != 0 {
		t.Errorf("the batch holds %d items, want none: nothing on disk moved", len(result.Write))
	}
	assertIDs(t, "added", result.Added, nil)
	assertIDs(t, "updated", result.Updated, nil)
	assertIDs(t, "removed", result.Remove, nil)
	if len(result.Unchanged) != len(after) {
		t.Errorf("unchanged: got %d identifiers, want all %d", len(result.Unchanged), len(after))
	}
}

func TestAFullReExaminationDisbelievesAnUnchangedSignal(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700))
	after := resolved(t, lib, at("The Matrix (1999).mkv", 700))

	result, err := Reconcile(before, after, true)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	film := idOf(t, after, "The Matrix (1999).mkv")
	assertIDs(t, "updated", result.Updated, []string{film})

	// The library's own row has no file, so it has no signal to disbelieve. A
	// full scan that rewrote every container would report a library's whole
	// spine as updated on every run.
	folder := idOfNamed(t, after, library.KindCollectionFolder, "Movies")
	assertIDs(t, "unchanged", result.Unchanged, []string{folder})
}

// TestFullChangesOnlyWhetherAnUnchangedSignalIsBelieved runs the same two sets
// twice and asserts that the two answers agree on **every other row**.
//
// Spec §3.8's rule is that *"the default is the fast one, the full one is always
// available"*, and a full re-examination that also changed a removal decision
// would make that untrue in the dangerous direction: an operator reaching for
// the thorough option would be reaching for a different set of deletions.
func TestFullChangesOnlyWhetherAnUnchangedSignalIsBelieved(t *testing.T) {
	lib := aShowsLibrary()
	const (
		unchanged = "Kept/Season 01/Kept - S01E01 - Unchanged.mkv"
		resized   = "Kept/Season 01/Kept - S01E02 - Resized.mkv"
		retimed   = "Kept/Season 01/Kept - S01E03 - Retimed.mkv"
		appeared  = "Kept/Season 02/Kept - S02E01 - Appeared.mkv"
		vanished  = "Gone/Season 01/Gone - S01E01 - Vanished.mkv"
	)
	later := units.At(anInstant().Instant().Add(time.Hour))
	before := resolved(t, lib,
		at(unchanged, 700), at(resized, 800), entry(retimed, 900, anInstant()), at(vanished, 1000))
	after := resolved(t, lib,
		at(unchanged, 700), at(resized, 850), entry(retimed, 900, later), at(appeared, 1100))

	fast, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling without a re-examination: %v", err)
	}
	full, err := Reconcile(before, after, true)
	if err != nil {
		t.Fatalf("reconciling with a re-examination: %v", err)
	}

	// The controls on the corpus: there is something for full to disbelieve,
	// something to remove and something to retain. Without all three, the
	// equalities below are equalities of empty lists.
	if len(fast.Unchanged) == 0 {
		t.Fatal("the fast run believed nothing, so this case cannot tell the two runs apart")
	}
	if len(fast.Remove) == 0 || len(fast.Retained) == 0 {
		t.Fatal("this case needs both a removal and a retained container to watch full leave alone")
	}

	assertIDs(t, "added", full.Added, fast.Added)
	assertIDs(t, "removed", full.Remove, fast.Remove)
	assertIDs(t, "retained", full.Retained, fast.Retained)

	// Everything the fast run believed and that has a file is what moved, and
	// nothing else did.
	filesBehind := map[string]int{}
	for _, item := range after {
		filesBehind[item.ID] = len(item.Files)
	}
	var disbelieved, stillBelieved []string
	for _, id := range fast.Unchanged {
		if filesBehind[id] > 0 {
			disbelieved = append(disbelieved, id)
			continue
		}
		stillBelieved = append(stillBelieved, id)
	}
	if len(disbelieved) == 0 {
		t.Fatal("no believed row had a file, so this case cannot tell the two runs apart")
	}
	assertIDs(t, "updated under a full re-examination", full.Updated,
		sortedIDs(append(slices.Clone(fast.Updated), disbelieved...)...))
	assertIDs(t, "still believed under a full re-examination", full.Unchanged, stillBelieved)
}

// --- Row 4: a previous row with no path in the reading -----------------------

func TestAPreviousRowWithNoPathInTheReadingIsRemoved(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700), at("Deleted (2005).mkv", 800))
	after := resolved(t, lib, at("The Matrix (1999).mkv", 700))

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	assertIDs(t, "removed", result.Remove, []string{idOf(t, before, "Deleted (2005).mkv")})
	assertIDs(t, "added", result.Added, nil)
	assertIDs(t, "updated", result.Updated, nil)
	if len(result.Write) != 0 {
		t.Errorf("the batch holds %d items, want none", len(result.Write))
	}
}

// TestTheRemovalPassMarksFileBackedItemsRemovedAndLeavesTheContainersAboveThem
// is behaviours §5.2, asserted as a series row that survives its last episode's
// deletion.
//
// **Both halves are asserted.** That the series is missing from the removals is
// also what a build that removed nothing at all produces, so the episode being
// removed is the control and the containers being *named* as retained is what
// says the pass looked at them and decided.
func TestTheRemovalPassMarksFileBackedItemsRemovedAndLeavesTheContainersAboveThem(t *testing.T) {
	lib := aShowsLibrary()
	const episode = "The Series/Season 01/The Series - S01E01 - Pilot.mkv"
	before := resolved(t, lib, at(episode, 700))
	after := resolved(t, lib)

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	assertIDs(t, "removed", result.Remove, []string{idOf(t, before, episode)})
	assertIDs(t, "retained", result.Retained, sortedIDs(
		idOf(t, before, "The Series"),
		idOf(t, before, "The Series/Season 01"),
	))
	if len(result.Write) != 0 {
		t.Errorf("the batch holds %d items, want none: a retained container is not rewritten either", len(result.Write))
	}
}

// --- The identifier that is re-derived and compared --------------------------

func TestARecomputedIdentifierThatDisagreesWithTheStoredOneIsAnError(t *testing.T) {
	lib := aMoviesLibrary()
	const path = "The Matrix (1999).mkv"
	after := resolved(t, lib, at(path, 700))

	before := slices.Clone(after)
	const stored = "00000000000000000000000000000000"
	for i := range before {
		if before[i].Path == path {
			before[i].ID = stored
		}
	}

	result, err := Reconcile(before, after, false)

	// Asserted as an error, and not as the absence of a rewritten row: the
	// absence is also what a build that ignored the disagreement produces.
	var mismatch *IdentifierMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("reconciling: got %v, want an *IdentifierMismatchError", err)
	}
	if mismatch.Path != path || mismatch.Stored != stored || mismatch.Derived != idOf(t, after, path) {
		t.Errorf("the error names root %d, %q, stored %s, derived %s — one of the four is wrong",
			mismatch.Root, mismatch.Path, mismatch.Stored, mismatch.Derived)
	}
	if !reflect.DeepEqual(result, Reconciliation{}) {
		t.Errorf("a partial reconciliation came back beside the error: %+v", result)
	}

	// The control: the same two sets with the stored identifier the derivation
	// actually produces reconcile without an error at all, so the case above
	// failed for the reason it names.
	if _, err := Reconcile(after, after, false); err != nil {
		t.Fatalf("the same sets with agreeing identifiers: %v", err)
	}
}

// TestARenameIsARemovalAndAnAdditionAndNotAnIdentifierMismatch is 003 §3.8's
// rename row, and it is here because it is the case the comparison above is
// most likely to be broken into.
//
// Identity is path-derived, so a rename **must** change the identifier. A build
// that read that change as the disagreement above would fail the scan of every
// library anybody ever renamed a file in.
func TestARenameIsARemovalAndAnAdditionAndNotAnIdentifierMismatch(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700))
	after := resolved(t, lib, at("The Matrix Reloaded (2003).mkv", 700))

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	assertIDs(t, "added", result.Added, []string{idOf(t, after, "The Matrix Reloaded (2003).mkv")})
	assertIDs(t, "removed", result.Remove, []string{idOf(t, before, "The Matrix (1999).mkv")})
}

// TestAPathThatChangedOnlyItsCaseInAFoldingLibraryKeepsItsItem is the other
// side of the same comparison: the identifier did not move, so the row is
// updated in place rather than removed and added.
func TestAPathThatChangedOnlyItsCaseInAFoldingLibraryKeepsItsItem(t *testing.T) {
	lib := aMoviesLibrary()
	if lib.CaseSensitive {
		t.Fatal("this case is about a library that folds case")
	}
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700))
	after := resolved(t, lib, at("THE MATRIX (1999).MKV", 700))

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	id := idOf(t, after, "THE MATRIX (1999).MKV")
	if got := idOf(t, before, "The Matrix (1999).mkv"); got != id {
		t.Fatalf("this case needs the two paths to derive one identifier, and they derived %s and %s", got, id)
	}
	assertIDs(t, "updated", result.Updated, []string{id})
	assertIDs(t, "added", result.Added, nil)
	assertIDs(t, "removed", result.Remove, nil)
	if got := fileOf(t, result.Write, "THE MATRIX (1999).MKV").Path; got != "THE MATRIX (1999).MKV" {
		t.Errorf("the batch carries the path %q, want the one on disk", got)
	}
}

// --- The record that moved while the signal stood still ----------------------

func TestAContainerWhoseRecordMovedIsUpdatedThoughItHasNoSignalAtAll(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700))

	renamed := lib
	renamed.Name = "Films"
	renamed.NameFolded = "films"
	after := resolved(t, renamed, at("The Matrix (1999).mkv", 700))

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	// The library's own row has no file. Without a comparison of the record
	// there is nothing about it that could ever move, and the store would hold
	// the old name for the life of the installation.
	folder := idOfNamed(t, after, library.KindCollectionFolder, "Films")
	assertIDs(t, "updated", result.Updated, []string{folder})
	assertIDs(t, "unchanged", result.Unchanged, []string{idOf(t, after, "The Matrix (1999).mkv")})
}

// --- The modification time that is not a whole tick --------------------------

// TestAStoredModificationTimeThatIsNotAWholeTickReportsNoChange is the mistake
// that reports **every** file changed on the first rescan of every installation
// on a filesystem whose resolution is not a tick's.
//
// `units.Time` rounds to a whole tick (100ns) on the way in, and a store keeps
// ticks, so the value a scan reads and the value a store returns have both been
// through the same conversion. The case here is the one where that matters: an
// instant with a sub-tick remainder, taken through the store's round trip on one
// side and fresh from a reading on the other.
func TestAStoredModificationTimeThatIsNotAWholeTickReportsNoChange(t *testing.T) {
	raw := time.Date(2026, 3, 1, 12, 0, 0, 350, time.UTC)

	// The control on the case itself: the instant really is not a whole tick,
	// so this is not a test of two values that were never going to differ.
	if units.At(raw).Instant().Equal(raw) {
		t.Fatal("this case needs an instant that is not a whole tick, and it got one that is")
	}

	lib := aMoviesLibrary()
	const path = "The Matrix (1999).mkv"
	read := resolved(t, lib, entry(path, 700, units.At(raw)))

	// What a store hands back: the same instant, through the ticks column.
	stored := slices.Clone(read)
	for i := range stored {
		files := slices.Clone(stored[i].Files)
		for j := range files {
			files[j].ModifiedAt = units.TimeFromTicks(files[j].ModifiedAt.Ticks())
		}
		stored[i].Files = files
	}

	result, err := Reconcile(stored, read, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}
	assertIDs(t, "updated", result.Updated, nil)
	if len(result.Unchanged) != len(read) {
		t.Errorf("unchanged: got %d identifiers, want all %d — a second reading of the same file changed nothing",
			len(result.Unchanged), len(read))
	}

	// And the control the other way: one whole tick away **is** a change, so the
	// comparison is not simply agreeing with everything.
	oneTickLater := resolved(t, lib, entry(path, 700, units.At(raw.Add(units.TickDuration))))
	moved, err := Reconcile(stored, oneTickLater, false)
	if err != nil {
		t.Fatalf("reconciling one tick later: %v", err)
	}
	assertIDs(t, "updated one tick later", moved.Updated, []string{idOf(t, oneTickLater, path)})
}

// TestTheWalkConvertsAModificationTimeOnceSoASecondReadingOfTheSameFileIsNoChange
// is the assertion above with the real conversion in the middle rather than a
// hand-made one: a file whose modification time on disk carries a sub-tick
// remainder, walked twice.
//
// It is the pair to the pure case because the property is a property of the
// *pipeline*: the walk converts on the way in, the store keeps what the walk
// produced, and Reconcile compares two values that have each been through the
// same conversion once. A test of Reconcile alone cannot see a walk that handed
// it an unconverted instant.
func TestTheWalkConvertsAModificationTimeOnceSoASecondReadingOfTheSameFileIsNoChange(t *testing.T) {
	root := t.TempDir()
	const path = "The Matrix (1999).mkv"
	if err := os.WriteFile(filepath.Join(root, path), []byte("some bytes"), 0o644); err != nil {
		t.Fatalf("writing the film: %v", err)
	}
	raw := time.Date(2026, 3, 1, 12, 0, 0, 350, time.UTC)
	if err := os.Chtimes(filepath.Join(root, path), raw, raw); err != nil {
		t.Fatalf("setting the modification time: %v", err)
	}

	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.ModTime().UTC().Equal(units.At(info.ModTime()).Instant()) {
		t.Skipf("this filesystem stored %s as a whole tick, so the case this test is about "+
			"does not exist on it and nothing here is proven", info.ModTime().UTC())
	}

	first := walkOnce(t, root)
	second := walkOnce(t, root)

	lib := aMoviesLibrary()
	before := resolved(t, lib, first.Entries...)
	after := resolved(t, lib, second.Entries...)

	result, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling: %v", err)
	}
	assertIDs(t, "updated", result.Updated, nil)
	assertIDs(t, "removed", result.Remove, nil)
	if len(result.Unchanged) != len(after) {
		t.Errorf("unchanged: got %d identifiers, want all %d", len(result.Unchanged), len(after))
	}
}

func walkOnce(t *testing.T, root string) library.Reading {
	t.Helper()
	result, err := Walk(os.DirFS(root), 0, library.Movies)
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return result.Reading
}

// --- Determinism -------------------------------------------------------------

// TestReconcileAnswersTheSameWhateverOrderItsTwoSetsArriveIn is Principle VII at
// the layer a plan is produced rather than at the layer it is applied.
//
// Both loops walk their input in the caller's order and both indexes are maps,
// so a build that returned what it accumulated would hand a store a batch whose
// order came from a map's iteration.
func TestReconcileAnswersTheSameWhateverOrderItsTwoSetsArriveIn(t *testing.T) {
	lib := aShowsLibrary()
	before := resolved(t, lib,
		at("The Series/Season 01/The Series - S01E01 - Pilot.mkv", 700),
		at("The Series/Season 01/The Series - S01E02 - Second.mkv", 800),
		at("Gone/Season 01/Gone - S01E01 - First.mkv", 900),
		at("Gone/Season 01/Gone - S01E02 - Second.mkv", 950),
	)
	after := resolved(t, lib,
		at("The Series/Season 01/The Series - S01E01 - Pilot.mkv", 700),
		at("The Series/Season 01/The Series - S01E02 - Second.mkv", 850),
		at("The Series/Season 02/The Series - S02E01 - Later.mkv", 950),
	)

	forwards, err := Reconcile(before, after, false)
	if err != nil {
		t.Fatalf("reconciling forwards: %v", err)
	}
	backwards, err := Reconcile(reversed(before), reversed(after), false)
	if err != nil {
		t.Fatalf("reconciling backwards: %v", err)
	}

	if !reflect.DeepEqual(forwards, backwards) {
		t.Errorf("the two orders answered differently:\n forwards: %+v\nbackwards: %+v", forwards, backwards)
	}
	// The control: this corpus exercises every list, so the equality above is
	// an equality of something.
	for what, list := range map[string][]string{
		"added": forwards.Added, "updated": forwards.Updated,
		"unchanged": forwards.Unchanged, "removed": forwards.Remove,
		"retained": forwards.Retained,
	} {
		if len(list) == 0 {
			t.Errorf("%s is empty, so this case does not compare it", what)
		}
	}
}

func reversed(items []ports.ScannedItem) []ports.ScannedItem {
	out := slices.Clone(items)
	slices.Reverse(out)
	return out
}

func TestReconcileLeavesTheSlicesItWasGivenExactlyAsTheyWere(t *testing.T) {
	lib := aMoviesLibrary()
	before := resolved(t, lib, at("The Matrix (1999).mkv", 700), at("Deleted (2005).mkv", 800))
	after := resolved(t, lib, at("The Matrix (1999).mkv", 750), at("Appeared (2006).mkv", 800))

	beforeCopy := fmt.Sprintf("%+v", before)
	afterCopy := fmt.Sprintf("%+v", after)

	if _, err := Reconcile(before, after, false); err != nil {
		t.Fatalf("reconciling: %v", err)
	}

	if fmt.Sprintf("%+v", before) != beforeCopy {
		t.Error("the previous set was reordered or rewritten underneath its caller")
	}
	if fmt.Sprintf("%+v", after) != afterCopy {
		t.Error("the desired set was reordered or rewritten underneath its caller")
	}
}

// --- Every column the store keeps, one at a time -----------------------------

// TestEveryColumnTheStoreKeepsIsComparedAndTheOneWithNoColumnIsNot moves one
// field of one resolved item at a time and asserts that each move is an update.
//
// It exists because the file signal is only half of what decides an update, and
// because the other half is a list of field comparisons that is easy to write
// once and never extend. The field names are read out of `ports.ScannedItem`
// itself, so **a column added to the record with no line here fails this test**
// rather than silently never being compared — the shape `TestSkipStringNamesEveryRule`
// already uses one package down.
//
// Three fields are deliberately not compared, and the reasons are different:
// `ID` is the key the two sides are matched by, `Files` is the change signal and
// is compared by its own half, and `SortTitle` has no column behind it at all
// (003 plan §5) — a row read back always carries the empty string there, so
// comparing it would report every item updated on every scan the moment 004
// supplies one. The last of the three is asserted rather than excluded silently.
func TestEveryColumnTheStoreKeepsIsComparedAndTheOneWithNoColumnIsNot(t *testing.T) {
	lib := aMoviesLibrary()
	const path = "The Matrix (1999).mkv"
	before := resolved(t, lib, at(path, 700))

	number := 42
	date := units.At(time.Date(1999, 3, 31, 0, 0, 0, 0, time.UTC))
	moves := []struct {
		field  string
		mutate func(*ports.ScannedItem)
	}{
		{"LibraryID", func(i *ports.ScannedItem) { i.LibraryID = "00000000000000000000000000000000" }},
		{"ParentID", func(i *ports.ScannedItem) { i.ParentID = "00000000000000000000000000000000" }},
		{"Type", func(i *ports.ScannedItem) { i.Type = string(library.KindEpisode) }},
		{"Name", func(i *ports.ScannedItem) { i.Name = "Something Else" }},
		{"SortKey", func(i *ports.ScannedItem) { i.SortKey = "something else" }},
		{"Path", func(i *ports.ScannedItem) { i.Path = "Somewhere Else (1999).mkv" }},
		{"RootOrdinal", func(i *ports.ScannedItem) { i.RootOrdinal++ }},
		{"IndexNumber", func(i *ports.ScannedItem) { i.IndexNumber = &number }},
		{"ParentIndexNumber", func(i *ports.ScannedItem) { i.ParentIndexNumber = &number }},
		{"IndexNumberEnd", func(i *ports.ScannedItem) { i.IndexNumberEnd = &number }},
		{"ProductionYear", func(i *ports.ScannedItem) { i.ProductionYear = &number }},
		{"PremiereDate", func(i *ports.ScannedItem) { i.PremiereDate = &date }},
		{"Unplaceable", func(i *ports.ScannedItem) { i.Unplaceable = !i.Unplaceable }},
	}

	// Not compared, and the three reasons are not the same one.
	notColumns := map[string]string{
		"ID":        "the key the two sides are matched by",
		"Files":     "the change signal, compared by its own half",
		"SortTitle": "the one field with no column behind it (003 plan §5)",
	}

	covered := map[string]bool{}
	for _, move := range moves {
		covered[move.field] = true
	}
	record := reflect.TypeOf(ports.ScannedItem{})
	for i := 0; i < record.NumField(); i++ {
		name := record.Field(i).Name
		if _, excluded := notColumns[name]; excluded {
			if covered[name] {
				t.Errorf("%s is both moved here and declared not compared", name)
			}
			continue
		}
		if !covered[name] {
			t.Errorf("ports.ScannedItem.%s has no case here, so nothing says whether a scan "+
				"notices it moving", name)
		}
	}

	for _, move := range moves {
		t.Run(move.field, func(t *testing.T) {
			after := slices.Clone(before)
			var moved string
			for i := range after {
				if after[i].Path != path {
					continue
				}
				item := after[i]
				move.mutate(&item)
				after[i] = item
				moved = item.ID
			}
			if moved == "" {
				t.Fatalf("no item at %q to move", path)
			}

			result, err := Reconcile(before, after, false)
			if err != nil {
				t.Fatalf("reconciling: %v", err)
			}
			assertIDs(t, "updated", result.Updated, []string{moved})
		})
	}

	// The control on the whole table: an untouched copy is believed, so the
	// updates above came from the move and not from the copying.
	unmoved, err := Reconcile(before, slices.Clone(before), false)
	if err != nil {
		t.Fatalf("reconciling an untouched copy: %v", err)
	}
	assertIDs(t, "updated by an untouched copy", unmoved.Updated, nil)

	// And the field that must **not** be an update, asserted rather than left to
	// the exclusion list above.
	withSortTitle := slices.Clone(before)
	for i := range withSortTitle {
		if withSortTitle[i].Path == path {
			item := withSortTitle[i]
			item.SortTitle = "Matrix, The"
			withSortTitle[i] = item
		}
	}
	result, err := Reconcile(before, withSortTitle, false)
	if err != nil {
		t.Fatalf("reconciling a supplied sort title: %v", err)
	}
	assertIDs(t, "updated by a sort title", result.Updated, nil)
}
