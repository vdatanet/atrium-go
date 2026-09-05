package scan

// The scan's own tests sit at the `scan` level of 003 tasks' three: a Go test
// beside the package, over a real temporary tree and a **recording item store**.
//
// # Why a recording store here and a real one in internal/app
//
// The two properties this file exists for are 003 plan §6.9's, and both are
// about what the scan *asks* a store to do rather than about what a store does:
// that the additions arrive in batches, and that nothing renews a claim outside
// a batch. Neither has an injection point in a real store — a scan driven
// through the subcommand builds its own SQLite store and there is nowhere to
// stand between two of its transactions.
//
// So the store here is a fake that models exactly the two guarantees 003 T12
// already proved of the real one — a batch renews the claim and writes its
// items **or neither** — and fails on demand. What that buys is the scan's own
// half of the property, which is the half no store can hold: **the scan must
// have no way to renew a claim except by committing a batch.** A build that
// renewed beside the batch would leave the claim at the failed batch's instant,
// and `TestAFailedBatchLeavesTheClaimWhereTheLastCommittedBatchPutIt` is red.
//
// The guards, the seams and the summary are asserted through the subcommand
// instead, in internal/app, over a real data directory and a real SQLite store
// — because those are criteria about what the store ends up holding, and 001's
// closing audit found twice that a criterion written about an act is not met by
// a test about the mechanism that performs it.
//
// **What none of this proves**: anything a client would receive. 003 produces
// no wire representation at all (003 plan §8.1), so there is no field name, no
// casing and no numeric type here to be right or wrong about.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// --- The recording store ------------------------------------------------------

// recordingStore is a `ports.ItemStore` that keeps what it was asked to do.
//
// It models the two guarantees 003 T12 measured of the SQLite store and nothing
// else: `ApplyScanBatch` renews the claim and writes its items, or does
// neither; and `RemoveItems` and `ReleaseScan` are separate acts a caller can
// be observed not to have performed.
type recordingStore struct {
	// previous is what a scan reads back as the last scan's answer.
	previous []ports.ScannedItem

	// claimedAt and claimedBy are the claim, exactly as `scan_state` holds it.
	claimedAt units.Time
	claimedBy string

	// batches is one entry per batch that **committed**, in order.
	batches [][]string

	// written is every item a committed batch wrote, in order.
	written []ports.ScannedItem

	// removed is every identifier `RemoveItems` was called with. A nil
	// removed and an empty one are different answers: the first says the
	// method was never reached.
	removed  []string
	removals int

	// released is the summary document `ReleaseScan` recorded, and releases
	// counts the calls.
	released []byte
	releases int

	// failBatch is the one-based batch this store refuses. Zero refuses none.
	failBatch int
	batchSeen int

	// refuseClaim makes `ClaimScan` answer false, as it does for a library
	// another scanner holds.
	refuseClaim bool

	// displaced is the claimant `ClaimScan` reports having displaced or lost
	// to.
	displaced string
}

var errBatchRefused = errors.New("the store refused this batch")

func (s *recordingStore) ItemsForLibrary(context.Context, string) ([]ports.ScannedItem, error) {
	return slices.Clone(s.previous), nil
}

func (s *recordingStore) ApplyScanBatch(_ context.Context, batch ports.ScanBatch) error {
	s.batchSeen++
	if s.failBatch == s.batchSeen {
		// Neither the renewal nor the items: this is the whole of what T12
		// measured about the real store's one transaction, and it is the only
		// property this fake is allowed to have.
		return errBatchRefused
	}
	if batch.ClaimedBy != s.claimedBy {
		return errors.New("the claim is no longer held by " + batch.ClaimedBy)
	}
	s.claimedAt = batch.At
	s.batches = append(s.batches, writtenIDs(batch.Items))
	s.written = append(s.written, batch.Items...)
	return nil
}

func (s *recordingStore) RemoveItems(_ context.Context, ids []string) error {
	s.removals++
	if s.removed == nil {
		s.removed = []string{}
	}
	s.removed = append(s.removed, ids...)
	return nil
}

