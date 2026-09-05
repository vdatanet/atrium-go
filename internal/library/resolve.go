package library

import (
	"errors"
	"fmt"
	"sort"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// Entry is one candidate file a walk saw: its root-relative path, its size and
// when it last changed.
//
// It is what `internal/scan` produces and what this package consumes, and it
// carries nothing a resolver cannot use — no absolute path, no file handle and
// no directory entry. Path is slash-separated and of the shape [io/fs] yields:
// no leading separator, no `.` and no `..` elements.
type Entry struct {
	Path       string
	Size       int64
	ModifiedAt units.Time
}

// Reading is what one walk of one root saw, sorted.
//
// Root is the ordinal of the library root the paths are relative to, and it is
// carried rather than the root's path so that a library whose roots moved
// resolves to the same identifiers (003 §3.6).
type Reading struct {
	Root    int
	Entries []Entry
}

// Note is one path a resolution did not turn into an ordinary item, with the
// reason in the terms 003 §3.2 and §3.8 state them in.
//
// The two lists it appears in are counted apart on purpose: 003 §3.8 requires
// a scan to report a skipped file and an unplaceable one separately, "because
// an operator told that both were skipped would go looking for something that
// is not missing".
type Note struct {
	// Root is the reading the path came from.
	Root int

	// Path is the entry's path, exactly as the reading carried it.
	Path string

	// Reason says which rule the path met.
	Reason string
}

// Plan is the whole of what a library resolves to: every item, and a note for
// every path that became none.
//
// It is 003 plan §5's record, and the whole reason [Resolve] returns one rather
// than answering a path at a time. Items is sorted, and so is every list a map
// inside the resolver produced (Principle VII, and plan §5's own clause: the
// parent-child assignment of an inferred container and the part order of a
// multi-part film both reach a response body at 005).
type Plan struct {
	// Items is every item, sorted by root ordinal, then path, then
	// identifier. The library's own `CollectionFolder` row has no path and
	// sorts first.
	Items []ports.ScannedItem

	// Unplaceable is a path that became an item whose name says too little to
	// place it (003 §3.8). Under `tvshows` there are two ways to earn one,
	// [ReasonNoEpisodeNumber] and [ReasonNoSeries].
	//
	// **Neither `movies` nor `music` produces one**, and the reason is the
	// same for both: a film's name and a track's name never have to say
	// anything for the item to be placed. A track directly under a library
	// root has no album and no artist, and is still a complete item hanging
	// from the library's own row — where an episode with no episode number is
	// a file nothing can say which episode it is.
	//
	// **It is not [Plan.Skipped] and the two must not be added together.** A
	// skipped path is not in the library and an unplaceable one is: it has an
	// item, an identifier and a parent, and everything a client can do with an
	// item it can do with this one.
	Unplaceable []Note

	// Skipped is a path [Resolve] refused. See [Resolve]'s own comment on why
	// it is normally empty and why it must not be added to the walk's count.
	Skipped []Note
}

// ErrCollectionTypeUnknown is the library whose collection type is not one of
// 003 §3.1's three names at all.
var ErrCollectionTypeUnknown = errors.New("library: the collection type is not one of the three")

// Resolve turns every root's reading of one library into the items that library
// holds.
//
// # Why it takes every reading at once
//
// Because three of 003 §3's rules cannot be decided from one path, and 003 plan
// §5 is where that is argued: a directory holding several different titles is a
// category rather than a film (§3.3), a multi-part film is one item only once
// its siblings have been seen (§3.3), and an album's identity comes from the
// album artist across all of its tracks (§3.5). A resolver answering a path at a
// time would have to discover each of those by rewriting what it had already
// returned, and what a scan answered would then depend on the order the
// directory entries arrived in — which is exactly what Principle VII forbids.
//
// # Determinism
//
// The readings are copied and sorted — by root ordinal, and then by path within
// each root — before anything looks at them, and the items are sorted before
// they leave. No map is ever iterated to produce output: every grouping keyed by
// a map is walked in the sorted entry order and read out of the map by key. Two
// readings of one tree whose directory entries were created in opposite orders
// therefore produce the identical plan, part order included.
//
// # What it refuses
//
// A path that will not normalise fails the **whole library** and returns the
// [PathError] naming it — 003 plan §7's row, and never a partial plan. A
// collection type that is not one of the three is [ErrCollectionTypeUnknown].
//
// A path that is not a candidate under this library's collection type is a
// [Plan.Skipped] note rather than an error, and it is normally impossible: the
// walk applies [CollectionType.Candidate] already and hands only candidates
// here. The check is here because Resolve is a pure exported function with no
// walk behind it in a test or in a later feature, and making a `Movie` called
// `poster` out of a `poster.jpg` somebody passed in is worse than a note. It is
// the **same rule** as the walk's, not a second one, so a scan summary must
// report the walk's count and not the sum of the two.
func Resolve(lib ports.Library, readings []Reading) (Plan, error) {
	return ResolveWithTags(lib, readings, NoTags{})
}

// ResolveWithTags is [Resolve] with a metadata reader plugged into it, and it
// is the seam feature 004 fills.
//
// 003 §3.5 makes a music file's **embedded tags outrank its path** and says in
// terms that reading them is 004's, and 003 plan §6.2 settles the ordering of
// that conversation: the source is consulted **once per file, before
// grouping**, because the album artist decides which album a track belongs to.
// [Resolve] is this function with the [NoTags] source, which is what v1 ships
// — so the fallback path and the tag-driven path are the same code with one
// different collaborator rather than two behaviours with one tested.
//
// The source is consulted only for a `music` library. The other two collection
// types take everything from the path, which is 003 §3.3's and §3.4's own
// shape and not an omission here: 004 replaces a *name* on those, and it does
// so through `ports.ScannedItem.SortTitle` and its own metadata pass rather
// than through this seam.
func ResolveWithTags(lib ports.Library, readings []Reading, tags TagSource) (Plan, error) {
	collection := CollectionType(lib.CollectionType)
	if !collection.Valid() {
		return Plan{}, fmt.Errorf("%w: %q", ErrCollectionTypeUnknown, lib.CollectionType)
	}
	if tags == nil {
		tags = NoTags{}
	}

	sorted := sortReadings(readings)
	kept, skipped := partitionCandidates(collection, sorted)

	root := libraryRootItem(lib)
	plan := Plan{Items: []ports.ScannedItem{root}, Skipped: skipped}

	switch collection {
	case Movies:
		items, err := resolveMovies(lib, kept, root.ID)
		if err != nil {
			return Plan{}, err
		}
		plan.Items = append(plan.Items, items...)

	case Shows:
		items, unplaceable, err := resolveShows(lib, kept, root.ID)
		if err != nil {
			return Plan{}, err
		}
		plan.Items = append(plan.Items, items...)
		plan.Unplaceable = unplaceable

	case Music:
		items, err := resolveMusic(lib, kept, tags, root.ID)
		if err != nil {
			return Plan{}, err
		}
		plan.Items = append(plan.Items, items...)
	}

	sortItems(plan.Items)
	return plan, nil
}

// partitionCandidates applies the walk's own predicate a second time and
// separates what it keeps from what it refuses. See [Resolve] on why the
// second application is here and why its count is not the scan's.
func partitionCandidates(collection CollectionType, readings []Reading) ([]Reading, []Note) {
	kept := make([]Reading, len(readings))
	var skipped []Note
	for i, reading := range readings {
		entries := make([]Entry, 0, len(reading.Entries))
		for _, entry := range reading.Entries {
			if ok, skip := collection.Candidate(entry.Path); !ok {
				skipped = append(skipped, Note{
					Root:   reading.Root,
					Path:   entry.Path,
					Reason: skip.String(),
				})
				continue
			}
			entries = append(entries, entry)
		}
		kept[i] = Reading{Root: reading.Root, Entries: entries}
	}
	return kept, skipped
}

// libraryRootItem is the library's own `CollectionFolder` row.
//
// Its key is the library's **configured identity** and not a path (003 §3.6's
// table), which is why it is the one identifier in this feature that does not
// go through [Normalise]: normalisation reduces a path or a name to one form,
// and a library's identity has no other form to be reduced from. It is
// allocated once and kept, so nothing about it moves when a root does.
//
// The row deliberately has no path. A library may be configured with several
// roots and no one of them is the library (003 plan §4.2).
func libraryRootItem(lib ports.Library) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:        DeriveID(lib.ID, KindCollectionFolder, lib.ID),
		LibraryID: lib.ID,
		Type:      string(KindCollectionFolder),
		Name:      lib.Name,
	}
	item.SortKey = SortKeyFor(&item)
	return item
}

