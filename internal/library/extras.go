package library

import (
	"path"
	"strings"
)

// Extras — trailers, featurettes, deleted scenes, interviews,
// behind-the-scenes — are recognised by the name of the directory holding them
// and by the name of the file itself (003 §3.4). **v1 ignores them rather than
// attaching them**: this feature produces structure and nothing else, an extra
// is not structure, and there is nowhere to attach one, because an item's files
// are the parts of the work itself and filing a trailer among them would make
// it play as part of the film.
//
// The three lists below are the reference's own, read at the pinned tag
// `[source: Emby.Naming/Common/NamingOptions.cs:484-695 @ v10.11.11]`. They are
// a source reading and not a probe: no measurement in this project has sent a
// file named for an extra, so what is asserted about them is what the reference
// is written to do rather than what it was seen to do.
//
// They are constants, and there is nothing to configure them with — the same
// argument 003 plan §4.3 makes for the extension lists.

// extrasFolderNames are the `ExtraRuleType.DirectoryName` tokens of the
// reference's `VideoExtraRules`, in its order
// `[source: Emby.Naming/Common/NamingOptions.cs:484-568 @ v10.11.11]`.
//
// **`Specials` is not among them, and its absence is load-bearing.** It is an
// alias for season zero
// `[source: Emby.Naming/TV/SeasonPathParser.cs:82 @ v10.11.11]`, it sits beside
// `Extras` and `Featurettes` in real libraries, and a scanner that grouped the
// three would drop every special episode in every series while producing a scan
// that looks entirely correct (003 §3.4). The failure is invisible in a
// summary, which is why there is an assertion at this predicate and not only at
// the resolver.
//
// `theme-music` is the one `DirectoryName` token of the reference's list that is
// **not** here. It is declared `MediaType.Audio`, and the reference applies a
// rule only when the file's media type matches it
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:40-44 @ v10.11.11]`. See
// RecognisesExtras for why that gate is expressed here as a collection type.
var extrasFolderNames = []string{
	"trailers",
	"backdrops",
	"behind the scenes",
	"deleted scenes",
	"interviews",
	"scenes",
	"samples",
	"shorts",
	"featurettes",
	"extras",
	"extra",
	"other",
	"clips",
}

// extrasFilenames are the `ExtraRuleType.Filename` tokens of `MediaType.Video`
// `[source: Emby.Naming/Common/NamingOptions.cs:570-580 @ v10.11.11]`. The
// whole stem has to be the token, so this rule refuses exactly one file.
//
// The reference's third `Filename` token, `theme`, is `MediaType.Audio` and is
// left out for the reason RecognisesExtras gives.
var extrasFilenames = []string{
	"trailer",
	"sample",
}

// extrasSuffixes are the `ExtraRuleType.Suffix` tokens of `MediaType.Video`, in
// the reference's order
// `[source: Emby.Naming/Common/NamingOptions.cs:588-694 @ v10.11.11]`.
//
// Four spellings of each of `trailer` and `sample`, and one of everything else;
// `- trailer` and `- sample` carry a space, which is not a typo.
var extrasSuffixes = []string{
	"-trailer",
	".trailer",
	"_trailer",
	"- trailer",
	"-sample",
	".sample",
	"_sample",
	"- sample",
	"-scene",
	"-clip",
	"-interview",
	"-behindthescenes",
	"-deleted",
	"-deletedscene",
	"-featurette",
	"-short",
	"-extra",
	"-other",
}

// RecognisesExtras reports whether the extras rules apply under this collection
// type. They apply under `movies` and `tvshows`, and not under `music`.
//
// That is not a preference; it is the reference's own media-type gate expressed
// in the one term this package has. Every rule in the list above is declared
// `MediaType.Video`, and the reference skips a rule whose media type the file
// does not have
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:40-44 @ v10.11.11]`. Every
// extension `movies` and `tvshows` admit — `.mkv` `.mp4` `.avi` `.ts` — is in
// the reference's `VideoFileExtensions`, and every extension `music` admits —
// `.flac` `.m4a` `.dsf` — is in its `AudioFileExtensions` and in neither the
// other
// `[source: Emby.Naming/Common/NamingOptions.cs:24-80,213-295 @ v10.11.11]`.
// So a file that reaches these rules under `movies` or `tvshows` is always a
// video file there, and a file that reaches them under `music` never is: the
// collection type decides exactly what the media type decides.
//
// The two `MediaType.Audio` tokens the reference carries — `theme` as a whole
// filename and `theme-music` as a directory name — are deliberately not
// implemented. Under `movies` and `tvshows` they would change nothing, because
// a file either matches is refused by §3.2's extension lists first; under
// `music` implementing them would be a behaviour this project owns outright, on
// a source reading with no measurement behind it, that hides files an operator
// expects to see. Both halves show *more* rather than less, which is the safe
// direction for a scanner (003 plan §6.1).
func (c CollectionType) RecognisesExtras() bool {
	return c == Movies || c == Shows
}

