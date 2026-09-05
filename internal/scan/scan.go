package scan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// Changes is 003 §3.8's summary: what one scan of one library did.
//
// It is 003 plan §5's record, and three of its six fields come from a
// [Reconciliation] while the other three are the walk's and the resolver's
// counts.
//
// **[Changes.Skipped] and [Changes.Unplaceable] are two numbers and adding them
// together is the mistake §3.8 is written to prevent.** *"A file that was
// skipped is not in the library and a file that was noticed is, so an operator
// told that both were 'skipped' would go looking for something that is not
// missing."* A skipped path has no item; an unplaceable one has an item, an
// identifier and a parent, and everything a client can do with an item it can
// do with that one.
//
// The three identifier lists are lists rather than counts because a summary an
// operator reads names what moved, and because a test asserting a count of one
// cannot say *which* one. They are never nil: a scan that changed nothing says
// so with three empty lists rather than with three nulls in its document.
type Changes struct {
	// Added is an item the previous scan had no row for.
	Added []string `json:"added"`

	// Updated is an item whose record or whose file signal moved, and whose
	// identifier therefore did not.
	Updated []string `json:"updated"`

	// Removed is an item that is gone. It holds only file-backed items:
	// a container that lost every one of its files keeps its record
	// ([behaviours §5.2], 003 plan §6.5).
	//
	// [behaviours §5.2]: ../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed
	Removed []string `json:"removed"`

	// Examined is how many candidate files every root of the library yielded.
	Examined int `json:"examined"`

	// Skipped is how many paths a rule refused, summed over every root. It is
	// the **walk's** count and never the resolver's: [library.Resolve] applies
	// `CollectionType.Candidate` a second time as the same rule rather than a
	// second one, so adding `Plan.Skipped` to it counts a refusal twice.
	//
	// **No test can see that mistake and that is a measurement rather than a
	// gap.** Over a real scan the walk hands the resolver only paths it has
	// already accepted, so `Plan.Skipped` is always empty and adding it changes
	// nothing; a corpus that could tell them apart would have to reach
	// [library.Resolve] without a walk in front of it, which no scan does. It
	// is the walk's count anyway because the rule is *"report what the walk
	// refused"* rather than *"report whatever happens to be non-empty"*
	// [measurement: 003 T13, 26 mutations, this the only survivor, 2026-09-05].
	//
	// It is not a file count either. A rule that refuses a whole subtree — a
	// hidden directory, an `.ignore` marker — contributes one refusal naming
	// the directory and nothing for the files beneath it, because the walk
	// never opened them.
	Skipped int `json:"skipped"`

	// Unplaceable is how many items were made whose names said too little to
	// place them. See the type's own comment: this is not [Changes.Skipped]
	// and the two are never added together.
	Unplaceable int `json:"unplaceable"`
}

// Document is the summary as it is stored on the library's scan state and as
// `atrium library scan --format json` prints it.
//
// It is `encoding/json` rather than `internal/wire` for the reason
// `users.Policy.Document` gives one feature earlier: a stored document is not a
// wire body. No client ever receives these bytes, no property name here is
// Jellyfin's, and the renaming and casing rules `internal/wire` exists to
// enforce have nothing to say about a document this project both writes and
// reads.
func (c Changes) Document() ([]byte, error) {
	document, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("scan: encoding the summary: %w", err)
	}
	return document, nil
}

// ErrAlreadyScanning is the library another scanner holds the claim on.
//
// It is not a fault. Two scanners over one store is a state this feature
// creates on purpose — an operator may run `atrium library scan` against a data
// directory a server is serving from (003 plan §6.7) — so this is an outcome
// the caller reports and exits on, and 003 plan §7's row for it says
// *"not a fault"*.
var ErrAlreadyScanning = errors.New("the library is already being scanned")

// AlreadyScanningError names the claimant, which is the whole reason
// `ClaimScan` returns one.
type AlreadyScanningError struct {
	LibraryID   string
	LibraryName string

	// ClaimedBy is the scanner that holds the claim, as it named itself. It
	// is empty only if the store lost the name, which it does not.
	ClaimedBy string
}

func (e *AlreadyScanningError) Error() string {
	return fmt.Sprintf("scan: library %q (%s) is already being scanned by %q",
		e.LibraryName, e.LibraryID, e.ClaimedBy)
}

func (e *AlreadyScanningError) Unwrap() error { return ErrAlreadyScanning }

// ErrRootNotADirectory is a configured root that exists and is a file, a
// socket or a device.
var ErrRootNotADirectory = errors.New("the library root is not a directory")