// sortReadings copies the readings and sorts them, and sorts each one's
// entries, so that the caller's slices are never reordered underneath it and
// nothing downstream depends on the order a walk happened to yield.
func sortReadings(readings []Reading) []Reading {
	out := make([]Reading, len(readings))
	for i, reading := range readings {
		entries := make([]Entry, len(reading.Entries))
		copy(entries, reading.Entries)
		sort.SliceStable(entries, func(a, b int) bool {
			return entries[a].Path < entries[b].Path
		})
		out[i] = Reading{Root: reading.Root, Entries: entries}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Root < out[b].Root })
	return out
}

// sortItems puts a plan's items in the one order every caller sees: root
// ordinal, then path, then identifier.
//
// The identifier is the last key rather than the first because a list ordered
// by a digest is unreadable in a failure message, and it is there at all
// because two items can share a path — a library's own row has none, and 003
// plan §6.3 records that two canonically equal filenames derive one key.
func sortItems(items []ports.ScannedItem) {
	sort.SliceStable(items, func(a, b int) bool {
		if items[a].RootOrdinal != items[b].RootOrdinal {
			return items[a].RootOrdinal < items[b].RootOrdinal
		}
		if items[a].Path != items[b].Path {
			return items[a].Path < items[b].Path
		}
		return items[a].ID < items[b].ID
	})
}
