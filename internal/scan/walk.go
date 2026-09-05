// Package scan is 003's act: the walk that reads a library root, the
// reconciliation of what it read against what the last scan stored, the guards
// that keep an unmounted share from reading as an empty library, and the
// batching and summary of 003 §3.8.
//
// It imports `internal/library` and never the other way round (003 plan §3):
// the domain is a function over paths and names that 004 and 005 read without
// wanting a walker, and the walker is called by exactly two callers.
//
// # What lives here so far
//
// [Walk], of 003 plan §6.1 — one walk of one library root, over an [fs.FS].
// The reconciliation, the guards and the batching arrive with their own tasks.
package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/units"
)

// IgnoreMarker is the filename an operator writes their own exclusion as
// (003 §3.2).
//
// It is exported because an operator-visible filename is a name a message has
// to be able to spell, and a second spelling of it somewhere else is a second
// thing to keep in step.
const IgnoreMarker = ".ignore"

// Result is one walk of one library root: the reading a resolver consumes, and
// a note for every path a rule refused.
//
// The two are returned together rather than separately because 003 §3.8's
// summary reports both — *"files examined, files skipped with the reason"* —
// and because a caller that had to ask twice could ask once.
//
// **Skipped is the scan's skip count, and [library.Plan]'s is not.**
// [library.Resolve] applies [library.CollectionType.Candidate] a second time
// over the same paths, as the same rule rather than a second one, so adding the
// two together counts a refusal twice.
type Result struct {
	// Reading is every candidate file this root holds, sorted on the path.
	Reading library.Reading

	// Skipped is every path a rule refused, sorted on the path. A rule that
	// refuses a whole subtree — a hidden directory, an `.ignore` marker —
	// contributes **one** note, naming the directory, and nothing for the
	// files under it: the walk does not descend, so it never saw them.
	Skipped []library.Note
}