// ErrUnavailableRoot is 003 §3.8's *"an unavailable root is not an empty
// root"*, in the form the section states: a root that **cannot be read**.
var ErrUnavailableRoot = errors.New("a library root could not be read")

// UnavailableRootError is guard 1 of 003 plan §6.5, and it covers all three of
// plan §7's reading rows at once: a root that does not exist, a root that is a
// directory and cannot be read, and an error anywhere inside the walk.
//
// The three are one error because they have one answer — *"the library's scan
// fails and nothing is written"* — and because the distinction between them is
// not one the caller acts on. What the caller needs is which library, which
// root, and the underlying failure to show an operator, and all three are here.
type UnavailableRootError struct {
	LibraryID   string
	LibraryName string

	// Root is the root's ordinal in `ports.Library.Roots`, and Path is the
	// configured path. Both, because an operator with three roots configured
	// needs to know which one and a message carrying only the ordinal makes
	// them count.
	Root int
	Path string

	Err error
}

func (e *UnavailableRootError) Error() string {
	return fmt.Sprintf("scan: library %q (%s): root %d (%s) could not be read, so this scan changes nothing: %v",
		e.LibraryName, e.LibraryID, e.Root, e.Path, e.Err)
}

func (e *UnavailableRootError) Unwrap() error { return e.Err }

// Is reports [ErrUnavailableRoot] as well as the underlying cause, so that a
// caller can match the *class* of failure without knowing which of plan §7's
// three rows produced it.
func (e *UnavailableRootError) Is(target error) bool { return target == ErrUnavailableRoot }

// ErrEmptyRoot is guard 2 of 003 plan §6.5 and the refusal half of AC-16.
var ErrEmptyRoot = errors.New("a library root reads as holding no candidate file where the last scan saw some")

// EmptyRootError is the root that read as holding nothing where the previous
// scan of this library recorded at least one file (003 §3.8's 2026-09-05
// amendment, AC-16).
//
// **It names the root**, which the criterion requires: a share that fails to
// mount is very often perfectly readable — the mount point is an ordinary empty
// directory — so the operator's next act is to look at a path, and a refusal
// that made them guess which of three would be a refusal they route around.
//
// Zero is the threshold because it needs no number. It is the mount failure's
// signature, where *"fewer than before"* is a judgement nobody could defend,
// and a root an operator really emptied is a deliberate act — which is what
// `--allow-empty-root` is for.
type EmptyRootError struct {
	LibraryID   string
	LibraryName string
	Root        int
	Path        string

	// PreviousFiles is how many files the previous scan recorded under this
	// root. It is in the message because it is the whole evidence for the
	// refusal: an operator reading *"0 candidate files now, 4,312 before"*
	// knows immediately whether this is a mount that failed or a directory
	// they emptied.
	PreviousFiles int
}

func (e *EmptyRootError) Error() string {
	return fmt.Sprintf(
		"scan: library %q (%s): root %d (%s) holds no candidate file and the last scan recorded %d, "+
			"so it is treated as unavailable rather than as emptied and this scan changes nothing: "+
			"pass --allow-empty-root if you meant it",
		e.LibraryName, e.LibraryID, e.Root, e.Path, e.PreviousFiles)
}

func (e *EmptyRootError) Unwrap() error { return ErrEmptyRoot }

// DefaultBatchSize is how many items one committed step of a scan writes.
//
// 003 plan §6.9: *"batches are sized in items, not in time"*. The number is a
// trade between round trips and how long one write transaction holds SQLite's
// write lock, and ADR-0003 measured the second half — 57,664 reads completed
// during one transaction inserting 30,000 rows, worst read latency 393 µs
// `[measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02]`. Five
// hundred is two orders of magnitude inside that, so the batching keeps that
// measurement relevant rather than replacing it with a hope about a transaction
// held open for a whole tree.
const DefaultBatchSize = 500

// DefaultStaleAfter is how old a claim has to be before a scan breaks it.
//
// 003 plan §6.9 argues the shape of this number rather than the number: the
// claim is renewed on every committed batch, so it *"only has to exceed the
// time between two batches"* and never the time a whole library takes. Two
// minutes exceeds one transaction over [DefaultBatchSize] items by orders of
// magnitude on any storage this project targets, and it also bounds the other
// cost: a scanner killed between two batches leaves a claim, and two minutes is
// how long that library is unscannable afterwards.
//
// **This is what forces the claim to be taken after the reading rather than
// before it**; see [Scanner.Scan].
const DefaultStaleAfter = units.Ticks(2 * 60 * 10_000_000)