func (s *recordingStore) ClaimScan(_ context.Context, _, by string, at units.Time, _ units.Ticks) (bool, string, error) {
	if s.refuseClaim {
		return false, s.displaced, nil
	}
	s.claimedAt = at
	s.claimedBy = by
	return true, s.displaced, nil
}

func (s *recordingStore) ReleaseScan(_ context.Context, _ string, _ units.Time, summary []byte, _ bool) error {
	s.releases++
	s.released = summary
	s.claimedBy = ""
	return nil
}

func (s *recordingStore) RebuildDerived(context.Context) error { return nil }

// aScanner builds a scanner over the recording store with a stated batch size.
func aScanner(t *testing.T, store *recordingStore, batchSize int) *Scanner {
	t.Helper()
	scanner, err := New(Config{
		Items:     store,
		Clock:     &steppingClock{},
		ClaimedBy: "test/1",
		BatchSize: batchSize,
	})
	if err != nil {
		t.Fatalf("building a scanner: %v", err)
	}
	return scanner
}

// steppingClock answers a different instant on every call, one second apart, so
// that "the claim is at the instant this batch stamped" is a statement a test
// can make about *which* batch.
type steppingClock struct{ calls int }

func (c *steppingClock) Now() units.Time {
	c.calls++
	return units.At(time.Date(2026, 3, 1, 12, 0, c.calls, 0, time.UTC))
}

// --- A tree to scan -----------------------------------------------------------