// Walk reads one root of one library and returns everything under it that is a
// candidate media file.
//
// fsys is the root, and every caller in this project builds it with
// [os.DirFS]. The paths in the result are relative to it, slash-separated, and
// of the shape [io/fs] yields — which is exactly the shape
// [library.CollectionType.Candidate] and [library.Normalise] expect.
// rootOrdinal is the library root's position in [ports.Library.Roots], carried
// through into the reading so that a library whose roots moved resolves to the
// same identifiers (003 §3.6).
//
// # What it refuses, and where each rule is decided
//
// The path-shaped rules of 003 §3.2 — a hidden component, an extras folder
// name, an extras suffix, an extension not on this collection type's list — are
// [library.CollectionType.Candidate]'s, asked here per file. Four more need
// something a path does not carry and are this function's:
//
//   - **A hidden directory is not descended into.** A file under one is refused
//     by Candidate anyway, because Candidate reads every component, so the
//     reading is the same either way; what differs is that the walk never opens
//     the directory and never reports the files inside it one by one.
//   - **The `.ignore` marker.** An empty or whitespace-only marker excludes the
//     directory holding it and everything beneath, and the search runs from a
//     file's directory up to **the library root and no further**. Both halves
//     fall out of pruning the subtree at the directory that carries the marker,
//     and the second falls out of fsys being the root: there is nothing above it
//     to look at. See [Walk]'s divergence note below.
//   - **A zero-byte file**, 003 §3.2's incomplete copy, which needs a size.
//   - **An entry that is not a regular file once followed** — a symbolic link
//     to a directory, a device node, a socket, and a link pointing at nothing.
//     A link to a *file* is followed and is the file it points at, size
//     included, which is a shape 003 §3.2 does not mention and a walk cannot
//     avoid taking a position on.
//
// 003 §3.2's last rule — *"files being written, detected by size change between
// two passes"* — is **not** here and cannot be. It is a property of a pair of
// scans rather than of a file, so it is decided where both readings exist
// (003 plan §6.1, §6.4), and what v1 does is narrower than the row suggests: a
// half-copied file becomes an item with the wrong size and the next scan
// corrects it.
//
// # Two deliberate divergences from the reference, and one accepted shortfall
//
//   - **A marker above the library root excludes nothing.** The reference
//     searches upwards to the *filesystem* root
//     `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:41-68 @ v10.11.11]`,
//     so a stray `.ignore` in a home directory empties every library beneath it.
//     That is a foot-gun rather than a feature, and diverging shows more rather
//     than less, which is the safe direction for a scanner
//     ([behaviours §3.0.3]). U-42, and a source reading rather than a
//     measurement: no probe in this project has sent a `.ignore` file of any
//     kind.
//   - **A zero-byte file is no item.** The reference makes one
//     `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`,
//     and this is one of the forty-seven differences 003 declares over its own
//     fixture tree.
//   - **A non-empty marker excludes nothing.** At the reference it is a set of
//     `.gitignore`-style patterns of which only the matches are excluded, with
//     the fallback that a marker whose every pattern fails to parse excludes
//     everything. Implementing that means a matcher this project would own for a
//     shape no measurement shows anybody using, and getting it subtly wrong
//     hides files an operator expects to see. Accepted shortfall (U-42, 003 plan
//     §6.1), asserted rather than commented.
//
// # Determinism, and errors
//
// The reading is **sorted on the path before it leaves**, because [fs.WalkDir]
// yields a directory's children before a sibling file that sorts ahead of the
// directory's own name, and because a caller may hand this function an [fs.FS]
// whose ReadDir does not sort at all. Principle VII, at the layer where a
// filesystem's own ordering could still reach the answer.
//
// **Any error fails the whole walk** and returns no partial reading: a
// permission refused deep in a tree, an I/O error, an unreadable root. 003
// §3.8's rule is that an unavailable root is not an empty one, and *"treating an
// unmounted share as 'every item was deleted' is the single most destructive
// thing a scanner can do"*. A caller must not reconcile against what a failed
// walk returned, and there is nothing here to be tempted by.
//
// [behaviours §3.0.3]: ../../docs/compatibility/behaviours.md#303-the-shape-of-a-safe-divergence
// [ports.Library.Roots]: ../ports/libraries.go
func Walk(fsys fs.FS, rootOrdinal int, collection library.CollectionType) (Result, error) {
	if !collection.Valid() {
		// Not a refusal per file: an unknown type admits nothing, so a walk
		// that carried on would read a whole library as empty and hand a
		// reconciliation the deletion of everything in it.
		return Result{}, fmt.Errorf("scan: root %d: %q: %w", rootOrdinal, collection, library.ErrCollectionTypeUnknown)
	}

	var entries []library.Entry
	var skipped []library.Note
	note := func(p string, reason library.Skip) {
		skipped = append(skipped, library.Note{Root: rootOrdinal, Path: p, Reason: reason.String()})
	}

	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// The root itself is never hidden — it is `.`, which is path
			// syntax and not a name — but it *can* carry the marker, and
			// that is the inclusive end of "up to the library root".
			if p != "." && library.IsHiddenName(path.Base(p)) {
				note(p, library.SkipHidden)
				return fs.SkipDir
			}
			excludes, err := markerExcludes(fsys, p)
			if err != nil {
				return err
			}
			if excludes {
				note(p, library.SkipIgnoreMarker)
				return fs.SkipDir
			}
			return nil
		}

		if ok, reason := collection.Candidate(p); !ok {
			note(p, reason)
			return nil
		}

		// fs.Stat and not d.Info(): Info reports on the directory entry
		// itself, so a symbolic link to a film would be an item whose size
		// is the length of the target's path. Stat follows the link, which
		// is what the reference does, and what is still not a regular file
		// afterwards — a link to a directory, a device node, a socket — is
		// not a media file.
		info, err := fs.Stat(fsys, p)
		if errors.Is(err, fs.ErrNotExist) {
			// A link pointing at nothing, or a file that was removed
			// between the directory read and this one. Neither is an
			// unreadable root, and failing a library's whole scan over
			// one would mean a download moved while a scan ran costs the
			// operator every item in that library.
			note(p, library.SkipNotARegularFile)
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			note(p, library.SkipNotARegularFile)
			return nil
		}
		if info.Size() == 0 {
			note(p, library.SkipZeroBytes)
			return nil
		}

		entries = append(entries, library.Entry{
			Path:       p,
			Size:       info.Size(),
			ModifiedAt: units.At(info.ModTime()),
		})
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("scan: reading root %d: %w", rootOrdinal, err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Path < skipped[j].Path })

	return Result{
		Reading: library.Reading{Root: rootOrdinal, Entries: entries},
		Skipped: skipped,
	}, nil
}

// markerExcludes reports whether the directory dir carries an [IgnoreMarker]
// that excludes it.
//
// Only an **empty or whitespace-only** marker excludes anything (003 §3.2's
// 2026-09-05 amendment). A marker holding anything else is the reference's
// pattern list, which v1 does not implement and which therefore excludes
// nothing.
//
// A marker that is not a regular file — a directory somebody created called
// `.ignore`, a socket — excludes nothing and is not an error. It is not the
// operator's exclusion, and failing a whole library's scan over it would be a
// worse answer than reading past it.
func markerExcludes(fsys fs.FS, dir string) (bool, error) {
	marker := path.Join(dir, IgnoreMarker)

	info, err := fs.Stat(fsys, marker)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}

	// The marker is an operator's note to a scanner, not a media file: a
	// pattern list is a handful of lines. Reading it whole is what lets the
	// emptiness be decided on the bytes rather than on the length reported
	// for it.
	content, err := fs.ReadFile(fsys, marker)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(content)) == "", nil
}
