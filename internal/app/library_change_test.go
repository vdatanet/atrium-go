package app

// AC-11 and AC-14: what changes on disk, and the user data that outlives an
// item.
//
// # The two criteria, and why they are one file
//
// AC-14 is 003 §3.8's change table made a set of mutations: *"an incremental
// rescan notices exactly what changed"*. AC-11 is one row of that same table —
// a deleted file — followed by the thing the deletion must **not** take with
// it. The deletion row is written once, here, in the AC-11 test, rather than
// twice: 003 T15's lesson is that writing one criterion twice is how one of the
// two ends up asserting a subset of the other.
//
// So the five mutations below cover §3.8's four rows that the two criteria
// name:
//
//	a modified file, size moved and time held      AC-14
//	a modified file, time moved and size held      AC-14
//	a file that appears                            AC-14
//	a file that is renamed                         AC-14
//	a file that is deleted, and comes back         AC-11 (and §3.8's fourth row)
//
// **Size and modification time are varied independently**, which 003 plan §8.4
// requires in terms: *"a build reading only one of the two passes a test that
// varies both"*. Each of the two begins by failing if the case did not turn out
// to be the case it is named for — the half being held really did stand still —
// because `os.Chtimes` typed one line up is a test that quietly varies both and
// proves the weaker thing.
//
// # Where they sit
//
// At the `app` level of 003 tasks' three: through the subcommand, over a real
// temporary data directory, a real SQLite store and a real tree. Both criteria
// are about **what the store ends up holding across two scans**, and neither
// `library.Resolve` nor `scan.Reconcile` has a store to hold anything.
//
// # AC-11's middle clause, and the store method it needed
//
// AC-11 has three clauses and the middle one is the criterion: *delete a file,
// scan, the item is gone; **a row written into a precious table keyed on that
// identifier before the deletion survives the scan**; restore the file, scan,
// and the identifier returns so the association is live again.*
//
// 007 owns user data and 007 does not exist, so the row is written by this test
// through [sqlite.Store.SetItemUserData] rather than by a feature — which is
// stated rather than skipped. The alternative is a criterion with no test until
// somebody else's feature lands, and that is exactly the shape both of this
// project's closing audits caught. The table is
// `migrations/precious/0004_item_user_data.sql` and its header says why it is
// filed by 003.
//
// # This test is what notices, and that is its whole job
//
// 003 plan §6.5 closes with the risk this feature leaves behind:
//
//	"There is no retention rule to write and no orphan to sweep, and the risk
//	is the opposite of the obvious one: a later feature that 'tidies up' user
//	data whose item is gone would break AC-11 and nothing in this feature
//	would fail."
//
// **Nothing else in this suite would notice.** A scan's removal is
// `ItemStore.RemoveItems`, whose cascade reaches `item_files` and stops there;
// `atrium library remove` is a second producer of removals and reaches no
// further either (003 T14). Both are green on a build that deleted every
// `item_user_data` row naming a removed identifier, because no other test in
// this repository writes such a row. The assertion below is the one that goes
// red, and that is measured rather than argued: adding that sweep to
// `RemoveItems` fails
// [TestADeletedFilesUserDataOutlivesItAndTheAssociationComesBack] and **not one
// other test in the tree** `[measurement: 003 T16, 7 mutations, 2026-09-05]`.
//
// So a later feature that adds a retention rule, an orphan sweep or a cascade
// has to come here and argue with 003 §3.8 rather than discover the rule by
// breaking it in a client's library.
//
// # The mutations, which were run
//
// One at a time, against a scratch copy of the tree, each run against all five
// tests here and against the whole of `go test ./...`
// `[measurement: 003 T16, 7 mutations, 2026-09-05]`. The last column is what
// else in the repository went red, which is what says whether an assertion here
// is the only one watching:
//
//	                                        size time appear rename AC-11 else
//	signalMoved stops reading Size          RED  -    -      -      -     scan
//	signalMoved stops reading ModifiedAt    -    RED  -      -      -     scan
//	RemoveItems sweeps item_user_data       -    -    -      -      RED   nothing
//	a new item with no file is not written  -    -    RED    -      -     scan, app
//	a previous row adopted where the whole
//	  file signal matches                   -    -    -      -      -     nothing
//	  the same, over size and time alone    -    -    -      RED    -     scan
//	the scan applies no removal at all      -    -    -      RED    RED   app
//
// Three of the seven are worth reading rather than counting.
//
// **The sweep is caught here and nowhere else**, which is the paragraph above
// turned into a measurement. It is the only row of this table whose last column
// is empty in both directions: the mutation is invisible to every other test in
// the repository, and this one sees it.
//
// **A rename tracked by content is what the rename row is really about.** §3.8
// says a rename is *"treated as delete plus add — identity is path-derived, so
// it changes"*, and the build that gets that wrong is not one that forgets to
// remove anything: it is one that recognises the file at its new path by its
// size and its modification time and carries the old identifier over. That
// build reports one update where the criterion requires one removal and one
// addition. A test that asserted only *"the new path has an item"* passes on
// it, which is why the assertion below is on the **shape** of the change.
//
// **And that mutation was wrong as first written, which is 003 T15's lesson
// arriving again.** *"Adopt a previous row whose file signal matches"* spelled
// with `signalMoved` adopts nothing at all: `signalMoved` compares the file's
// **path** as well as its size and time (a multi-part film's parts are one
// item's files, 003 §3.3), and a renamed file's path is the one thing that did
// move. That build is green everywhere, including here. The mutation the
// criterion can actually fail is adoption over the size and the modification
// time **alone**, and running it is the only thing that told the two apart —
// the sixteenth time in this feature that an assertion, or the mutation named
// beside one, could not produce the failure it was named for.
//
// # What none of this proves
//
// Anything a client would receive. 003 registers no route, so the user data
// asserted below is the **row**, and that a `UserData` object on an `/Items`
// response carries it is 007's — as is every rule about what the values mean.
// This file asserts that the row is still there, not that anything reads it.

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// --- Helpers ------------------------------------------------------------------