// aTreeOfFilms writes n films into a fresh directory and returns it.
//
// The names are zero-padded so that the order the walk yields them in is the
// order they were declared in, which is what lets a test say *"the first batch
// holds the first two"*.
func aTreeOfFilms(t *testing.T, n int) string {
	t.Helper()
	root := t.TempDir()
	for i := range n {
		name := filepath.Join(root, films[i]+".mkv")
		if err := os.WriteFile(name, []byte(strings.Repeat("x", i+1)), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return root
}

// films are names that sort in declaration order and are not otherwise
// interesting: nothing here exercises a naming rule, which is what the
// resolver's own tests are for.
var films = []string{
	"Film 01 (2001)", "Film 02 (2002)", "Film 03 (2003)", "Film 04 (2004)",
	"Film 05 (2005)", "Film 06 (2006)", "Film 07 (2007)", "Film 08 (2008)",
}

func aLibraryAt(roots ...string) ports.Library {
	return ports.Library{
		ID:             "0123456789abcdef0123456789abcdef",
		Name:           "Movies",
		NameFolded:     "movies",
		CollectionType: string(library.Movies),
		Roots:          roots,
	}
}

// --- Batching, and the only partial state this feature can leave behind -------

func TestTheAdditionsAreWrittenInBatchesOfTheStatedSize(t *testing.T) {
	store := &recordingStore{}
	scanner := aScanner(t, store, 2)

	changes, err := scanner.Scan(context.Background(), aLibraryAt(aTreeOfFilms(t, 6)), Options{})
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}

	// Seven items: the library's own row and six films. Seven rather than an
	// even number on purpose — a batch size that divides the work exactly
	// hides an off-by-one in the last batch's bounds.
	if len(changes.Added) != 7 {
		t.Fatalf("added %d items, want 7: %v", len(changes.Added), changes.Added)
	}
	// The control the whole test rests on: with a batch size of two and seven
	// items there have to be four batches, and a scan that wrote everything in
	// one transaction would report one.
	sizes := make([]int, 0, len(store.batches))
	for _, batch := range store.batches {
		sizes = append(sizes, len(batch))
	}
	if !slices.Equal(sizes, []int{2, 2, 2, 1}) {
		t.Fatalf("committed batches of %v items, want 2, 2, 2, 1", sizes)
	}
	// Every item, once. A batch loop whose bounds overlapped would write one
	// twice and a store that keyed on the identifier would hide it.
	if len(store.written) != 7 {
		t.Fatalf("wrote %d items across the batches, want 7", len(store.written))
	}

	// And the batch order is the reconciliation's, which is ancestors before
	// what hangs from them: the library's own row is in the first batch.
	if len(store.written) == 0 || store.written[0].Type != string(library.KindCollectionFolder) {
		t.Errorf("the first item written is %v, want the library's own row", store.written)
	}
}

func TestAScanThatDiesBetweenTwoBatchesHasAddedSomeItemsAndRemovedNone(t *testing.T) {
	// A previous scan holding one film that is no longer on disk, so this scan
	// has something real to remove — without it the assertion below is an
	// assertion about an empty list, which is 003 T9's *"an agreement test
	// needs a corpus that could have disagreed"* one layer up.
	root := aTreeOfFilms(t, 5)
	lib := aLibraryAt(root)
	store := &recordingStore{previous: resolved(t, lib, at("A Departed Film (1999).mkv", 100))}
	store.failBatch = 2
	scanner := aScanner(t, store, 2)

	_, err := scanner.Scan(context.Background(), lib, Options{})
	if !errors.Is(err, errBatchRefused) {
		t.Fatalf("scanning with a failing second batch: got %v, want the batch's own error", err)
	}

	// Added some.
	if len(store.batches) != 1 {
		t.Fatalf("committed %d batches, want exactly the one before the failure", len(store.batches))
	}
	if len(store.written) != 2 {
		t.Fatalf("wrote %d items, want the 2 of the committed batch", len(store.written))
	}

	// And removed none. `removed` is nil rather than empty, which is the
	// difference between *"the removal was reached and had nothing to do"* and
	// *"the removal was never reached"* — and the reconciliation above had one
	// identifier to remove, so an empty list here would be the wrong answer
	// too.
	if store.removed != nil {
		t.Errorf("RemoveItems was reached with %v; a scan that dies between batches removes nothing", store.removed)
	}
	if store.releases != 0 {
		t.Errorf("ReleaseScan was reached %d times; a scan that died did not finish", store.releases)
	}
}

func TestAFailedBatchLeavesTheClaimWhereTheLastCommittedBatchPutIt(t *testing.T) {
	root := aTreeOfFilms(t, 5)
	lib := aLibraryAt(root)
	store := &recordingStore{}
	store.failBatch = 3
	scanner := aScanner(t, store, 2)

	if _, err := scanner.Scan(context.Background(), lib, Options{}); !errors.Is(err, errBatchRefused) {
		t.Fatalf("scanning with a failing third batch: got %v, want the batch's own error", err)
	}

	// Two batches committed, so the claim is at the instant the *second* one
	// stamped. The control is that there was a third batch to renew it: with
	// one batch the assertion below would hold on a build that renewed the
	// claim before every batch as well.
	if len(store.batches) != 2 {
		t.Fatalf("committed %d batches, want the 2 before the failure", len(store.batches))
	}
	if store.batchSeen != 3 {
		t.Fatalf("the store saw %d batches, want 3", store.batchSeen)
	}

	// The stepping clock: call 1 is the claim, 2 and 3 are the two committed
	// batches, 4 is the batch that failed. So the claim is at second 3 and a
	// renewal outside the batch would have put it at second 4.
	want := units.At(time.Date(2026, 3, 1, 12, 0, 3, 0, time.UTC))
	if !store.claimedAt.Equal(want) {
		t.Errorf("the claim is at %s, want %s — the instant the last committed batch stamped", store.claimedAt, want)
	}
}

func TestTheClaimIsTakenAfterEveryRootHasBeenReadAndNotBefore(t *testing.T) {
	// 003 plan §6.9 defends staleAfter by saying the claim only has to outlive
	// the gap between two batches. Nothing renews a claim during a walk, so a
	// claim taken before the reading would have to outlive the whole walk —
	// and, more visibly, a guard's refusal would leave a claim behind that
	// stops the operator scanning again once they have fixed the mount.
	store := &recordingStore{}
	scanner := aScanner(t, store, DefaultBatchSize)

	lib := aLibraryAt(filepath.Join(t.TempDir(), "not-here"))
	_, err := scanner.Scan(context.Background(), lib, Options{})
	if !errors.Is(err, ErrUnavailableRoot) {
		t.Fatalf("scanning an absent root: got %v, want an unavailable root", err)
	}
	if store.claimedBy != "" {
		t.Errorf("the library is claimed by %q after a refusal that wrote nothing", store.claimedBy)
	}
}

func TestALibraryAnotherScannerHoldsIsReportedWithTheClaimantsName(t *testing.T) {
	store := &recordingStore{refuseClaim: true, displaced: "elsewhere/99"}
	scanner := aScanner(t, store, DefaultBatchSize)

	_, err := scanner.Scan(context.Background(), aLibraryAt(aTreeOfFilms(t, 2)), Options{})
	if !errors.Is(err, ErrAlreadyScanning) {
		t.Fatalf("scanning a claimed library: got %v, want ErrAlreadyScanning", err)
	}
	var already *AlreadyScanningError
	if !errors.As(err, &already) {
		t.Fatalf("the error is not an *AlreadyScanningError: %v", err)
	}
	if already.ClaimedBy != "elsewhere/99" {
		t.Errorf("the refusal names %q as the claimant, want elsewhere/99", already.ClaimedBy)
	}
	if !strings.Contains(err.Error(), "elsewhere/99") {
		t.Errorf("the message does not name the claimant: %s", err)
	}
	if store.releases != 0 || store.removed != nil {
		t.Errorf("a scanner that lost the claim wrote something: %d releases, removed %v", store.releases, store.removed)
	}
}

// --- The summary document -----------------------------------------------------

func TestTheSummaryRecordedOnTheLibraryIsTheOneTheScanReturned(t *testing.T) {
	root := aTreeOfFilms(t, 3)
	// One path a rule refuses, so that the skip count is not zero and the
	// document has something to be wrong about.
	if err := os.WriteFile(filepath.Join(root, "Not A Film (1999).mp3"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing the refused file: %v", err)
	}

	store := &recordingStore{}
	scanner := aScanner(t, store, DefaultBatchSize)
	changes, err := scanner.Scan(context.Background(), aLibraryAt(root), Options{})
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}

	document, err := changes.Document()
	if err != nil {
		t.Fatalf("encoding the summary: %v", err)
	}
	if string(store.released) != string(document) {
		t.Errorf("the recorded summary is %s, want %s", store.released, document)
	}
	if changes.Skipped != 1 {
		t.Errorf("skipped %d, want the one refused extension", changes.Skipped)
	}
	if changes.Examined != 3 {
		t.Errorf("examined %d, want 3", changes.Examined)
	}
	// An empty list is an empty list and not a null: an operator's tool
	// reading the document should not have to tell the two apart.
	if !strings.Contains(string(document), `"removed":[]`) {
		t.Errorf("a scan that removed nothing wrote %s", document)
	}
}

func TestAConfigurationThatCouldNotProduceThisFeaturesMessagesIsRefused(t *testing.T) {
	// Every one of these is a value that makes a plan §7 row unprintable or a
	// guarantee unkeepable, and the refusal is at construction because a
	// scanner is built once and used against every library.
	for _, testCase := range []struct {
		name   string
		config Config
	}{
		{"no item store", Config{Clock: &steppingClock{}, ClaimedBy: "x"}},
		{"no clock", Config{Items: &recordingStore{}, ClaimedBy: "x"}},
		{"no claimant", Config{Items: &recordingStore{}, Clock: &steppingClock{}}},
		{"a negative batch size", Config{Items: &recordingStore{}, Clock: &steppingClock{}, ClaimedBy: "x", BatchSize: -1}},
		{"a negative staleAfter", Config{Items: &recordingStore{}, Clock: &steppingClock{}, ClaimedBy: "x", StaleAfter: -1}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := New(testCase.config); err == nil {
				t.Fatal("the configuration was accepted")
			}
		})
	}
}

func TestALibraryWhoseCollectionTypeIsNotOneOfTheThreeFailsBeforeAnythingIsRead(t *testing.T) {
	store := &recordingStore{}
	scanner := aScanner(t, store, DefaultBatchSize)

	lib := aLibraryAt(aTreeOfFilms(t, 2))
	lib.CollectionType = "photos"
	_, err := scanner.Scan(context.Background(), lib, Options{})
	if !errors.Is(err, library.ErrCollectionTypeUnknown) {
		t.Fatalf("scanning a library of an unknown type: got %v, want ErrCollectionTypeUnknown", err)
	}
	if store.claimedBy != "" {
		t.Errorf("the library was claimed: %q", store.claimedBy)
	}
}