// IsExtrasFolderName reports whether one directory component names an extras
// folder.
//
// **It is asked of the directory immediately containing the file, and of no
// other**, because that is the reference's rule: it compares the token against
// `Path.GetFileName(Path.GetDirectoryName(path))`
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:35,51 @ v10.11.11]`. So a
// file nested one level below an extras folder — `Extras/Making Of/clip.mkv` —
// is *not* excluded, there or here. Excluding it would be an item this server
// lacks and the reference has, which is a difference nothing in this project
// has measured and nothing declares.
//
// The comparison ignores case, as the reference's `OrdinalIgnoreCase` does
// `[source: Emby.Naming/Common/NamingOptions.cs:697-699 @ v10.11.11]`.
//
// The reference also refuses to apply this rule when the directory *is* the
// library root
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:52 @ v10.11.11]`, so a
// library whose own root directory is called `Extras` still holds items. Here
// that costs no code: paths are root-relative, so a file directly under the
// root has no containing directory component to ask about.
func IsExtrasFolderName(name string) bool {
	folded := foldASCIICase(name)
	for _, token := range extrasFolderNames {
		if folded == token {
			return true
		}
	}
	return false
}

// IsExtrasFilename reports whether the whole stem of a filename is one of the
// extras filename tokens — `trailer.mkv` and `sample.mkv` and nothing else.
//
// The stem is compared **untrimmed**, which is the reference's own asymmetry:
// its `Filename` rules match `fileNameWithoutExtension` while its `Suffix`
// rules match the same stem with trailing digits removed
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:32-34,48-49 @ v10.11.11]`.
// So `trailer.mkv` is an extra and `trailer2.mkv` is an item.
func IsExtrasFilename(filename string) bool {
	folded := foldASCIICase(stemOf(filename))
	for _, token := range extrasFilenames {
		if folded == token {
			return true
		}
	}
	return false
}

// HasExtrasSuffix reports whether a filename's stem ends in one of the extras
// suffixes, and refuses exactly that one file.
//
// Trailing digits are removed before the comparison, so `-trailer2` is
// recognised as well as `-trailer`; the reference's comment says so in as many
// words `[source: Emby.Naming/Video/ExtraRuleResolver.cs:33-34 @ v10.11.11]`.
// The comparison ignores case, as the reference's `OrdinalIgnoreCase` does
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:49 @ v10.11.11]`.
func HasExtrasSuffix(filename string) bool {
	folded := foldASCIICase(strings.TrimRight(stemOf(filename), "0123456789"))
	for _, token := range extrasSuffixes {
		if strings.HasSuffix(folded, token) {
			return true
		}
	}
	return false
}

// stemOf is a filename without its extension. It takes the base first, so a
// caller may hand it a whole path.
func stemOf(filename string) string {
	name := path.Base(filename)
	return strings.TrimSuffix(name, path.Ext(name))
}
