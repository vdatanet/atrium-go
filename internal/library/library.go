// Package library is 003's domain: the rules that turn a path and a name into
// an item, with no filesystem, no store and no HTTP anywhere in it.
//
// It is read by features that never scan anything — 004 merges metadata over
// the names and numbers derived here, and 005 orders every list by the sort key
// computed here — so the dependency runs one way and never back (003 plan §3).
// Nothing in this package opens a file. `internal/scan` performs the walk and
// calls in here for every decision a single path can settle.
//
// # What lives here so far
//
// The collection types of 003 §3.1, the measured extension lists of §3.2, and
// the three exclusions that can be decided from one path: a hidden component,
// an extras folder name and an extras suffix (003 plan §6.1). Normalisation and
// identity are in identity.go — [Normalise] and [DeriveID], of §3.6 and plan
// §6.3, which every item this feature produces derives its identifier through.
// The two sort-name derivations of §3.7 are in sortkey.go — [SortKeyFor], which
// every item is keyed with, and [SortKeyBase], which a caller holding a bare
// name uses; that split is the whole of what keeps the wrong one of the two out
// of reach. The resolvers arrive with their own tasks.
package library

import (
	"path"
	"strings"
)

// CollectionType is a library's collection type.
//
// It is not a hint. It selects which resolution rules apply, and a file under a
// music root is never resolved as a movie no matter what it is called
// (003 §3.1). Mixed-content roots are not supported in v1.
type CollectionType string

// The three names of 003 §3.1, spelled exactly as an operator configures them
// and exactly as the reference's own collection_type reads.
const (
	Movies CollectionType = "movies"
	Shows  CollectionType = "tvshows"
	Music  CollectionType = "music"
)

// AllCollectionTypes returns the three, in the order 003 §3.2's table gives
// them.
//
// It returns a fresh slice on every call: a package-level slice is a value a
// caller can sort or write into, and a list that changed under one reader
// because another had reordered it is the least debuggable failure available.
func AllCollectionTypes() []CollectionType {
	return []CollectionType{Movies, Shows, Music}
}

// ParseCollectionType turns an operator's string into a collection type,
// reporting whether it is one of the three.
//
// The match is exact. Accepting "Movies" or "TVShows" would be a leniency this
// project owns and the reference does not, and the caller that wants to be
// forgiving can fold before it asks — where the forgiveness is visible.
func ParseCollectionType(s string) (CollectionType, bool) {
	for _, c := range AllCollectionTypes() {
		if CollectionType(s) == c {
			return c, true
		}
	}
	return "", false
}

// Valid reports whether c is one of 003 §3.1's three names.
func (c CollectionType) Valid() bool {
	_, ok := ParseCollectionType(string(c))
	return ok
}

// admittedExtensions is 003 §3.2's measured table, verbatim and in its order.
//
//	movies   .mkv .mp4 .avi .ts
//	tvshows  .mkv .avi .mp4
//	music    .flac .m4a .dsf
//
// `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`
//
// It is a measured **lower bound** — what one real library of 8,288 items
// contained — and not the reference's configured lists, which the API does not
// expose. An extension nobody has a file of was not measured and its absence
// here is not evidence of refusal (003 §3.2).
//
// These lists are code and not configuration: they are a measured contract
// ([behaviours §2.15]), not an operator preference, and a per-library override
// would be a configuration surface with no named consumer, which Principle VI
// forbids (003 plan §4.3). There is deliberately nothing to configure them
// with.
//
// The map is never iterated. Every list that leaves this package leaves through
// Extensions, in the table's own order, because Principle VII does not allow a
// map's iteration order to reach anything a caller can observe.
//
// [behaviours §2.15]: ../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item
var admittedExtensions = map[CollectionType][]string{
	Movies: {".mkv", ".mp4", ".avi", ".ts"},
	Shows:  {".mkv", ".avi", ".mp4"},
	Music:  {".flac", ".m4a", ".dsf"},
}