// libraryCalled is one of the fixture's declared libraries, by name.
func libraryCalled(t *testing.T, libraries []libraryReport, name string) libraryReport {
	t.Helper()
	for _, report := range libraries {
		if report.Name == name {
			return report
		}
	}
	t.Fatalf("the fixture declared no library called %q", name)
	return libraryReport{}
}

// anAccount creates an account through `atrium user add` and returns its
// identifier, which is what a precious row keyed on a user needs.
func anAccount(t *testing.T, data, name string) string {
	t.Helper()
	mustProvision(t, "", userAdd, "--"+flagDataDirectory, data,
		"--"+flagName, name, "--"+flagNoPassword)

	store, err := openStoreAt(context.Background(), data)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	user, found, err := store.UserByFoldedName(context.Background(), users.Fold(name))
	if err != nil || !found {
		t.Fatalf("reading back the account %q: found %v, err %v", name, found, err)
	}
	return user.ID
}

// writeUserData writes one account's favourite and resume position for an item,
// which until 007 exists is how a precious row keyed on an item identifier
// comes to be.
func writeUserData(t *testing.T, data, userID, itemID string, held sqlite.ItemUserData) {
	t.Helper()
	store, err := openStoreAt(context.Background(), data)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	if err := store.SetItemUserData(context.Background(), userID, itemID, held); err != nil {
		t.Fatalf("writing user data for %s: %v", itemID, err)
	}
}

// readUserData reads it back, and reports whether there is a row at all.
func readUserData(t *testing.T, data, userID, itemID string) (sqlite.ItemUserData, bool) {
	t.Helper()
	store, err := openStoreAt(context.Background(), data)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	held, found, err := store.ItemUserData(context.Background(), userID, itemID)
	if err != nil {
		t.Fatalf("reading user data for %s: %v", itemID, err)
	}
	return held, found
}

// itemWithID is the stored item carrying an identifier, if any still is.
func itemWithID(items []ports.ScannedItem, id string) (ports.ScannedItem, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return ports.ScannedItem{}, false
}

