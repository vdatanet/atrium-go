// Package libraryfixture declares the scanning world of
// [conformance §L2] — paths and filler bytes, no copyrighted media, ever
// (003 spec §6) — and writes it into a directory.
//
// # One declaration, two ways to reach it
//
// Two consumers need this tree and one of them may import nothing of ours:
// `conformance/` speaks HTTP and reaches the tree by running
// `go run ./tools/build_library_fixture -into <dir>` as a subprocess
// (architecture §3, 003 plan §3). So the declaration lives here, the program
// is a second door onto the same value, and there is exactly one place a path
// is written down.
//
// # Generated rather than committed, for three reasons that are each a file
// git cannot hold
//
// A zero-byte file survives a checkout; a *modification time* does not, and
// the whole of 003 §6.4's change signal is one. A file that is being written
// cannot be a committed file at all. And the tree has to be mutated between
// two scans for AC-11 and AC-14, so every test needs its own copy anyway
// (003 plan §8.5).
//
// # The declaration is the reference's reading read backwards
//
// docs/compatibility/reference-fixture-reading.json was taken by mounting
// *this* tree into a container
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`, so
// a fixture that drifts from it makes 003's comparison against that reading
// meaningless while leaving it green — which 003 plan §8.5 calls the worst
// available outcome. Two rules follow, and the second is the one that is easy
// to get wrong:
//
//  1. **Every path the reading names exists here.** The check is in the tests
//     beside this file and it reads the expected paths out of the JSON rather
//     than out of a transcription, because a transcription is how the two stop
//     being the same tree and it would still pass.
//
//  2. **Nothing is added that either server would make an item of.** The tree
//     carries files the reading cannot name — the second part of a multi-part
//     film, which both servers fold into the one item the reading does name,
//     and the exclusions 003 §3.2 exists to exercise. Every one of them is a
//     file *both* readings drop, and the Why field on each says which rule
//     drops it there. A file only Atrium drops would be a difference the
//     reading has no row for; a file neither drops would be an item the
//     reading has no row for. Both are drift, and drift is invisible.
//
// # Four libraries, not six
//
// Movies, Shows and Music are 003's, and Empty is a library with nothing in
// it, which [behaviours §5.7] needs and which 003 must be able to configure
// though it has nothing to scan. The reading also holds Films and Tunes: they
// are the media world 008 encodes with ffmpeg, they live behind a build tag
// (architecture §8), and they are named in NotBuiltHere so that a check
// scoped to four libraries cannot be mistaken for a check over six.
//
// [conformance §L2]: ../../docs/compatibility/conformance.md
// [behaviours §5.7]: ../../docs/compatibility/behaviours.md
package libraryfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
)

// ReferenceReading is the path, relative to the repository root, of the
// reference's recorded reading of this tree. It is named here rather than
// spelled again in every test that checks against it.
const ReferenceReading = "docs/compatibility/reference-fixture-reading.json"

// NotBuiltHere names the libraries of ReferenceReading that this package does
// not build, and why. It exists so that a check over the reading can assert it
// has accounted for every library in the file: a check that silently skipped a
// third of them would read exactly like a check over all six.
var NotBuiltHere = map[string]string{
	"Films": "008's media world, which ffmpeg encodes and a build tag separates (architecture §8)",
	"Tunes": "008's media world, which ffmpeg encodes and a build tag separates (architecture §8)",
}

// File is one file of the tree.
//
// A file's bytes are filler unless the bytes themselves are the point. Two
// files in this tree measure zero, and each of those is a rule rather than an
// omission — see Empty.
type File struct {
	// Path is relative to the library's root, slash-separated.
	Path string

	// Empty declares that this file measures zero bytes *and that its
	// emptiness is the rule it exists to exercise*: the incomplete copy of
	// 003 §3.2, and the `.ignore` marker, of which only an empty one
	// excludes anything. A builder that wrote one byte into either would
	// disable a rule silently and cost nothing visible, which is why the
	// tests assert both as lengths.
	Empty bool

	// Content is the file's exact bytes, for the two files whose bytes are
	// their reason for existing. Empty and Content are mutually exclusive.
	Content []byte

	// Why says what this file is here to exercise, and — for a file the
	// reference's reading does not name — which rule drops it at *both*
	// servers. Every file needs one.
	Why string
}

// Library is one library of the fixture: a name, the collection type it is
// configured with, the directories that must exist even though they hold no
// candidate file, and the files.
type Library struct {
	Name string

	// CollectionType is one of 003 §3.1's three names. The tests assert it
	// against the reading's own collection_type for the same library, so a
	// library configured differently here from the way the reference was
	// configured cannot pass unnoticed.
	CollectionType string

	// Dirs are directories that hold no file of their own. A tree that let
	// them be implied by the files under them could not express the empty
	// season directory 003 §3.4 calls normal, nor a library with nothing in
	// it at all.
	Dirs []string

	Files []File
}

// Libraries is the declaration: the four libraries 003 builds, in a stable
// order.
//
// It returns a fresh copy on every call. The alternative is a package-level
// slice a caller can write into, and a fixture that changed under one test
// because another sorted it would be the least debuggable failure in the
// suite.
func Libraries() []Library {
	return []Library{movies(), shows(), music(), empty()}
}

// LibraryNames is the four library names, in declaration order.
func LibraryNames() []string {
	libraries := Libraries()
	names := make([]string, len(libraries))
	for i, library := range libraries {
		names[i] = library.Name
	}
	return names
}

func movies() Library {
	return Library{
		Name:           "Movies",
		CollectionType: "movies",
		Files: []File{
			{Path: "  Padded   (1999).mkv", Why: "a name carrying leading and trailing whitespace a name derived from a path is trimmed of (003 §3.5)"},
			{Path: "10 Things I Hate About You (1999).mkv", Why: "a title beginning with digits, which the sort key pads (003 §3.7.1)"},
			{Path: "100% Wolf (2020).mkv", Why: "a percent sign in a title"},
			{Path: "2 Fast 2 Furious (2003).mkv", Why: "the pair 10 Things sorts after, by bytes, once both are padded (003 §3.7.1)"},
			{Path: "A Bridge Too Far (1977).mkv", Why: "a leading article the sort key strips (003 §3.7.1)"},
			{Path: "A Broadcast Capture (2011).ts", Why: "the .ts extension, which only the movies list admits (003 §3.2)"},
			{Path: "A Newer Transfer (2015).mp4", Why: "the .mp4 extension"},
			{Path: "Amélie (2001).mkv", Why: "a non-ASCII name, which normalisation has to reduce to one form (003 §3.6)"},
			{
				Path:  "An Incomplete Copy (2000).mkv",
				Empty: true,
				Why:   "the zero-byte film: an incomplete copy, which 003 §3.2 ignores and the reference makes an item of — one of the declared differences (003 plan §6.1)",
			},
			{Path: "An Old Transfer (1985).avi", Why: "the .avi extension"},
			{Path: "Don't Look Up (2021).mkv", Why: "an apostrophe in a title"},
			{Path: "Rock & Roll (1978).mkv", Why: "an ampersand, which the sort key drops and leaves a double space behind (003 §3.7.1)"},
			{Path: "S.W.A.T. (2003).mkv", Why: "dots inside a title, which the sort key turns into a trailing space (003 §3.7.1)"},
			{Path: "The Long Film (1998)/The Long Film (1998) - part1.mkv", Why: "a multi-part film: one item with two parts, not two items (003 §3.3, AC-4)"},
			{
				Path: "The Long Film (1998)/The Long Film (1998) - part2.mkv",
				Why:  "the second part. Named by neither reading: both servers fold it into the one item the reading names The Long Film (1998)",
			},
			{Path: "The Matrix (1999)/The Matrix (1999).mkv", Why: "folder-per-film, where the directory names the work (003 §3.3)"},
			{
				Path:    "The Matrix (1999)/poster.jpg",
				Content: jpegCarryingAnOrientation(),
				Why:     "the image with an EXIF orientation conformance §L2 plants beside a film for 006's resize edge. Named by neither reading: no collection type admits .jpg (003 §3.2)",
			},
			{Path: "Wall-E (2008).mkv", Why: "a hyphen inside a title"},
			{Path: "iRobot (2004).mkv", Why: "a lowercase initial, which case folding has to reach (003 §3.6)"},
			{
				Path:    "An Old Transfer (1985).srt",
				Content: subtitleInALegacyEncoding(),
				Why:     "the legacy single-byte subtitle conformance §L2 keeps for behaviours §5.11. Named by neither reading: no collection type admits .srt (003 §3.2)",
			},
			{
				Path: "Not A Film (1999).mp3",
				Why:  "theme music beside a film. Named by neither reading: measured to become no item of any type under a video root `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`, behaviours §2.15",
			},
			{
				Path: ".hidden/A Hidden Film (1990).mkv",
				Why:  "a candidate inside a hidden directory. Named by neither reading: 003 §3.2's dot rule here, `**/.*` there `[source: Emby.Server.Implementations/Library/IgnorePatterns.cs:89 @ v10.11.11]`",
			},
			{
				Path: "._Wall-E (2008).mkv",
				Why:  "a macOS resource fork, which is a hidden *file* carrying an admitted extension. Excluded by the same two rules, and it is the one that separates skipping a file from not descending into a tree",
			},
			{
				Path:  "Excluded/.ignore",
				Empty: true,
				Why:   "the marker, and only an empty one excludes anything (003 §3.2's 2026-09-05 amendment, U-42)",
			},
			{
				Path: "Excluded/An Excluded Film (1994).mkv",
				Why:  "what the marker excludes. Named by neither reading: excluded here by 003 §3.2, and there by `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:41-68 @ v10.11.11]`",
			},
		},
	}
}

func shows() Library {
	return Library{
		Name:           "Shows",
		CollectionType: "tvshows",
		Dirs: []string{
			// A season directory with no episodes in it. 003 §3.4 calls that
			// normal, and the reference makes a Season of it, so it has to be
			// a directory that exists rather than one the files imply.
			"The Series/Season 03",
		},
		Files: []File{
			{Path: "24/Season 01/24 - S01E01 - 12-00 AM.mkv", Why: "a series whose title is digits: the numbering is matched against the filename first, so 24 keeps its title (003 §3.4, AC-7)"},
			{Path: "The Daily Show/Season 2024/The Daily Show - 2024-01-31.mkv", Why: "date-based naming for a daily show (003 §3.4)"},
			{Path: "The Series/Specials/The Series - S00E01 - A Special.mkv", Why: "Specials is season zero and is not an extras directory (003 §3.4, AC-6)"},
			{Path: "The Series/Season 01/The Series - S01E01 - Pilot.mkv", Why: "the ordinary episode"},
			{Path: "The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv", Why: "a multi-episode file: one item spanning two numbers (003 §3.4, AC-5)"},
			{Path: "The Series/Season 01/The Series - S01E04 - Old Transfer.avi", Why: "the .avi extension under a tvshows root"},
			{Path: "The Series/Season 01/blob.mkv", Why: "a name that says too little to place: an unplaceable item rather than a skipped file, which 003 §3.8 counts apart"},
			{Path: "The Series/The Series - S02E01 - No Season Directory.mkv", Why: "a season inferred from the episode where no directory exists (003 §3.4)"},
			{Path: "The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4", Why: "an episode number beyond any real count, which is not an error (003 §3.4)"},
			{
				Path: "The Series/Season 01/Not An Episode.mka",
				Why:  "the .mka case of behaviours §2.15. Named by neither reading: measured to become no item of any type under a video root `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`",
			},
		},
	}
}

func music() Library {
	return Library{
		Name:           "Music",
		CollectionType: "music",
		Files: []File{
			{Path: "The Artist/First Album (2001)/01 - Opening.flac", Why: "an album with a year in its directory (003 §3.5)"},
			{Path: "The Artist/First Album (2001)/02 - Second.flac", Why: "a second track, so the album is not one row"},
			{Path: "The Artist/Double Album/CD1/01 - First Disc.flac", Why: "an album split across discs: one album, tracks carrying disc numbers (003 §3.5, AC-8)"},
			{Path: "The Artist/Double Album/CD2/01 - Second Disc.flac", Why: "the second disc of the same album"},
			{Path: "The Artist/Second Album/01 - In Another Container.m4a", Why: "the .m4a extension (003 §3.2)"},
			{Path: "The Artist/Second Album/02 - And Another.dsf", Why: "the .dsf extension (003 §3.2)"},
			{Path: "The Artist/spandau_ballet-through_the_barricades/01 - Tagged Differently.flac", Why: "a directory whose name a tag will contradict, which is 004's half of 003 §3.5's precedence rule"},
			{Path: "Various Artists/A Compilation (1999)/01 - By One Artist.flac", Why: "a compilation with a different artist per track: one album, not one per track (003 §3.5, AC-9)"},
			{Path: "Various Artists/A Compilation (1999)/02 - By Another.flac", Why: "the second artist of the compilation"},
			{Path: "Various Artists/A Compilation (1999)/03 - By A Third.flac", Why: "the third"},
		},
	}
}

// empty is a library with no file and no directory of its own — and it is
// still a directory that exists.
//
// That is buildLibrary's first line rather than a row in Dirs, and the
// difference is the whole point: a builder that created only the directories
// its files imply would leave this library as a path that is *missing*, which
// is AC-12's unreadable root and not an empty library. The two states have to
// be told apart, because behaviours §5.7 needs a library that reads clean and
// holds no item, and AC-12 needs a scan that fails and removes nothing.
// TestEmptyIsADirectoryThatExistsAndHoldsNothing asserts both halves: that the
// declaration holds nothing, and that the built tree holds the directory
// anyway.
func empty() Library {
	return Library{
		Name:           "Empty",
		CollectionType: "movies",
	}
}

// Build writes the four libraries into root, creating root if it is not there.
//
// It refuses a root that already holds one of them. The tree is generated per
// test and per run precisely so that nothing carries over between two of them,
// and a build that wrote over a tree somebody had mutated would hand the next
// scan a mixture of the two.
func Build(root string) error {
	libraries := Libraries()

	for _, library := range libraries {
		if _, err := os.Stat(filepath.Join(root, library.Name)); err == nil {
			return fmt.Errorf("libraryfixture: %s already exists in %s: the tree is built into a fresh directory, never over an existing one", library.Name, root)
		}
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	for _, library := range libraries {
		if err := buildLibrary(filepath.Join(root, library.Name), library); err != nil {
			return err
		}
	}
	return nil
}

func buildLibrary(root string, library Library) error {
	// Every library gets a directory, whether or not it holds a file. The
	// Empty library is nothing else, and see empty() for why that is a rule
	// and not a convenience.
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	for _, dir := range library.Dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			return err
		}
	}

	for _, file := range library.Files {
		if file.Why == "" {
			return fmt.Errorf("libraryfixture: %s/%s has no Why: a file nobody can say the reason for is a file nobody can decide to remove", library.Name, file.Path)
		}
		if file.Empty && len(file.Content) > 0 {
			return fmt.Errorf("libraryfixture: %s/%s is declared empty and carries content", library.Name, file.Path)
		}

		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Bytes is what the builder writes for this file.
//
// Filler is derived from the path, so it is stable across builds and differs
// between files: a tree whose every file held the same bytes would make a
// content hash a constant, and 003 §6.4's signal is the pair of size and
// modification time, which needs the sizes to be a property of the
// declaration rather than of the run.
func (f File) Bytes() []byte {
	switch {
	case f.Empty:
		return nil
	case f.Content != nil:
		return f.Content
	default:
		return filler(f.Path)
	}
}

// filler expands the path's digest to the size the path decides.
func filler(path string) []byte {
	digest := sha256.Sum256([]byte(path))
	size := 512 + int(digest[0])*4

	out := make([]byte, 0, size)
	for len(out) < size {
		out = append(out, digest[:]...)
	}
	return out[:size]
}

// subtitleInALegacyEncoding is the one input behaviours §5.11 has: a subtitle
// file in a legacy single-byte encoding, which no detector is asked about and
// which is therefore not valid UTF-8.
func subtitleInALegacyEncoding() []byte {
	// Windows-1252: 0xE9 is é, 0xE0 is à. In UTF-8 either would be two bytes,
	// so a reader that assumed UTF-8 fails on this file rather than guessing.
	text := "1\r\n" +
		"00:00:01,000 --> 00:00:04,000\r\n" +
		"O\xf9 est le caf\xe9 ?\r\n" +
		"\r\n" +
		"2\r\n" +
		"00:00:05,000 --> 00:00:08,000\r\n" +
		"\xc0 c\xf4t\xe9 du cin\xe9ma.\r\n"
	return []byte(text)
}

// jpegCarryingAnOrientation is the other input no remote request reaches: an
// image with an EXIF orientation planted beside a film, which 006 owes a
// resize edge for.
//
// The APP1 segment is spliced in after the start-of-image marker, because
// image/jpeg writes none and the orientation is the whole point of the file.
func jpegCarryingAnOrientation() []byte {
	var encoded bytes.Buffer
	frame := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for x := 0; x < 4; x++ {
		for y := 0; y < 2; y++ {
			frame.Set(x, y, color.RGBA{R: uint8(40 * x), G: uint8(80 * y), B: 128, A: 255})
		}
	}
	if err := jpeg.Encode(&encoded, frame, &jpeg.Options{Quality: 80}); err != nil {
		// image/jpeg cannot fail on an in-memory image of a supported type,
		// and a fixture that silently shipped a truncated file would be worse
		// than a build that stopped.
		panic("libraryfixture: encoding the fixture's JPEG: " + err.Error())
	}

	body := encoded.Bytes()
	out := make([]byte, 0, len(body)+len(exifOrientationSegment))
	out = append(out, body[:2]...) // SOI
	out = append(out, exifOrientationSegment...)
	out = append(out, body[2:]...)
	return out
}

// exifOrientationSegment is an APP1 segment holding a TIFF header and one IFD
// entry: tag 0x0112 (Orientation), type SHORT, value 6 — rotate 90° clockwise,
// which is the value a phone writes and a resizer has to honour.
var exifOrientationSegment = func() []byte {
	tiff := make([]byte, 0, 26)
	u16 := func(v uint16) { tiff = binary.LittleEndian.AppendUint16(tiff, v) }
	u32 := func(v uint32) { tiff = binary.LittleEndian.AppendUint32(tiff, v) }

	tiff = append(tiff, 'I', 'I') // little-endian
	u16(42)                       // the TIFF magic
	u32(8)                        // offset of the first IFD
	u16(1)                        // one entry
	u16(0x0112)                   // Orientation
	u16(3)                        // SHORT
	u32(1)                        // one value
	u16(6)                        // rotate 90° clockwise
	u16(0)                        // the other half of the four-byte value field
	u32(0)                        // no next IFD

	payload := append([]byte("Exif\x00\x00"), tiff...)

	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	return append(segment, payload...)
}()

// ExifHeader is the marker a test looks for to know the orientation survived
// into the written file.
const ExifHeader = "Exif\x00\x00"

// Paths is every path this library declares, directories included, in the
// order the declaration gives them. It is here so that a caller can say what
// it wrote without walking a tree it has just built.
func (l Library) Paths() []string {
	paths := make([]string, 0, len(l.Dirs)+len(l.Files))
	paths = append(paths, l.Dirs...)
	for _, file := range l.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

// String is the one-line summary the builder's program prints per library.
func (l Library) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s): %d files", l.Name, l.CollectionType, len(l.Files))
	switch len(l.Dirs) {
	case 0:
	case 1:
		fmt.Fprintf(&b, ", 1 directory holding none")
	default:
		fmt.Fprintf(&b, ", %d directories holding none", len(l.Dirs))
	}
	return b.String()
}