// Config is what a scan needs that is not the library.
type Config struct {
	// Items is the derived half. Every write this feature performs goes
	// through it.
	Items ports.ItemStore

	// Clock stamps the claim, every renewal and the release.
	Clock ports.Clock

	// ClaimedBy is how this scanner names itself in the claim, and it is what
	// the *other* scanner's refusal prints. It is required: an empty claimant
	// makes 003 plan §7's two messages name nobody.
	ClaimedBy string

	// StaleAfter is how old a claim has to be before it is broken. Zero means
	// [DefaultStaleAfter].
	StaleAfter units.Ticks

	// BatchSize is how many items one committed step writes. Zero means
	// [DefaultBatchSize].
	BatchSize int

	// Logger receives the progress 003 §3.8 asks for beside the summary: the
	// reason for every skipped path and every unplaceable item, at debug,
	// because those are per-path and a summary document holding every skipped
	// path of a large library is not a summary. Nil discards them.
	Logger *slog.Logger

	// Tags is 004's seam. Nil is `library.NoTags`, which is what v1 ships.
	Tags library.TagSource
}

// Scanner performs 003's act.
type Scanner struct {
	items      ports.ItemStore
	clock      ports.Clock
	claimedBy  string
	staleAfter units.Ticks
	batchSize  int
	logger     *slog.Logger
	tags       library.TagSource
}