// itemAt is the stored item at a root-relative path.
func itemAt(t *testing.T, items []ports.ScannedItem, path string) ports.ScannedItem {
	t.Helper()
	for _, item := range items {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("no stored item at %q; the store holds %s", path, describe(items))
	return ports.ScannedItem{}
}

// theLibraryRootRow is the library's own `CollectionFolder` row, which is the
// one item in a library that hangs from nothing (003 §3.6).
//
// It is found by type and not by an empty path: an inferred container has no
// path either, and a helper that returned the first path-less row would name a
// season on some trees and the root on others.
func theLibraryRootRow(t *testing.T, items []ports.ScannedItem) ports.ScannedItem {
	t.Helper()
	for _, item := range items {
		if item.Type == string(library.KindCollectionFolder) {
			return item
		}
	}
	t.Fatalf("the store holds no CollectionFolder row; it holds %s", describe(items))
	return ports.ScannedItem{}
}

// theOneFileOf is the single file behind an item, and it fails the test where
// there is not exactly one — a multi-part film's signal is two files and a
// test that read `Files[0]` of one would be varying half of it.
func theOneFileOf(t *testing.T, item ports.ScannedItem) ports.ScannedFile {
	t.Helper()
	if len(item.Files) != 1 {
		t.Fatalf("the item at %q has %d files, and this assertion is about one",
			item.Path, len(item.Files))
	}
	return item.Files[0]
}

// nothingElseMoved fails unless every identifier the store held before is still
// there, naming an item at the same path.
//
// It is the control every mutation below needs in the direction the criterion
// does not name: a scan that noticed the change **and rewrote the rest of the
// library** meets *"exactly what changed"* on the half a test usually asserts
// and fails it on the half it usually does not.
func nothingElseMoved(t *testing.T, what string, before, after []ports.ScannedItem, except ...string) {
	t.Helper()
	for _, was := range before {
		if slices.Contains(except, was.ID) {
			continue
		}
		is, ok := itemWithID(after, was.ID)
		if !ok {
			t.Errorf("%s: the item %s (%s at %q) is gone, and only %v should have moved",
				what, was.ID, was.Type, was.Path, except)
			continue
		}
		if is.Path != was.Path || is.Type != was.Type || is.Name != was.Name {
			t.Errorf("%s: %s was %s %q at %q and is now %s %q at %q",
				what, was.ID, was.Type, was.Name, was.Path, is.Type, is.Name, is.Path)
		}
	}
}

// reportedNothingElse fails unless every library but the one named reported no
// change at all. One tree is mutated per test, so a second library reporting an
// addition is a scan writing outside the library it read.
func reportedNothingElse(t *testing.T, summaries map[string]scanReport, mutated string) {
	t.Helper()
	for name, summary := range summaries {
		if name == mutated {
			continue
		}
		if len(summary.Added)+len(summary.Updated)+len(summary.Removed) != 0 {
			t.Errorf("%q reports %d added, %d updated and %d removed, and only %q was touched",
				name, len(summary.Added), len(summary.Updated), len(summary.Removed), mutated)
		}
	}
}

// --- AC-14: a modified file, with the two halves of the signal varied apart ---

// TestAFileWhoseSizeMovedIsReInspectedWithItsModificationTimeHeldStill is the
// first half of §3.8's *"File modified (size or time of change)"* row.
//
// The file's length changes and its recorded time of change is put back exactly
// where it was, which is what an ordinary restore from a backup does to a file
// and is the reason 003 plan §6.4 states the signal as a pair. A build whose
// change detection reads only the modification time reports this library
// unchanged.
//
// What is asserted afterwards is not the signal but its consequence: the item
// keeps its identifier, its stored **size** is the new one — a media source
// carries `Size`, so that number is observable (behaviours §2.17) — and its
// stored modification time did not move, because it did not.
func TestAFileWhoseSizeMovedIsReInspectedWithItsModificationTimeHeldStill(t *testing.T) {
	t.Parallel()

	trees := t.TempDir()
	data, libraries := theWholeFixture(t, trees)
	mustScan(t, data)

	movies := libraryCalled(t, libraries, "Movies")
	const film = "The Matrix (1999)/The Matrix (1999).mkv"
	before := storedItems(t, data, movies.ID)
	item := itemAt(t, before, film)
	was := theOneFileOf(t, item)

	user := anAccount(t, data, "Ada")
	held := sqlite.ItemUserData{IsFavourite: true, PlaybackPositionTicks: 123_456_789}
	writeUserData(t, data, user, item.ID, held)

	// The mutation: more bytes, and the modification time put back.
	onDisk := filepath.Join(trees, "Movies", filepath.FromSlash(film))
	stat, err := os.Stat(onDisk)
	if err != nil {
		t.Fatalf("reading %s: %v", onDisk, err)
	}
	wasModified := stat.ModTime()
	if err := os.WriteFile(onDisk, []byte(strings.Repeat("x", int(was.Size)+512)), 0o644); err != nil {
		t.Fatalf("rewriting %s: %v", onDisk, err)
	}
	if err := os.Chtimes(onDisk, wasModified, wasModified); err != nil {
		t.Fatalf("putting the modification time of %s back: %v", onDisk, err)
	}

	// The control, and it is the reason this test is not the one below with a
	// different name: the case really is the case it is named for.
	stat, err = os.Stat(onDisk)
	if err != nil {
		t.Fatalf("reading %s again: %v", onDisk, err)
	}
	if stat.Size() == was.Size {
		t.Fatalf("%s is still %d bytes, so nothing varied", onDisk, was.Size)
	}
	if !stat.ModTime().Equal(wasModified) {
		t.Fatalf("%s reports a modification time of %s and it was put back to %s, so this test "+
			"varies both halves of the signal and proves the weaker thing",
			onDisk, stat.ModTime(), wasModified)
	}

	summaries := scanSummaries(t, data)

	if !slices.Equal(summaries["Movies"].Updated, []string{item.ID}) {
		t.Errorf("a file whose size moved with its modification time held still is reported "+
			"updated %v, want exactly %v — a build reading only the modification time reports "+
			"nothing here", summaries["Movies"].Updated, []string{item.ID})
	}
	if len(summaries["Movies"].Added)+len(summaries["Movies"].Removed) != 0 {
		t.Errorf("a modified file is an update and not a removal and an addition: %d added, %d removed",
			len(summaries["Movies"].Added), len(summaries["Movies"].Removed))
	}
	reportedNothingElse(t, summaries, "Movies")

	after := storedItems(t, data, movies.ID)
	rescanned := itemAt(t, after, film)
	if rescanned.ID != item.ID {
		t.Errorf("the item at %q is %s and was %s: a modified file keeps its identity, because "+
			"the identifier is a function of the path and the path did not move (003 §3.8)",
			film, rescanned.ID, item.ID)
	}
	is := theOneFileOf(t, rescanned)
	if is.Size != stat.Size() {
		t.Errorf("the store holds %d bytes for %q and the file is %d: the item was not re-inspected",
			is.Size, film, stat.Size())
	}
	if !is.ModifiedAt.Equal(was.ModifiedAt) {
		t.Errorf("the stored modification time moved to %s from %s, and the file's did not move at all",
			is.ModifiedAt, was.ModifiedAt)
	}

	// And the half of the row that is not about the file at all.
	stillHeld, found := readUserData(t, data, user, item.ID)
	if !found {
		t.Fatalf("the user data keyed on %s is gone after the item was updated (003 §3.8)", item.ID)
	}
	if stillHeld != held {
		t.Errorf("the user data keyed on %s reads back %+v, want %+v", item.ID, stillHeld, held)
	}

	nothingElseMoved(t, "a file whose size moved", before, after, item.ID)
}

// TestAFileWhoseModificationTimeMovedIsReInspectedWithItsSizeHeldStill is the
// other half of the same row, and the mutation that separates the two is
// `signalMoved` dropping one comparison or the other.
//
// A re-encode keeps a file's length and moves its time of change, which is the
// failure this direction catches. A build whose change detection reads only the
// size reports this library unchanged.
func TestAFileWhoseModificationTimeMovedIsReInspectedWithItsSizeHeldStill(t *testing.T) {
	t.Parallel()

	trees := t.TempDir()
	data, libraries := theWholeFixture(t, trees)
	mustScan(t, data)

	movies := libraryCalled(t, libraries, "Movies")
	const film = "2 Fast 2 Furious (2003).mkv"
	before := storedItems(t, data, movies.ID)
	item := itemAt(t, before, film)
	was := theOneFileOf(t, item)

	user := anAccount(t, data, "Ada")
	held := sqlite.ItemUserData{IsFavourite: false, PlaybackPositionTicks: 42}
	writeUserData(t, data, user, item.ID, held)

	// The mutation: the same bytes, two hours later. Exactly two hours, so
	// that the sub-tick remainder `units.At` rounds is the one it rounded
	// before and the stored value below is comparable.
	onDisk := filepath.Join(trees, "Movies", film)
	stat, err := os.Stat(onDisk)
	if err != nil {
		t.Fatalf("reading %s: %v", onDisk, err)
	}
	wasModified := stat.ModTime()
	later := wasModified.Add(2 * time.Hour)
	if err := os.Chtimes(onDisk, later, later); err != nil {
		t.Fatalf("moving the modification time of %s: %v", onDisk, err)
	}

	// The control in this direction: the length really did stand still, and
	// the time really did move. The comparison is against what the filesystem
	// reported a moment ago rather than against the stored tick, because the
	// stored value has been rounded and an equality against it could be false
	// before anything was varied at all.
	stat, err = os.Stat(onDisk)
	if err != nil {
		t.Fatalf("reading %s again: %v", onDisk, err)
	}
	if stat.Size() != was.Size {
		t.Fatalf("%s is %d bytes and was %d, so this test varies both halves of the signal",
			onDisk, stat.Size(), was.Size)
	}
	if stat.ModTime().Equal(wasModified) {
		t.Fatalf("%s still reports a modification time of %s, so nothing varied",
			onDisk, stat.ModTime())
	}

	summaries := scanSummaries(t, data)

	if !slices.Equal(summaries["Movies"].Updated, []string{item.ID}) {
		t.Errorf("a file whose modification time moved with its size held still is reported "+
			"updated %v, want exactly %v — a build reading only the size reports nothing here",
			summaries["Movies"].Updated, []string{item.ID})
	}
	if len(summaries["Movies"].Added)+len(summaries["Movies"].Removed) != 0 {
		t.Errorf("a modified file is an update and not a removal and an addition: %d added, %d removed",
			len(summaries["Movies"].Added), len(summaries["Movies"].Removed))
	}
	reportedNothingElse(t, summaries, "Movies")

	after := storedItems(t, data, movies.ID)
	rescanned := itemAt(t, after, film)
	if rescanned.ID != item.ID {
		t.Errorf("the item at %q is %s and was %s: a modified file keeps its identity (003 §3.8)",
			film, rescanned.ID, item.ID)
	}
	is := theOneFileOf(t, rescanned)
	if is.Size != was.Size {
		t.Errorf("the store holds %d bytes for %q and it held %d, and the file's length did not move",
			is.Size, film, was.Size)
	}
	if !is.ModifiedAt.Equal(units.At(later)) {
		t.Errorf("the store holds a modification time of %s for %q, want %s: the item was not "+
			"re-inspected", is.ModifiedAt, film, units.At(later))
	}

	stillHeld, found := readUserData(t, data, user, item.ID)
	if !found {
		t.Fatalf("the user data keyed on %s is gone after the item was updated (003 §3.8)", item.ID)
	}
	if stillHeld != held {
		t.Errorf("the user data keyed on %s reads back %+v, want %+v", item.ID, stillHeld, held)
	}

	nothingElseMoved(t, "a file whose modification time moved", before, after, item.ID)
}

// --- AC-14: a file that appears ------------------------------------------------

// TestAFileThatAppearsIsAddedWithItsAncestors is §3.8's *"New file — add the
// item, creating ancestors as needed"*.
//
// The tree gains a series nobody declared: one episode, three levels down, in
// directories that did not exist. What must appear is **three** items and not
// one — the series and the season are the *"as needed"* of that row, and a
// build that wrote only the file-backed item leaves an episode hanging from a
// parent identifier no row carries.
//
// It is asserted as the parent chain rather than as a count, because three
// items whose `parent_id` columns point anywhere at all is also a count of
// three.
func TestAFileThatAppearsIsAddedWithItsAncestors(t *testing.T) {
	t.Parallel()

	trees := t.TempDir()
	data, libraries := theWholeFixture(t, trees)
	mustScan(t, data)

	shows := libraryCalled(t, libraries, "Shows")
	before := storedItems(t, data, shows.ID)

	const (
		series  = "Another Series"
		season  = "Another Series/Season 01"
		episode = "Another Series/Season 01/Another Series - S01E01 - Pilot.mkv"
	)
	for _, path := range []string{series, season, episode} {
		for _, item := range before {
			if item.Path == path {
				t.Fatalf("the fixture already holds something at %q, so this test adds nothing", path)
			}
		}
	}

	onDisk := filepath.Join(trees, "Shows", filepath.FromSlash(episode))
	if err := os.MkdirAll(filepath.Dir(onDisk), 0o755); err != nil {
		t.Fatalf("making %s: %v", filepath.Dir(onDisk), err)
	}
	if err := os.WriteFile(onDisk, []byte("a new episode"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", onDisk, err)
	}

	summaries := scanSummaries(t, data)

	after := storedItems(t, data, shows.ID)
	newSeries := itemAt(t, after, series)
	newSeason := itemAt(t, after, season)
	newEpisode := itemAt(t, after, episode)

	if got := []string{newSeries.Type, newSeason.Type, newEpisode.Type}; !slices.Equal(
		got, []string{"Series", "Season", "Episode"}) {
		t.Errorf("the three items that appeared are %v, want [Series Season Episode]", got)
	}

	// The chain, which is what "creating ancestors" means and what a count
	// cannot say.
	if newEpisode.ParentID != newSeason.ID {
		t.Errorf("the episode's parent is %s and the season that appeared with it is %s",
			newEpisode.ParentID, newSeason.ID)
	}
	if newSeason.ParentID != newSeries.ID {
		t.Errorf("the season's parent is %s and the series that appeared with it is %s",
			newSeason.ParentID, newSeries.ID)
	}
	root := theLibraryRootRow(t, after)
	if newSeries.ParentID != root.ID {
		t.Errorf("the series' parent is %s and the library's own row is %s",
			newSeries.ParentID, root.ID)
	}

	wanted := []string{newSeries.ID, newSeason.ID, newEpisode.ID}
	slices.Sort(wanted)
	if !slices.Equal(summaries["Shows"].Added, wanted) {
		t.Errorf("the scan reports %v added and the three items that appeared are %v: "+
			"an appearing file is added **with its ancestors** (003 §3.8)",
			summaries["Shows"].Added, wanted)
	}
	if len(summaries["Shows"].Updated)+len(summaries["Shows"].Removed) != 0 {
		t.Errorf("a file that appeared updated %d items and removed %d, and it should have "+
			"done neither: %v, %v",
			len(summaries["Shows"].Updated), len(summaries["Shows"].Removed),
			summaries["Shows"].Updated, summaries["Shows"].Removed)
	}
	reportedNothingElse(t, summaries, "Shows")

	nothingElseMoved(t, "a file that appeared", before, after)
}

// --- AC-14: a file that is renamed ---------------------------------------------

// TestARenamedFileIsARemovalAndAnAdditionAndNotAnUpdate is §3.8's *"File
// renamed — treated as delete plus add — identity is path-derived, so it
// changes"*.
//
// **The name of this test is the criterion.** A build that recognised the file
// at its new path by its size and its modification time and carried the old
// identifier over reports one update, holds an item at the new path, and passes
// every other assertion in this file. So what is asserted is the *shape* of the
// change — one removal and one addition, no update — and not merely that
// something is at the new path.
//
// A rename is deliberately not [scan.IdentifierMismatchError] either: that
// error is a path whose stored and derived identifiers disagree, and a renamed
// file's previous row is at a path the reading no longer holds.
func TestARenamedFileIsARemovalAndAnAdditionAndNotAnUpdate(t *testing.T) {
	t.Parallel()

	trees := t.TempDir()
	data, libraries := theWholeFixture(t, trees)
	mustScan(t, data)

	movies := libraryCalled(t, libraries, "Movies")
	const (
		was = "Don't Look Up (2021).mkv"
		now = "Look Down (2021).mkv"
	)
	before := storedItems(t, data, movies.ID)
	item := itemAt(t, before, was)

	if err := os.Rename(
		filepath.Join(trees, "Movies", was),
		filepath.Join(trees, "Movies", now)); err != nil {
		t.Fatalf("renaming %s: %v", was, err)
	}

	summaries := scanSummaries(t, data)

	after := storedItems(t, data, movies.ID)
	renamed := itemAt(t, after, now)

	if renamed.ID == item.ID {
		t.Fatalf("the item at %q carries the identifier %s that the item at %q carried: "+
			"identity is path-derived, so a rename changes it (003 §3.8)", now, renamed.ID, was)
	}
	if _, ok := itemWithID(after, item.ID); ok {
		t.Errorf("the store still holds an item under %s, which is the identifier the renamed "+
			"file had before", item.ID)
	}

	if !slices.Equal(summaries["Movies"].Removed, []string{item.ID}) {
		t.Errorf("the scan reports %v removed, want exactly %v: a rename is a delete **plus** "+
			"an add, and a build that tracked the file by its content reports neither",
			summaries["Movies"].Removed, []string{item.ID})
	}
	if !slices.Equal(summaries["Movies"].Added, []string{renamed.ID}) {
		t.Errorf("the scan reports %v added, want exactly %v",
			summaries["Movies"].Added, []string{renamed.ID})
	}
	if len(summaries["Movies"].Updated) != 0 {
		t.Errorf("the scan reports %v updated: a renamed file is not an update, and a build "+
			"that carried the old identifier over reports exactly one",
			summaries["Movies"].Updated)
	}
	reportedNothingElse(t, summaries, "Movies")

	nothingElseMoved(t, "a renamed file", before, after, item.ID)
}

// --- AC-11, which carries §3.8's deletion row ----------------------------------

// TestADeletedFilesUserDataOutlivesItAndTheAssociationComesBack is AC-11, in
// the three clauses the criterion has, and it is also §3.8's *"File deleted —
// remove the item, **preserving user data** in case it returns"* row of AC-14.
//
// The three clauses:
//
//  1. delete a file, scan, the item is gone
//  2. a row written into a precious table keyed on that identifier before the
//     deletion is still there afterwards
//  3. restore the file, scan, and the identifier returns, so the association
//     is live again
//
// **The middle clause is the criterion**, and the first and third are what make
// it mean anything: a row that survived a scan which removed nothing has
// survived nothing, and a row naming an identifier that never comes back is a
// row nobody can use.
//
// It is also the only assertion in this repository that would notice a later
// feature adding a retention rule or an orphan sweep — see this file's header,
// where the measurement is. 003 plan §6.5's closing paragraph is what it
// guards, and the guard is deliberately at the level a sweep would be written
// at: through a real scan, over a real store, against a row a real store method
// wrote.
func TestADeletedFilesUserDataOutlivesItAndTheAssociationComesBack(t *testing.T) {
	t.Parallel()

	trees := t.TempDir()
	data, libraries := theWholeFixture(t, trees)
	mustScan(t, data)

	movies := libraryCalled(t, libraries, "Movies")
	const film = "Rock & Roll (1978).mkv"
	before := storedItems(t, data, movies.ID)
	item := itemAt(t, before, film)

	user := anAccount(t, data, "Ada")
	held := sqlite.ItemUserData{IsFavourite: true, PlaybackPositionTicks: 987_654_321}
	writeUserData(t, data, user, item.ID, held)

	// The control the middle clause needs before anything is deleted: the row
	// this test is about really is there, and carries what it was given. A
	// build whose write silently did nothing would satisfy every "it is still
	// there" assertion below by having nothing to lose.
	if written, found := readUserData(t, data, user, item.ID); !found || written != held {
		t.Fatalf("the user data keyed on %s reads back %+v (found %v) before anything was "+
			"deleted, want %+v", item.ID, written, found, held)
	}

	onDisk := filepath.Join(trees, "Movies", film)
	if err := os.Remove(onDisk); err != nil {
		t.Fatalf("deleting %s: %v", onDisk, err)
	}

	summaries := scanSummaries(t, data)

	// Clause 1: the item is gone.
	if !slices.Equal(summaries["Movies"].Removed, []string{item.ID}) {
		t.Errorf("the scan reports %v removed and the deleted file's item is %s",
			summaries["Movies"].Removed, item.ID)
	}
	withoutTheFilm := storedItems(t, data, movies.ID)
	if _, ok := itemWithID(withoutTheFilm, item.ID); ok {
		t.Errorf("the store still holds the item %s whose only file was deleted", item.ID)
	}
	reportedNothingElse(t, summaries, "Movies")
	nothingElseMoved(t, "a deleted file", before, withoutTheFilm, item.ID)

	// Clause 2, and it is the criterion. 003 §3.8: user data is "keyed by
	// identity and retained after the item is gone".
	survived, found := readUserData(t, data, user, item.ID)
	if !found {
		t.Fatalf("the user data keyed on %s is gone with the item, and 003 §3.8 requires that a "+
			"file which disappears not cost the user their favourites and resume position. "+
			"If a retention rule or an orphan sweep was added since this test was written, "+
			"this is the assertion 003 plan §6.5 predicted it would break — and nothing else "+
			"in this suite would have noticed", item.ID)
	}
	if survived != held {
		t.Errorf("the user data keyed on %s reads back %+v after the item was removed, want %+v",
			item.ID, survived, held)
	}

	// Clause 3: the file comes back — a re-download, a remount, a share that
	// was briefly unavailable — and the identifier is the one the row names.
	// The bytes are deliberately different: identity is path-derived, so what
	// makes the association live again is the path and not the content.
	if err := os.WriteFile(onDisk, []byte("the same film, fetched again"), 0o644); err != nil {
		t.Fatalf("restoring %s: %v", onDisk, err)
	}

	summaries = scanSummaries(t, data)
	if !slices.Equal(summaries["Movies"].Added, []string{item.ID}) {
		t.Errorf("the scan after the file came back reports %v added, want exactly %v: the "+
			"identifier is a function of the path, so the item that returns is the item that left",
			summaries["Movies"].Added, []string{item.ID})
	}
	if len(summaries["Movies"].Removed) != 0 {
		t.Errorf("the scan after the file came back reports %v removed", summaries["Movies"].Removed)
	}

	restored := storedItems(t, data, movies.ID)
	if returned := itemAt(t, restored, film); returned.ID != item.ID {
		t.Errorf("the restored file's item is %s and the row of user data names %s",
			returned.ID, item.ID)
	}

	// And the association is live: the identifier the surviving row names is an
	// item the store holds again. That is the sentence the whole criterion is,
	// and it is asserted as the join a client would eventually make rather than
	// as two facts side by side.
	live, found := readUserData(t, data, user, item.ID)
	if !found {
		t.Fatalf("the user data keyed on %s is gone after the file came back", item.ID)
	}
	if live != held {
		t.Errorf("the user data keyed on %s reads back %+v after the file came back, want %+v",
			item.ID, live, held)
	}
	if _, ok := itemWithID(restored, item.ID); !ok {
		t.Errorf("the row of user data names %s and the store holds no such item, so the "+
			"association is not live", item.ID)
	}
}
