package library

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Kind is the type of an item, and it is part of every identifier.
//
// The eight are 003 plan §4.2's `items.type` column, spelled exactly as that
// column holds them and as `ports.ScannedItem.Type` carries them, because a
// second spelling of a type is a second identifier for the same item.
type Kind string

// The eight types 003 produces. Nothing here invents one: they are the column's
// own values (003 plan §4.2) and the `Type` of every row in the expected item
// set.
const (
	KindCollectionFolder Kind = "CollectionFolder"
	KindMovie            Kind = "Movie"
	KindSeries           Kind = "Series"
	KindSeason           Kind = "Season"
	KindEpisode          Kind = "Episode"
	KindMusicArtist      Kind = "MusicArtist"
	KindMusicAlbum       Kind = "MusicAlbum"
	KindAudio            Kind = "Audio"
)

// AllKinds returns the eight, in the order 003 plan §4.2's table gives them,
// as a fresh slice on every call — for the same reason
// [AllCollectionTypes] does.
func AllKinds() []Kind {
	return []Kind{
		KindCollectionFolder,
		KindMovie,
		KindSeries,
		KindSeason,
		KindEpisode,
		KindMusicArtist,
		KindMusicAlbum,
		KindAudio,
	}
}

// identifierBytes is how much of the digest an identifier keeps: 16 bytes,
// which is 32 characters of hexadecimal and exactly the shape
// [behaviours §1.4] records for every identifier this API carries.
//
// It is the same constant, for the same reason, as the one `internal/users`
// and `internal/sessions` each hold. They are not shared, because a package
// that imported another's identifier width would make the width a dependency
// rather than a measured fact each derivation states for itself.
//
// [behaviours §1.4]: ../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters
const identifierBytes = 16

// separator is the byte between the three inputs of [DeriveID]. See DeriveID.
const separator = 0x00

// ErrPathAbsolute is the key that is absolute: it begins with a separator, or
// it carries a drive letter.
//
// It is an error and not a normalisation. 003 §3.6: *"A path that is absolute,
// or that climbs above its root, is refused rather than normalised — either one
// means the caller has a path that is not relative to the root it believes it
// is."*
var ErrPathAbsolute = errors.New("the key is absolute and not relative to a library root")

// ErrPathClimbsAboveRoot is the key with a `..` element that leaves the root
// behind. An interior `..` that stays inside the root is not this: `a/../b` is
// `b`, which is a normalisation, and `../b` is this error.
var ErrPathClimbsAboveRoot = errors.New("the key climbs above its library root")

// PathError is what [Normalise] returns, and it carries the key it refused so
// that the operator reading the log line can see which one it was.
//
// # Why this is an error and not a [Skip]
//
// The two are different channels on purpose, and 003 plan §7 is where the
// difference is spent: a [Skip] refuses **one file** and is not a fault — a
// library root contains artwork, subtitles and operating-system detritus — while
// a key that will not normalise **fails the whole library's scan** and changes
// nothing, because *"a caller holding a path it believes is relative and is not
// has computed the wrong root, not the wrong file"* (003 plan §6.3).
//
// The distinction is not decorative and the test for it is not either: the very
// same string that [CollectionType.Candidate] answers `NotSkipped` for —
// `/Movies/The Matrix (1999).mkv` is a candidate as far as every path rule in
// §3.2 is concerned — is refused here. A caller that consulted only the skip
// vocabulary would carry on and write an item under a root it does not have.
type PathError struct {
	// Path is the key exactly as the caller passed it, before any step ran.
	Path string

	// Reason is [ErrPathAbsolute] or [ErrPathClimbsAboveRoot].
	Reason error
}

func (e *PathError) Error() string {
	return fmt.Sprintf("library: %s: %q", e.Reason, e.Path)
}

// Unwrap gives `errors.Is` the sentinel, so a caller can ask which of the two
// it is without matching on a message.
func (e *PathError) Unwrap() error { return e.Reason }