// New builds a scanner, and refuses a configuration that cannot produce this
// feature's messages.
func New(config Config) (*Scanner, error) {
	switch {
	case config.Items == nil:
		return nil, errors.New("scan: a scanner needs an item store")
	case config.Clock == nil:
		return nil, errors.New("scan: a scanner needs a clock")
	case config.ClaimedBy == "":
		return nil, errors.New("scan: a scanner needs a name to claim libraries under")
	case config.StaleAfter < 0:
		return nil, fmt.Errorf("scan: staleAfter is %d, which is not a duration", config.StaleAfter)
	case config.BatchSize < 0:
		return nil, fmt.Errorf("scan: the batch size is %d", config.BatchSize)
	}

	scanner := &Scanner{
		items:      config.Items,
		clock:      config.Clock,
		claimedBy:  config.ClaimedBy,
		staleAfter: config.StaleAfter,
		batchSize:  config.BatchSize,
		logger:     config.Logger,
		tags:       config.Tags,
	}
	if scanner.staleAfter == 0 {
		scanner.staleAfter = DefaultStaleAfter
	}
	if scanner.batchSize == 0 {
		scanner.batchSize = DefaultBatchSize
	}
	if scanner.logger == nil {
		scanner.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if scanner.tags == nil {
		scanner.tags = library.NoTags{}
	}
	return scanner, nil
}

// Options is what an operator asked of one scan.
type Options struct {
	// Full is 003 §3.8's re-examination, and it changes exactly one thing:
	// whether an unchanged signal is believed (plan §6.4, [Reconcile]).
	Full bool

	// AllowEmptyRoot is the operator saying they meant it, and it is the
	// second half of AC-16. It disables guard 2 and **nothing else** — a root
	// that cannot be read is still a failed scan, because "I emptied this
	// directory" is not "I unmounted this share".
	AllowEmptyRoot bool
}

// Scan reads every root of one library and makes the store hold what it found.
//
// # The order, which is the whole of 003 plan §6.5
//
//	read what the store holds
//	read every root                      guard 1: any failure ends the scan here
//	check no root read as empty          guard 2: AC-16, unless the operator said they meant it
//	Resolve, then Reconcile              over the *whole* library, never a root at a time
//	claim the library
//	write the additions and updates      in batches, each renewing the claim
//	apply the removals                   guard 3: reached only once every root succeeded
//	release the claim and record what happened
//
// **Guard 3 is the order rather than a check.** A removal is computed from a
// complete reading of every root and is applied after every batch, so a scan
// that was cancelled, that hit a failed batch, or whose *second* root failed
// while the first succeeded removes nothing at all. That is the state a
// per-root reconciliation gets wrong and this one cannot: [Reconcile] takes
// whole sets, and a library whose second root failed never reaches it.
//
// # The claim is taken after the reading, and 003 plan §6.9's own argument is
// # what forces that
//
// §6.9 defends `staleAfter` by saying the claim is renewed on every committed
// batch, so the value *"only has to exceed the time between two batches"* and
// never the time a whole scan takes. Nothing renews a claim during a walk —
// there is no renewal outside a batch, on purpose — so a claim taken before the
// reading would make `staleAfter` have to exceed the walk of the largest
// library an operator has, which is the guess §6.9 exists to avoid.
//
// Two consequences, both stated rather than discovered. Two scanners may walk
// one library at once and only one of them writes, which costs a walk and
// nothing else. And a reconciliation computed against a reading another scanner
// has since changed is **refused rather than applied**: every identifier in
// `Remove` came from this store, so one that now matches no row makes
// `RemoveItems` fail its rows-affected check (001's rule) and nothing is
// removed.
//
// A refusal by either guard therefore leaves no claim behind at all, which is
// what lets an operator fix a mount and scan again immediately. A failure
// *after* the claim — a batch that would not commit — leaves it to go stale,
// which is 003 plan §7's row and is bounded by [DefaultStaleAfter].
func (s *Scanner) Scan(ctx context.Context, lib ports.Library, options Options) (Changes, error) {
	collection := library.CollectionType(lib.CollectionType)
	if !collection.Valid() {
		return Changes{}, fmt.Errorf("scan: library %q (%s): %w: %q",
			lib.Name, lib.ID, library.ErrCollectionTypeUnknown, lib.CollectionType)
	}

	previous, err := s.items.ItemsForLibrary(ctx, lib.ID)
	if err != nil {
		return Changes{}, err
	}

	readings, examined, skipped, err := s.read(ctx, lib, collection)
	if err != nil {
		return Changes{}, err
	}

	if !options.AllowEmptyRoot {
		if err := guardAgainstAnEmptyRoot(lib, readings, previous); err != nil {
			return Changes{}, err
		}
	}

	plan, err := library.ResolveWithTags(lib, readings, s.tags)
	if err != nil {
		return Changes{}, fmt.Errorf("scan: library %q (%s): %w", lib.Name, lib.ID, err)
	}
	for _, note := range plan.Unplaceable {
		s.logger.Debug("unplaceable",
			"library", lib.Name, "root", note.Root, "path", note.Path, "reason", note.Reason)
	}

	reconciliation, err := Reconcile(previous, plan.Items, options.Full)
	if err != nil {
		return Changes{}, fmt.Errorf("scan: library %q (%s): %w", lib.Name, lib.ID, err)
	}

	won, displaced, err := s.items.ClaimScan(ctx, lib.ID, s.claimedBy, s.clock.Now(), s.staleAfter)
	if err != nil {
		return Changes{}, err
	}
	if !won {
		return Changes{}, &AlreadyScanningError{LibraryID: lib.ID, LibraryName: lib.Name, ClaimedBy: displaced}
	}
	if displaced != "" {
		// 003 plan §7's row: broken and taken, with a log line naming the
		// previous claimant. It is the only place that name can be printed —
		// after the call the row names this scanner.
		s.logger.Warn("broke a scanning claim that had gone stale",
			"library", lib.Name, "previous-claimant", displaced)
	}

	changes := Changes{
		Added:       nonNil(reconciliation.Added),
		Updated:     nonNil(reconciliation.Updated),
		Removed:     nonNil(reconciliation.Remove),
		Examined:    examined,
		Skipped:     skipped,
		Unplaceable: len(plan.Unplaceable),
	}

	if err := s.write(ctx, lib, reconciliation.Write); err != nil {
		return Changes{}, err
	}

	// After every batch, and never batched itself: 003 plan §6.9. A scan that
	// died before this point has added and updated some items and removed
	// none, which is the only partial state this feature can leave behind.
	if err := s.items.RemoveItems(ctx, reconciliation.Remove); err != nil {
		return Changes{}, err
	}

	document, err := changes.Document()
	if err != nil {
		return Changes{}, err
	}
	if err := s.items.ReleaseScan(ctx, lib.ID, s.clock.Now(), document, options.Full); err != nil {
		return Changes{}, err
	}

	s.logger.Info("scanned",
		"library", lib.Name,
		"added", len(changes.Added), "updated", len(changes.Updated), "removed", len(changes.Removed),
		"examined", changes.Examined, "skipped", changes.Skipped, "unplaceable", changes.Unplaceable,
		"full", options.Full)
	return changes, nil
}

// read walks every root of the library and applies guard 1 to each.
//
// It returns the readings, the number of candidate files across all of them,
// and the number of refusals — the two counts 003 §3.8's summary carries that a
// reconciliation cannot know about.
//
// **Nothing partial ever comes back.** The first root that fails ends this, and
// the caller has no reading to be tempted into reconciling against, which is
// the same shape [Walk] already gives one root.
func (s *Scanner) read(ctx context.Context, lib ports.Library,
	collection library.CollectionType) ([]library.Reading, int, int, error) {
	readings := make([]library.Reading, 0, len(lib.Roots))
	examined, skipped := 0, 0

	for ordinal, root := range lib.Roots {
		// The one place a walk consults the context, added at T14 when the
		// entry layer gained a scheduled scan the server's own shutdown
		// cancels. [Walk] takes an `fs.FS` and no context — io/fs offers none —
		// so what a cancellation can bound is the *next* root rather than the
		// current one, and a stop therefore waits out one root's walk and no
		// more. Nothing is written either way: every failure here is guard 1's,
		// and guard 3 is the ordering that keeps a removal behind it.
		if err := ctx.Err(); err != nil {
			return nil, 0, 0, err
		}

		unavailable := func(err error) error {
			return &UnavailableRootError{
				LibraryID: lib.ID, LibraryName: lib.Name, Root: ordinal, Path: root, Err: err,
			}
		}

		// Before the walk, because plan §6.5's first guard is that the root
		// *"must resolve to a readable directory before the walk starts"* and
		// because `os.DirFS` of a path that does not exist is a perfectly
		// ordinary value whose first read fails — a distinction worth keeping
		// out of the message an operator sees.
		info, err := os.Stat(root)
		if err != nil {
			return nil, 0, 0, unavailable(err)
		}
		if !info.IsDir() {
			return nil, 0, 0, unavailable(ErrRootNotADirectory)
		}

		result, err := Walk(os.DirFS(root), ordinal, collection)
		if err != nil {
			return nil, 0, 0, unavailable(err)
		}

		for _, note := range result.Skipped {
			s.logger.Debug("skipped",
				"library", lib.Name, "root", note.Root,
				"path", filepath.Join(root, filepath.FromSlash(note.Path)), "reason", note.Reason)
		}

		readings = append(readings, result.Reading)
		examined += len(result.Reading.Entries)
		skipped += len(result.Skipped)
	}
	return readings, examined, skipped, nil
}

// write applies the additions and updates in batches, each of which renews the
// claim as part of its own transaction (003 plan §6.9).
//
// **There is deliberately no renewal outside a batch.** `ports.ItemStore`
// offers none, and that absence is the design: a renewal that committed
// separately would be a scanner that could report progress it did not make, or
// hold a claim over work it had abandoned. So a batch that fails leaves the
// claim exactly where the previous committed batch put it.
func (s *Scanner) write(ctx context.Context, lib ports.Library, items []ports.ScannedItem) error {
	for start := 0; start < len(items); start += s.batchSize {
		end := min(start+s.batchSize, len(items))
		batch := ports.ScanBatch{
			LibraryID: lib.ID,
			Items:     items[start:end],
			ClaimedBy: s.claimedBy,
			At:        s.clock.Now(),
		}
		if err := s.items.ApplyScanBatch(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}

// guardAgainstAnEmptyRoot is guard 2 of 003 plan §6.5 and the refusal half of
// AC-16.
//
// The comparison is against **files** and not against items, because a library
// that holds items and no files is exactly what a previous refusal leaves
// behind: the library's own `CollectionFolder` row has no file, and neither
// does an inferred container. Counting items would make a library whose every
// file had already gone refuse for ever.
func guardAgainstAnEmptyRoot(lib ports.Library, readings []library.Reading, previous []ports.ScannedItem) error {
	for _, reading := range readings {
		if len(reading.Entries) > 0 {
			continue
		}
		recorded := filesRecordedAtRoot(previous, reading.Root)
		if recorded == 0 {
			continue
		}
		return &EmptyRootError{
			LibraryID:     lib.ID,
			LibraryName:   lib.Name,
			Root:          reading.Root,
			Path:          lib.Roots[reading.Root],
			PreviousFiles: recorded,
		}
	}
	return nil
}

// filesRecordedAtRoot counts the files the previous scan recorded under one
// root of a library.
func filesRecordedAtRoot(previous []ports.ScannedItem, ordinal int) int {
	files := 0
	for _, item := range previous {
		if item.RootOrdinal != ordinal {
			continue
		}
		files += len(item.Files)
	}
	return files
}

// nonNil makes an empty list an empty list rather than a null, so that a
// summary document says *"nothing was added"* in the same shape whether or not
// anything was.
func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
