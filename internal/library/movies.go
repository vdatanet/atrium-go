package library

import (
	"sort"
	"strconv"
	"strings"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// partKind separates the two stacking rules the reference has, and keeping
// them apart is not decoration: a stack established by `cd1` does not take
// `cda` and one established by `cda` does not take `cd2`
// `[source: Emby.Naming/Video/StackResolver.cs:109-112 @ v10.11.11]`.
type partKind uint8

const (
	// partAbsent is the file that carries no part marker at all.
	partAbsent partKind = iota

	// partNumeric is `cd1`, `part2`, `disc10`.
	partNumeric

	// partAlphabetic is `cda`, `cdb` — a single letter `a`–`d` after the same
	// word, and **not** a bare trailing letter. See [splitPartMarker].
	partAlphabetic
)

// partTypes is the marker vocabulary, and it is the reference's rather than
// 003 §3.3's original parenthetical
// `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]`.
//
// `dis[ck]` in the source is two words here. The list is read in order and
// every match is delimited on both sides, so the order between `part` and `pt`
// changes nothing.
var partTypes = []string{"cd", "dvd", "part", "pt", "disc", "disk"}

// partSeparators are the characters that may stand between a title and a part
// marker. A closing bracket may stand there instead, with no separator at all.
const partSeparators = " _.-"

// partMarker is what [splitPartMarker] found.
type partMarker struct {
	// Type is the marker word, folded: one of [partTypes].
	Type string

	// Kind is numeric or alphabetic.
	Kind partKind

	// Token is the number exactly as it was written, folded. It is what
	// decides whether a second file is a **duplicate** part rather than a new
	// one, and `1` and `01` are two different parts by that test — which is
	// the reference's own, because its part dictionary is keyed on the string
	// `[source: Emby.Naming/Video/StackResolver.cs:135-138 @ v10.11.11]`.
	Token string

	// Ordinal is the part's position: the number for a numeric marker, and
	// 1–4 for `a`–`d`.
	Ordinal int
}

// splitPartMarker separates a filename's stem into the title it stacks under
// and the part marker that follows it, reporting whether there was one.
//
// # The vocabulary, and the divergence that is the point of it
//
// A marker is one of [partTypes], separated from the title by a space, an
// underscore, a full stop or a hyphen — or by a closing bracket, with no
// separator — optionally wrapped in a bracket of its own, and followed either
// by digits or by a **single letter `a`–`d` after that same word**
// `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]`.
//
// 003 §3.3 originally wrote the vocabulary as `part1`/`pt1`/`cd1`/`disc1`
// *"and the `-a`/`-b` form"*, and the second half of that is not a form the
// reference has: a **bare** trailing letter stacks nothing there. So
// `The Film - a.mkv` and `The Film - b.mkv` are two works and
// `The Film - cda.mkv` and `The Film - cdb.mkv` are one, and the specification
// was amended on 2026-09-05 rather than implemented as written.
//
// The correction matters more than the two words it changes, and it runs the
// opposite way to the warning §3.3 spends its paragraph on: getting stacking
// wrong usually **doubles** a library, and reading a bare letter as a marker
// merges two films into one and makes the second **disappear**. It is the one
// reading in this feature that loses an item, which is why
// [U-43](../../docs/compatibility/reference-target.md) names it as the row to
// measure first — and neither shape has been sent to a running reference, so
// this is a source reading and the test that asserts it is written to go red
// the day one is.
//
// # Where the title ends
//
// At the **earliest** place a marker can start, because the reference's title
// capture is non-greedy. The title may be empty, but the separator may not: a
// file called ` cd1.mkv` is a part of a nameless stack and one called
// `cd1.mkv` is not a part of anything, there and here.
func splitPartMarker(stem string) (string, partMarker, bool) {
	for token := 1; token < len(stem); token++ {
		typ, after, ok := partTypeAt(stem, token)
		if !ok {
			continue
		}
		marker, ok := partNumberAt(stem, after)
		if !ok {
			continue
		}
		marker.Type = typ
		base, ok := titleBeforePart(stem, token)
		if !ok {
			continue
		}
		return base, marker, true
	}
	return stem, partMarker{}, false
}

// partTypeAt matches one of [partTypes] at i, returning it folded and the index
// just past it.
func partTypeAt(stem string, i int) (string, int, bool) {
	rest := foldASCIICase(stem[i:])
	for _, typ := range partTypes {
		if strings.HasPrefix(rest, typ) {
			return typ, i + len(typ), true
		}
	}
	return "", 0, false
}

// partNumberAt reads the number that follows a marker word: digits, or one
// letter `a`–`d`, with an optional run of separators before it and an optional
// closing bracket after it, and then the end of the stem.
func partNumberAt(stem string, i int) (partMarker, bool) {
	for i < len(stem) && strings.ContainsRune(partSeparators, rune(stem[i])) {
		i++
	}
	if i >= len(stem) {
		return partMarker{}, false
	}

	var marker partMarker
	start := i
	switch {
	case isASCIIDigit(stem[i]):
		value := 0
		for i < len(stem) && isASCIIDigit(stem[i]) {
			value = value*10 + int(stem[i]-'0')
			i++
		}
		marker = partMarker{Kind: partNumeric, Token: stem[start:i], Ordinal: value}
	default:
		letter := foldASCIICase(stem[i : i+1])
		if letter < "a" || letter > "d" {
			return partMarker{}, false
		}
		i++
		marker = partMarker{
			Kind:    partAlphabetic,
			Token:   letter,
			Ordinal: int(letter[0]-'a') + 1,
		}
	}

	if i < len(stem) && (stem[i] == ')' || stem[i] == ']') {
		i++
	}
	if i != len(stem) {
		return partMarker{}, false
	}
	return marker, true
}

// titleBeforePart is what precedes a marker word at token, or false when
// nothing separates the two.
//
// An opening bracket immediately before the word belongs to the marker. Before
// that, either a closing bracket ends the title with no separator at all, or a
// run of [partSeparators] does.
func titleBeforePart(stem string, token int) (string, bool) {
	open := token
	if open > 0 && (stem[open-1] == '(' || stem[open-1] == '[') {
		open--
	}
	if open == 0 {
		return "", false
	}
	if c := stem[open-1]; c == ']' || c == ')' || c == '}' {
		return stem[:open], true
	}
	run := open
	for run > 0 && strings.ContainsRune(partSeparators, rune(stem[run-1])) {
		run--
	}
	if run == open {
		return "", false
	}
	return stem[:run], true
}

// filmCandidate is one film, after the group pass folded its parts together.
type filmCandidate struct {
	root int
	dir  string

	// nameSource is the stem the name derives from: the marker-stripped title
	// for a stack, and the whole stem for a file that stacked with nothing.
	//
	// The difference is what the multi-part assertion is really about. A build
	// that stacks the parts and then takes the **first file's** stem names the
	// film `The Long Film (1998) - part1`, and every count in the suite is
	// still right.
	nameSource string

	files []ports.ScannedFile
}

// resolveMovies is 003 plan §6.2's three passes for a `movies` library.
//
// Classify reads what one path yields, group folds multi-part films together
// and decides which directories name their film, and place derives the
// identifiers and the sort keys. The middle pass is why [Resolve] takes every
// reading at once: both of its decisions need a file's siblings, and neither
// can be taken and then revised without making the answer depend on the order
// the entries arrived in.
func resolveMovies(lib ports.Library, readings []Reading, parentID string) ([]ports.ScannedItem, error) {
	candidates := stackParts(lib, readings)
	nameByDirectory := directoriesThatNameTheirFilm(candidates)

	items := make([]ports.ScannedItem, 0, len(candidates))
	for _, candidate := range candidates {
		source := candidate.nameSource
		if directory, ok := nameByDirectory[directoryKey(candidate)]; ok {
			source = directory
		}
		name, year := cleanVideoName(source)

		path := candidate.files[0].Path
		key, err := Normalise(path, lib.CaseSensitive)
		if err != nil {
			return nil, err
		}

		item := ports.ScannedItem{
			ID:             DeriveID(lib.ID, KindMovie, key),
			LibraryID:      lib.ID,
			ParentID:       parentID,
			Type:           string(KindMovie),
			Name:           name,
			Path:           path,
			RootOrdinal:    candidate.root,
			ProductionYear: year,
			Files:          candidate.files,
		}
		item.SortKey = SortKeyFor(&item)
		items = append(items, item)
	}
	return items, nil
}

// stack is one group of files that share a title and a marker word.
type stack struct {
	root       int
	dir        string
	nameSource string
	typ        string
	kind       partKind
	tokens     map[string]bool
	parts      []stackPart
}

type stackPart struct {
	ordinal int
	entry   Entry
}

// stackParts is the group pass: it folds a multi-part film's files into one
// candidate and leaves everything else alone.
//
// The rules are the reference's, and each of the three exists because of a
// failure the others do not cover
// `[source: Emby.Naming/Video/StackResolver.cs:76-127 @ v10.11.11]`:
//
//   - Files stack only within **one directory**, so two unrelated films called
//     `Feature - cd1.mkv` in two folders are two films.
//   - The **first** file establishes the marker word and whether the stack is
//     numeric or alphabetic, and a later file that disagrees with either joins
//     nothing and stands on its own.
//   - A part number already in the stack is a **duplicate** and joins nothing,
//     because a stack with two second parts has lost a file rather than gained
//     one.
//
// And the rule that is easiest to leave out: **a stack of one is not a
// stack**. A lone `The Film - cd1.mkv` is a film with one file, named after its
// whole stem — which is how it keeps the name the reference gives it, since
// `cd1` is in the release-tag vocabulary and `part1` is not.
//
// No map is iterated here. The stacks are appended to a slice in the sorted
// entry order and read back out of it, which is what makes two readings of one
// tree in opposite orders produce the same part order.
func stackParts(lib ports.Library, readings []Reading) []filmCandidate {
	stacks := make(map[string]*stack)
	var order []*stack
	var loose []filmCandidate

	for _, reading := range readings {
		for _, entry := range reading.Entries {
			stem := stemOfPath(entry.Path)
			dir := dirName(entry.Path)

			base, marker, ok := splitPartMarker(stem)
			if !ok {
				loose = append(loose, singleFileCandidate(reading.Root, dir, stem, entry))
				continue
			}

			key := stackKey(lib, reading.Root, dir, base)
			existing, seen := stacks[key]
			if !seen {
				fresh := &stack{
					root:       reading.Root,
					dir:        dir,
					nameSource: base,
					typ:        marker.Type,
					kind:       marker.Kind,
					tokens:     map[string]bool{marker.Token: true},
					parts:      []stackPart{{ordinal: marker.Ordinal, entry: entry}},
				}
				stacks[key] = fresh
				order = append(order, fresh)
				continue
			}
			if existing.typ != marker.Type || existing.kind != marker.Kind || existing.tokens[marker.Token] {
				loose = append(loose, singleFileCandidate(reading.Root, dir, stem, entry))
				continue
			}
			existing.tokens[marker.Token] = true
			existing.parts = append(existing.parts, stackPart{ordinal: marker.Ordinal, entry: entry})
		}
	}

	candidates := loose
	for _, s := range order {
		if len(s.parts) < 2 {
			only := s.parts[0].entry
			candidates = append(candidates, singleFileCandidate(
				s.root, s.dir, stemOfPath(only.Path), only,
			))
			continue
		}
		sort.SliceStable(s.parts, func(a, b int) bool {
			if s.parts[a].ordinal != s.parts[b].ordinal {
				return s.parts[a].ordinal < s.parts[b].ordinal
			}
			return s.parts[a].entry.Path < s.parts[b].entry.Path
		})
		files := make([]ports.ScannedFile, len(s.parts))
		for i, part := range s.parts {
			files[i] = ports.ScannedFile{
				Ordinal:    i + 1,
				Path:       part.entry.Path,
				Size:       part.entry.Size,
				ModifiedAt: part.entry.ModifiedAt,
			}
		}
		candidates = append(candidates, filmCandidate{
			root:       s.root,
			dir:        s.dir,
			nameSource: s.nameSource,
			files:      files,
		})
	}

	// The candidates are deliberately **not** sorted here. There is one sort in
	// this feature and it is `sortItems`, on the way out of [Resolve]: two
	// sorts means one of them is dead, and a dead sort is a determinism
	// assertion that holds for a reason nobody can find.
	return candidates
}

// singleFileCandidate is a film with one file. Its ordinal is 0, which is what
// `ports.ScannedFile.Ordinal` reserves for a file that is not a part of
// anything.
func singleFileCandidate(root int, dir, stem string, entry Entry) filmCandidate {
	return filmCandidate{
		root:       root,
		dir:        dir,
		nameSource: stem,
		files: []ports.ScannedFile{{
			Ordinal:    0,
			Path:       entry.Path,
			Size:       entry.Size,
			ModifiedAt: entry.ModifiedAt,
		}},
	}
}

// stackKey is what decides that two files are parts of one film: the same
// root, the same directory and the same title, compared the way the library
// compares everything else.
//
// The title goes through [Normalise] with the library's own case rule rather
// than through a fold chosen here, so `The Film - cd1.mkv` and
// `THE FILM - cd2.mkv` are one film in a case-insensitive library and two in a
// case-sensitive one. A title is a name and not an extension, which is why this
// is not `foldASCIICase`.
//
// A title that will not normalise is used as it stands. Nothing is lost: the
// key is compared against other keys built the same way and never leaves this
// function, and a title cannot be an absolute path — the callers hand it the
// stem of one filename.
func stackKey(lib ports.Library, root int, dir, base string) string {
	normalised, err := Normalise(base, lib.CaseSensitive)
	if err != nil {
		normalised = base
	}
	return joinKey(strconv.Itoa(root), dir, normalised)
}

// directoryKey identifies the directory a candidate sits in.
func directoryKey(candidate filmCandidate) string {
	return joinKey(strconv.Itoa(candidate.root), candidate.dir)
}

// directoriesThatNameTheirFilm is 003 §3.3's measured rule, and the pass that
// makes the half of it a single path cannot decide decidable.
//
// **Where a film sits in its own directory, the directory names it.** Measured
// across 1,557 films, the directory's cleaned name matched what the reference
// resolved 1,087 times against the file's 457
// `[read: Jellyfin 10.11.11, 2026-08-27]`, and the reason is mechanical: the
// tools that fetch films mangle filenames and leave directories alone.
//
// **A directory holding several different titles is a category and names
// none of them**, which is the clause that needs the siblings. A multi-part
// film is one candidate by the time this runs, so its directory holds one film
// and names it — and that is why `The Long Film (1998)/… - part1.mkv` is
// `The Long Film` rather than `The Long Film (1998) - part1`.
//
// A library **root** never names a film, however few films it holds. A root is
// the library, and 003 plan §4.2 gives the library's own row no path precisely
// because a library may have several roots. `dirName` answers "." for a file
// directly under a root, and that is the value excluded here.
//
// The returned map is a lookup and is never iterated.
func directoriesThatNameTheirFilm(candidates []filmCandidate) map[string]string {
	counts := make(map[string]int)
	for _, candidate := range candidates {
		counts[directoryKey(candidate)]++
	}

	names := make(map[string]string)
	for _, candidate := range candidates {
		if candidate.dir == "." {
			continue
		}
		key := directoryKey(candidate)
		if counts[key] != 1 {
			continue
		}
		names[key] = baseName(candidate.dir)
	}
	return names
}

// joinKey builds a composite map key out of parts that can each contain any
// character a filename can, separated by a NUL — the same reason [DeriveID]
// separates its three inputs, one layer down and with nothing observable
// riding on it.
func joinKey(parts ...string) string {
	return strings.Join(parts, string(rune(separator)))
}