// Normalise reduces a path to the one form an identifier derives from, or
// refuses it.
//
// It is 003 §3.6's three steps, in the order that section states them, and the
// order is load-bearing rather than editorial:
//
//  1. **Separators to one form.** `\` becomes `/`, runs of separators collapse
//     to one, `.` elements disappear and a trailing separator is dropped — so a
//     walker on Windows and a walker on Linux hand the same directory in, and a
//     directory named with and without its trailing slash is one key.
//  2. **The text to Unicode NFC.** One filesystem hands back a decomposed
//     accent where another gives the precomposed character; they are the same
//     name and must be the same identifier.
//  3. **Case folded, and only when the library is not case-sensitive.** A
//     directory renamed only in its capitalisation is not a different
//     directory — unless the operator said, when the library was declared, that
//     it is (§3.6, and the setting is frozen there).
//
// # Why the form is normalised before the case is folded
//
// Because **case folding is not closed over normalisation forms**, and folding
// first gives a different answer for a decomposed capital. The input this
// package's test uses is `I` followed by U+0307 COMBINING DOT ABOVE:
//
//	NFC then fold  →  U+0130  →  "i"            (one rune)
//	fold then NFC  →  "i" U+0307  →  "i" U+0307 (two runes, nothing to compose)
//
// The two spellings of that name would be two items under the wrong order.
// `TestTheUnicodeFormIsNormalisedBeforeTheCaseIsFolded` names the character and
// asserts the two orders actually differ over it, so it cannot be a test that
// looks thorough while asserting an order no input can distinguish.
//
// # Which fold, and why it is not this package's ASCII one
//
// [strings.ToLower] — Unicode's simple lowercase mapping — and deliberately not
// `foldASCIICase`, which every extension comparison in this package uses. The
// ASCII fold is right there because the reference compares extensions
// ordinally `[source: Emby.Naming/Video/VideoResolver.cs:119-123 @ v10.11.11]`
// and a Kelvin sign must not spell `.mkv`. It is wrong here: `Amélie` and
// `AMÉLIE` are the same directory, and an ASCII fold would give them two
// identifiers, which is the exact loss §3.6 spends its case rule preventing.
//
// It is not a *full* case fold either. Full folding maps `ß` to `ss`, so
// `Straße` and `Strasse` — two names, two directories, two films — would become
// one item. Simple lowercasing is the conservative reading of *"renamed only in
// its capitalisation"*.
//
// # What NFC merges, which is worth knowing before a scan surprises somebody
//
// NFC is a canonical equivalence and it has singleton mappings: U+212A KELVIN
// SIGN decomposes to `K`, so a file named with one and a file named with a plain
// `K` are **one key here even in a case-sensitive library**. That is correct —
// they are canonically the same name — but a filesystem can hold both, and the
// two would then derive one identifier. Nothing in this feature can notice that;
// the scan's own duplicate handling is where it would surface.
//
// # What it refuses
//
// An absolute key ([ErrPathAbsolute]) and one that climbs above its root
// ([ErrPathClimbsAboveRoot]), both wrapped in a [PathError] naming the key. The
// absolute test is hand-rolled rather than [path/filepath.IsAbs], because that
// function answers differently depending on which platform the binary was built
// for and a key must normalise identically everywhere: `C:\Movies` is absolute
// on every platform this runs on, not only on the one that agrees.
//
// The returned key is empty when the error is not nil. There is no third
// outcome and no partially normalised key: a caller cannot carry on with what
// it got back.
func Normalise(p string, caseSensitive bool) (string, error) {
	slashed := strings.ReplaceAll(p, `\`, "/")

	if isAbsoluteKey(slashed) {
		return "", &PathError{Path: p, Reason: ErrPathAbsolute}
	}

	cleaned, ok := cleanElements(slashed)
	if !ok {
		return "", &PathError{Path: p, Reason: ErrPathClimbsAboveRoot}
	}

	composed := norm.NFC.String(cleaned)
	if caseSensitive {
		return composed, nil
	}
	return strings.ToLower(composed), nil
}

// isAbsoluteKey answers the same on every platform, which
// [path/filepath.IsAbs] does not.
//
// A leading separator covers the Unix form and, because `\\host\share` has
// already become `//host/share` by the time this is asked, the UNC one. The
// drive-letter form is the other shape a Windows walker produces.
func isAbsoluteKey(slashed string) bool {
	if strings.HasPrefix(slashed, "/") {
		return true
	}
	if len(slashed) >= 2 && slashed[1] == ':' {
		c := slashed[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}

// cleanElements drops empty and `.` elements and applies `..` lexically,
// reporting false when one of them would leave the root.
//
// The reduction is **lexical**, and that is a decision rather than an
// oversight: `a/../b` is `b` here whatever a symbolic link at `a` would have
// made of it. The walk never produces either element — `io/fs` does not — so
// what this handles is a key some caller built by hand, and for such a key the
// string is the identity.
func cleanElements(slashed string) (string, bool) {
	out := make([]string, 0, strings.Count(slashed, "/")+1)
	for _, element := range strings.Split(slashed, "/") {
		switch element {
		case "", ".":
			continue
		case "..":
			if len(out) == 0 {
				return "", false
			}
			out = out[:len(out)-1]
		default:
			out = append(out, element)
		}
	}
	return strings.Join(out, "/"), true
}

// DeriveID is an item's identifier: 32 lowercase hexadecimal characters, the
// first 16 bytes of SHA-256 over the library's identifier, a NUL, the item's
// kind, a NUL, and the **already normalised** key (003 plan §6.3).
//
// key is what [Normalise] returned. This function cannot normalise for itself,
// because whether the case is folded is a property of the library and only the
// caller holds it — so a caller that skipped the step derives an identifier
// nothing else will ever derive again, and every test in this package that
// passes a key passes a normalised one.
//
// Derived rather than allocated, which is Principle VII and the whole of §3.6:
// clients key their caches, favourites and resume positions on these strings,
// and an identifier that moved when a library was rescanned would silently
// discard a user's state.
//
// # The three inputs, and why each one is there
//
// **The library's identifier**, so two libraries configured over the same tree
// are two sets of items rather than one shared set. It is the library's
// *allocated identity* and never its root path — §3.6 keeps that identity
// stable across a rename and a remount precisely so that an operator moving a
// library from `/mnt/a` to `/mnt/b` keeps every identifier under it. Reading
// *"the library root"* in §3.6's table as the root's **path** would reintroduce
// the trap one level down, and it is the same argument
// [001 plan §4](../../specs/001-server-identity-and-discovery/plan.md#4-data-model)
// made against deriving the *server's* identity from its data directory's path.
// The reference has the trap: every one of 448 measured identifiers is
// reproducible from the file's absolute path alone
// `[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`, so
// moving a root there discards every favourite in the library
// ([behaviours §1.4]).
//
// **The kind**, so a directory and the item it backs cannot collide. A `Series`
// at `Shows/The Series` and a `Season` keyed on the same string are two items
// and must be two identifiers.
//
// **The key**, which is §3.6's table: the path relative to the root for a
// `Movie`, `Episode` and `Audio`; the series' identity plus the season number
// for a `Season`; the normalised name for a `Series`, `MusicAlbum` and
// `MusicArtist`; and the library's own identity for a `CollectionFolder`.
//
// # The NULs, which are the whole of what a concatenation gets wrong
//
// A plain concatenation cannot see where its inputs join, so `("ab", k, "c")`
// and `("a", k, "bc")` would be one identifier — one item swallowing another,
// with no error anywhere. This is the same construction, and the same argument,
// `internal/sessions` uses for a session identifier
// ([002 plan §6.5](../../specs/002-authentication-users-and-sessions/plan.md)),
// where the reference's own `MD5(Client + DeviceId)` has exactly that collision.
//
// # Reproducing the reference's bytes is not a goal
//
// [behaviours §1.4] settles it: the reference derives from an MD5 over a .NET
// type name and the file's absolute path, laid out mixed-endian. Reproducing
// its stability is the goal; reproducing its bytes would mean inheriting the
// moved-root trap, and §3.6 says in terms that it is not a goal.
//
// [behaviours §1.4]: ../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters
func DeriveID(libraryID string, kind Kind, key string) string {
	digest := sha256.New()
	digest.Write([]byte(libraryID))
	digest.Write([]byte{separator})
	digest.Write([]byte(kind))
	digest.Write([]byte{separator})
	digest.Write([]byte(key))
	return hex.EncodeToString(digest.Sum(nil)[:identifierBytes])
}