// Extensions returns the extensions this collection type admits, in the order
// 003 §3.2's table gives them, as a fresh slice. An unknown type admits
// nothing and returns nothing.
func (c CollectionType) Extensions() []string {
	src := admittedExtensions[c]
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// Admits reports whether a file carrying this extension is a candidate under
// this collection type.
//
// ext is the extension including its leading dot, as [path.Ext] yields it; a
// value without one never matches, and a file with no extension at all is not a
// candidate anywhere.
//
// The comparison ignores ASCII case, because the reference's does
// `[source: Emby.Naming/Video/VideoResolver.cs:119-123 @ v10.11.11]`, so a
// library holding `.MKV` reads the same on both servers. It is an ASCII fold
// and not [strings.ToLower]: the lists are ASCII, and a Unicode fold would let
// a Kelvin sign spell `.mkv` here while the reference's ordinal comparison
// refused it.
//
// **There is no fallback between the three lists**, and that absence is the
// observable part. A file whose extension is not on its own collection type's
// list is ignored; it is never promoted to another type because some other list
// would take it ([behaviours §2.15]).
//
// [behaviours §2.15]: ../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item
func (c CollectionType) Admits(ext string) bool {
	if len(ext) < 2 || ext[0] != '.' {
		return false
	}
	folded := foldASCIICase(ext)
	for _, admitted := range admittedExtensions[c] {
		if admitted == folded {
			return true
		}
	}
	return false
}

// Skip says which rule refused a path, for the caller that reports what a scan
// did rather than only what it found (003 §3.8's summary).
//
// It is a reason and not a severity: none of these is an error. A library root
// contains artwork, subtitles, `.nfo` sidecars and operating-system detritus,
// and none of that is an error (003 §3.2).
type Skip uint8

// The reasons a single path can be refused. The zero value is the path that was
// not refused, so a caller that ignores the reason still reads correctly.
const (
	// NotSkipped is the path that is a candidate.
	NotSkipped Skip = iota

	// SkipHidden is 003 §3.2's dot rule: any path component beginning with
	// a dot, which covers hidden directories, hidden files and macOS
	// resource forks.
	SkipHidden

	// SkipExtrasFolder is the containing directory naming an extras folder.
	// It refuses every file in that directory, suffixed or not.
	SkipExtrasFolder

	// SkipExtrasFilename is the file whose whole stem is `trailer` or
	// `sample`. It refuses exactly one file.
	SkipExtrasFilename

	// SkipExtrasSuffix is the file whose stem ends in one of the extras
	// suffixes. It refuses exactly one file.
	SkipExtrasSuffix

	// SkipExtension is the extension not being on the library's own
	// collection type's list — never on any other type's.
	SkipExtension
)

// String names the rule, in the terms 003 §3.2 states it in.
func (s Skip) String() string {
	switch s {
	case NotSkipped:
		return "not skipped"
	case SkipHidden:
		return "a path component begins with a dot"
	case SkipExtrasFolder:
		return "the containing directory is an extras folder"
	case SkipExtrasFilename:
		return "the file is named for an extra"
	case SkipExtrasSuffix:
		return "the file carries an extras suffix"
	case SkipExtension:
		return "the extension is not on this collection type's list"
	default:
		return "unknown"
	}
}

// Candidate reports whether relPath is a candidate media file under this
// collection type and, when it is not, which rule refused it.
//
// relPath is root-relative and slash-separated, of the shape [io/fs] yields:
// no leading separator, no `.` and no `..` elements. It is the *path* rules
// only. Three of 003 §3.2's exclusions need more than a path and are decided
// elsewhere: the `.ignore` marker needs the directories above the file, a
// zero-byte file needs its size, and a file being written needs two readings
// (003 plan §6.1).
//
// The rules are applied hidden, extras folder, extras filename, extras suffix,
// extension. The order decides only which reason is reported for a path that
// several rules refuse; the boolean is the same whichever way round they run.
func (c CollectionType) Candidate(relPath string) (bool, Skip) {
	for _, component := range strings.Split(relPath, "/") {
		if IsHiddenName(component) {
			return false, SkipHidden
		}
	}

	name := path.Base(relPath)
	if c.RecognisesExtras() {
		if IsExtrasFolderName(path.Base(path.Dir(relPath))) {
			return false, SkipExtrasFolder
		}
		if IsExtrasFilename(name) {
			return false, SkipExtrasFilename
		}
		if HasExtrasSuffix(name) {
			return false, SkipExtrasSuffix
		}
	}

	if !c.Admits(path.Ext(name)) {
		return false, SkipExtension
	}
	return true, NotSkipped
}

// IsHiddenName reports whether one path component is hidden under 003 §3.2's
// dot rule.
//
// `.` and `..` are path syntax rather than names, so neither is hidden: a
// caller walking a tree asks this of the directory it is standing in, and a
// root that answered "hidden" would exclude the library.
func IsHiddenName(name string) bool {
	if name == "." || name == ".." {
		return false
	}
	return strings.HasPrefix(name, ".")
}

// foldASCIICase lowercases the ASCII letters of s and leaves every other byte
// exactly as it is.
//
// It is deliberately not [strings.ToLower]. Every list this package compares
// against is ASCII, and a Unicode fold maps characters outside ASCII onto
// letters inside it — so `.mKv` spelled with a Kelvin sign would be admitted
// here and refused by the reference's ordinal comparison
// `[source: Emby.Naming/Video/VideoResolver.cs:119-123 @ v10.11.11]`.
func foldASCIICase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		b.WriteByte(ch)
	}
	return b.String()
}
