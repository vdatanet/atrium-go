package library

import (
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// These are `library`-level assertions, the weakest of 003 plan §8.1's three
// levels and the only one this task can reach: [Resolve] is a pure function
// with no route, no store and no wire representation between it and anything a
// client sees.
//
// **What none of it proves is 003 plan §8.3 row 3**: that a client asking for a
// season's children gets them. Everything below is about a `ParentID` field in
// a value. That the field reaches a `parent_id` column is T12's, that
// `/Items?parentId=` answers with it is 005's, and a green run here is evidence
// for neither. The deferral is left standing rather than dressed up: there is
// no test in this file that could be mistaken for it.

// aShowsLibrary is the library every test here resolves against. Its identity
// is a literal because it is an **input** to every identifier below.
func aShowsLibrary() ports.Library {
	return ports.Library{
		ID:             "fedcba9876543210fedcba9876543210",
		Name:           "Shows",
		NameFolded:     "shows",
		CollectionType: string(Shows),
	}
}

// resolveShowFiles resolves one root's reading of a `tvshows` library.
func resolveShowFiles(t *testing.T, paths ...string) Plan {
	t.Helper()
	plan, err := Resolve(aShowsLibrary(), []Reading{aReading(0, paths...)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return plan
}

// itemsOfKind is every item of one type, in the plan's own order.
func itemsOfKind(plan Plan, kind Kind) []ports.ScannedItem {
	var out []ports.ScannedItem
	for _, item := range plan.Items {
		if Kind(item.Type) == kind {
			out = append(out, item)
		}
	}
	return out
}

// number renders an optional number for a failure message, so that a nil and a
// zero cannot read the same. Season **0** is `Specials` and a season with no
// number at all is not it (003 §3.4), which is the one pair in this feature
// where "0" and "absent" are different answers about the same field.
func number(n *int) string {
	if n == nil {
		return "absent"
	}
	return strconv.Itoa(*n)
}

// ---------------------------------------------------------------------------
// AC-5 — a multi-episode file
// ---------------------------------------------------------------------------

// TestAMultiEpisodeFileIsOneItemSpanningTwoNumbers is AC-5.
//
// **The count is asserted first and it is the whole point.** Two items each
// carrying one of the two numbers satisfies every per-field assertion anybody
// would write about `IndexNumber` and `IndexNumberEnd`, and it is the failure
// spec §3.4 spells out — "**One** episode item spanning two numbers, not two
// items".
func TestAMultiEpisodeFileIsOneItemSpanningTwoNumbers(t *testing.T) {
	plan := resolveShowFiles(t, "The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes from one file, want 1: %q", len(episodes), namesOf(episodes))
	}
	episode := episodes[0]

	if episode.IndexNumber == nil || *episode.IndexNumber != 2 {
		t.Errorf("IndexNumber = %s, want 2", number(episode.IndexNumber))
	}
	if episode.IndexNumberEnd == nil || *episode.IndexNumberEnd != 3 {
		t.Errorf("IndexNumberEnd = %s, want 3", number(episode.IndexNumberEnd))
	}
	if episode.ParentIndexNumber == nil || *episode.ParentIndexNumber != 1 {
		t.Errorf("ParentIndexNumber = %s, want 1", number(episode.ParentIndexNumber))
	}
	if got, want := delimiter+episode.Name+delimiter, delimiter+"Two Parter"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
	if episode.Unplaceable {
		t.Error("the item is marked unplaceable")
	}
}

// TestASpacedHyphenBeforeDigitsIsNotAnEpisodeRange is the assertion the fixture
// tree needs and that AC-5's own test cannot make, because both halves of
// [readEpisodeRange]'s rule have to be wrong in opposite directions to be seen.
//
// `24 - S01E01 - 12-00 AM` ends in a spaced hyphen and then digits. A rule that
// took any hyphen makes that one file **episodes 1 to 12** — and the count is
// still one item, the season is still 1, and the episode number is still 1, so
// AC-5's test, AC-7's test and the fixture's parent-child comparison all stay
// green.
func TestASpacedHyphenBeforeDigitsIsNotAnEpisodeRange(t *testing.T) {
	for _, row := range []struct {
		stem string
		want *int
	}{
		{"24 - S01E01 - 12-00 AM", nil},
		{"The Series - S01E02-E03 - Two Parter", intPointer(3)},
		{"The Series - S01E02-03", intPointer(3)},
		{"The Series - S01E02 - E03", intPointer(3)},
		// The reference's own worked example: a resolution behind the hyphen
		// is not a second episode number
		// `[source: Emby.Naming/TV/EpisodePathParser.cs:154-160 @ v10.11.11]`.
		{"series - s09e14-1080p", nil},
		{"series - s09e14-720i", nil},
	} {
		got := parseEpisodeNumbering(row.stem)
		if number(got.episodeEnd) != number(row.want) {
			t.Errorf("%q: IndexNumberEnd = %s, want %s", row.stem, number(got.episodeEnd), number(row.want))
		}
		if got.episode == nil || *got.episode != episodeOf(row.stem) {
			t.Errorf("%q: IndexNumber = %s, want %d", row.stem, number(got.episode), episodeOf(row.stem))
		}
	}
}

// episodeOf is the episode number each row of the table above carries, written
// out so that a row cannot pass by having no episode number either.
func episodeOf(stem string) int {
	switch stem {
	case "24 - S01E01 - 12-00 AM":
		return 1
	case "series - s09e14-1080p", "series - s09e14-720i":
		return 14
	default:
		return 2
	}
}

func intPointer(n int) *int { return &n }

// ---------------------------------------------------------------------------
// AC-6 — `Specials`, and the companion
// ---------------------------------------------------------------------------

// TestSpecialsIsSeasonZeroAndExtrasBesideItIsNoSeasonAtAll is AC-6 and the
// companion spec §3.4 asks for in the same breath.
//
// The specification's warning is that grouping the three "would drop every
// special episode in every series while producing a scan that looks entirely
// correct", so both halves are asserted over **one** tree that holds all three
// directories with a plausibly-numbered file in each:
//
//   - `Specials` is season 0 and holds its episode. Adding `specials` to the
//     extras folder names takes the season and the episode away.
//   - `Extras` and `Featurettes` are seasons of nothing. Removing them from
//     the extras folder names gives season 0 **three** episodes, and — because
//     the reference's own season parser reads `Extras` as season zero too
//     `[source: Emby.Naming/TV/SeasonPathParser.cs:81-86 @ v10.11.11]` — could
//     hand season zero the wrong directory for its path.
//
// T2 asserts `Specials` at the predicate that decides candidacy
// (`TestSpecialsIsNotAnExtrasName`). This is the resolver's half and not a
// second copy of that: what is asserted here is the season, the number and the
// path, none of which a predicate has.
func TestSpecialsIsSeasonZeroAndExtrasBesideItIsNoSeasonAtAll(t *testing.T) {
	plan := resolveShowFiles(t,
		"The Series/Extras/The Series - S00E02 - Deleted Scene.mkv",
		"The Series/Featurettes/The Series - S00E03 - Making Of.mkv",
		"The Series/Specials/The Series - S00E01 - A Special.mkv",
	)

	seasons := itemsOfKind(plan, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want exactly 1: %q", len(seasons), namesOf(seasons))
	}
	season := seasons[0]

	if season.IndexNumber == nil || *season.IndexNumber != 0 {
		t.Errorf("season IndexNumber = %s, want 0", number(season.IndexNumber))
	}
	if got, want := delimiter+season.Name+delimiter, delimiter+"Specials"+delimiter; got != want {
		t.Errorf("season name = %s, want %s", got, want)
	}
	if season.Path != "The Series/Specials" {
		t.Errorf("season path = %q, want %q — season zero took a directory that is not its own",
			season.Path, "The Series/Specials")
	}

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1: the extras became episodes: %q",
			len(episodes), namesOf(episodes))
	}
	if episodes[0].ParentID != season.ID {
		t.Errorf("the special's parent is %q, want the season %q", episodes[0].ParentID, season.ID)
	}

	// And the two extras are skipped files rather than items nothing named:
	// 003 §3.8 counts a skip and an unplaceable item apart.
	if len(plan.Skipped) != 2 {
		t.Fatalf("skipped %d files, want the two extras: %+v", len(plan.Skipped), plan.Skipped)
	}
	for _, note := range plan.Skipped {
		if note.Reason != SkipExtrasFolder.String() {
			t.Errorf("%q was skipped as %q, want %q", note.Path, note.Reason, SkipExtrasFolder.String())
		}
	}
	if len(plan.Unplaceable) != 0 {
		t.Errorf("unplaceable = %+v, want none", plan.Unplaceable)
	}
}

// ---------------------------------------------------------------------------
// AC-7 — the series called `24`
// ---------------------------------------------------------------------------

// TestTheSeriesCalledTwentyFourKeepsItsTitleAndTakesItsNumbersFromTheFilename
// is AC-7, over the fixture's own path.
//
// Spec §3.4: "**Ambiguity is resolved by position**, not by preference: the
// pattern is matched against the filename first, then against the parent
// directory. A series called `24` must not have its title read as an episode
// number, and this is exactly where naive scanners fail."
//
// Over this path alone the two orders agree about the season, because the
// directory is `Season 01` and says 1 as well. The two tests below are the ones
// that can tell them apart, and they exist because this one cannot.
func TestTheSeriesCalledTwentyFourKeepsItsTitleAndTakesItsNumbersFromTheFilename(t *testing.T) {
	plan := resolveShowFiles(t, "24/Season 01/24 - S01E01 - 12-00 AM.mkv")

	series := itemsOfKind(plan, KindSeries)
	if len(series) != 1 {
		t.Fatalf("resolved %d series, want 1: %q", len(series), namesOf(series))
	}
	if got, want := delimiter+series[0].Name+delimiter, delimiter+"24"+delimiter; got != want {
		t.Errorf("series name = %s, want %s", got, want)
	}
	if series[0].IndexNumber != nil {
		t.Errorf("the series acquired episode number %s", number(series[0].IndexNumber))
	}
	if series[0].ParentIndexNumber != nil {
		t.Errorf("the series acquired season number %s", number(series[0].ParentIndexNumber))
	}
	if series[0].ProductionYear != nil {
		t.Errorf("the series acquired production year %s", number(series[0].ProductionYear))
	}

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	if episodes[0].IndexNumber == nil || *episodes[0].IndexNumber != 1 {
		t.Errorf("episode IndexNumber = %s, want 1", number(episodes[0].IndexNumber))
	}
	if episodes[0].ParentIndexNumber == nil || *episodes[0].ParentIndexNumber != 1 {
		t.Errorf("episode ParentIndexNumber = %s, want 1", number(episodes[0].ParentIndexNumber))
	}
	if got, want := delimiter+episodes[0].Name+delimiter, delimiter+"12-00 AM"+delimiter; got != want {
		t.Errorf("episode name = %s, want %s", got, want)
	}
}

// TestTheFilenamesSeasonBeatsTheDirectorysWhenTheyDisagree is the mutation the
// clause above names — "matching the directory first" — made able to fail.
//
// # And the finding, because the fixture path alone cannot make it
//
// The task list says the fixture's `24/Season 01/24 - S01E01 - 12-00 AM.mkv` is
// "built to catch exactly that". **It is not, under this resolver, and the
// reason is T5's finding for the third time in this feature: two rules repair
// the mutation.** The directory is `Season 01` and says 1 as loudly as the
// filename does, so both orders agree; and where a `24` tree has no season
// directory — `24/24 - S01E01…` — there is no containing directory below the
// series to match first at all, so the swapped order changes nothing there
// either. What the fixture path really catches is the *other* half of §3.4's
// sentence, that a series' own title is not read as a number, and
// [TestASeriesDirectoryIsNeverAlsoASeasonDirectory] is where that is asserted.
//
// **Only a tree where the two sources disagree can catch the order**, so this
// is that tree: a season directory numbered 5 holding a file whose name says
// season 1. Position decides and the filename wins, which also means the season
// the episode lands in has **no directory of its own** — the number 5 folder is
// not season 1's, and filling season 1's path from it is the second mutation
// this test kills.
func TestTheFilenamesSeasonBeatsTheDirectorysWhenTheyDisagree(t *testing.T) {
	plan := resolveShowFiles(t, "The Series/Season 05/The Series - S01E01 - Pilot.mkv")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	if got := episodes[0].ParentIndexNumber; got == nil || *got != 1 {
		t.Fatalf("ParentIndexNumber = %s, want 1 — the directory was matched before the filename",
			number(episodes[0].ParentIndexNumber))
	}

	seasons := itemsOfKind(plan, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want 1: %q", len(seasons), namesOf(seasons))
	}
	if got := seasons[0].IndexNumber; got == nil || *got != 1 {
		t.Errorf("season IndexNumber = %s, want 1", number(seasons[0].IndexNumber))
	}
	if got, want := delimiter+seasons[0].Name+delimiter, delimiter+"Season 1"+delimiter; got != want {
		t.Errorf("season name = %s, want %s", got, want)
	}
	if seasons[0].Path != "" {
		t.Errorf("season 1 carries path %q, which is season 5's directory", seasons[0].Path)
	}
	if episodes[0].ParentID != seasons[0].ID {
		t.Errorf("the episode is parented to %q and the one season is %q", episodes[0].ParentID, seasons[0].ID)
	}
}

// TestASeriesDirectoryIsNeverAlsoASeasonDirectory is the same rule where the
// filename cannot rescue it: the name carries an episode number and **no**
// season, so the only thing that can supply one is a directory.
//
// The series directory must not be that directory, however numeric its name.
// The reference never asks it either — its season parser runs only on a
// directory whose parent is a `Series`
// `[source: Emby.Server.Implementations/Library/Resolvers/TV/SeasonResolver.cs:45 @ v10.11.11]`
// — and with no season folder and an episode number in hand, the season is 1
// `[…/EpisodeResolver.cs:78-82 @ v10.11.11]`.
func TestASeriesDirectoryIsNeverAlsoASeasonDirectory(t *testing.T) {
	plan := resolveShowFiles(t, "24/24 - EP05 - Day One.mkv")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	if got := episodes[0].IndexNumber; got == nil || *got != 5 {
		t.Fatalf("IndexNumber = %s, want 5", number(episodes[0].IndexNumber))
	}
	if got := episodes[0].ParentIndexNumber; got == nil || *got != 1 {
		t.Errorf("ParentIndexNumber = %s, want 1 — the series directory `24` was read as season 24",
			number(episodes[0].ParentIndexNumber))
	}
}

// ---------------------------------------------------------------------------
// A season inferred where no directory exists (§3.4, §3.6)
// ---------------------------------------------------------------------------

// TestASeasonInferredWhereNoDirectoryExistsIsKeyedOnItsSeriesAndItsNumber is
// spec §3.4's second table row and §3.6's identity rule at once.
//
// The season has **no path to derive an identifier from**, which is the whole
// reason §3.6 keys a season on "its series' identity plus its season number".
// The assertion that makes that a statement rather than a hope is the second
// tree: the same series and the same number, reached through a season
// *directory* instead, is the **same item**. A build that derived a season's
// identifier from its directory passes every count and every name here and
// fails only this.
func TestASeasonInferredWhereNoDirectoryExistsIsKeyedOnItsSeriesAndItsNumber(t *testing.T) {
	inferred := resolveShowFiles(t, "The Show/The Show - S04E01 - Pilot.mkv")

	seasons := itemsOfKind(inferred, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want 1: %q", len(seasons), namesOf(seasons))
	}
	season := seasons[0]

	if season.IndexNumber == nil || *season.IndexNumber != 4 {
		t.Fatalf("season IndexNumber = %s, want 4", number(season.IndexNumber))
	}
	if season.Path != "" {
		t.Errorf("the inferred season carries path %q; it has no directory of its own", season.Path)
	}
	if got, want := delimiter+season.Name+delimiter, delimiter+"Season 4"+delimiter; got != want {
		t.Errorf("season name = %s, want %s", got, want)
	}
	if season.SortKey != "0004" {
		t.Errorf("season sort key = %q, want %q (003 §3.7.2)", season.SortKey, "0004")
	}

	series := itemsOfKind(inferred, KindSeries)
	if len(series) != 1 {
		t.Fatalf("resolved %d series, want 1", len(series))
	}
	if want := DeriveID(aShowsLibrary().ID, KindSeason, joinKey(series[0].ID, "4")); season.ID != want {
		t.Errorf("season identifier = %q, want the one derived from the series and the number, %q",
			season.ID, want)
	}

	// The same season, reached through a directory instead, is the same item.
	withDirectory := resolveShowFiles(t, "The Show/Season 04/The Show - S04E01 - Pilot.mkv")
	directorySeasons := itemsOfKind(withDirectory, KindSeason)
	if len(directorySeasons) != 1 {
		t.Fatalf("resolved %d seasons from the directory tree, want 1", len(directorySeasons))
	}
	if directorySeasons[0].ID != season.ID {
		t.Errorf("the directory-backed season is %q and the inferred one is %q; a season's identity is "+
			"its series plus its number and not its path", directorySeasons[0].ID, season.ID)
	}
	if directorySeasons[0].Path != "The Show/Season 04" {
		t.Errorf("the directory-backed season's path = %q, want %q",
			directorySeasons[0].Path, "The Show/Season 04")
	}
}

// TestAnInferredSeasonAndItsDirectoryAreOneItem is the fixture's own shape:
// `The Series - S02E01 - No Season Directory.mkv` sits beside a
// `The Series/Season 02/` that holds another episode, and the two are **one**
// season because §3.6 keys a season on its series plus its number.
//
// The season a candidate reached without a directory still ends up with the
// directory's path, and the episode that had no directory is parented to it.
//
// **This is not an assertion about entry order**, and it was one until the
// mutation that should have failed it did not. [Resolve] sorts every reading
// before anything looks at it — T5's own test owns that — so two orders of the
// same paths are literally the same input by the time this resolver runs, and a
// loop over both of them proved one thing twice. What has to be right is the
// sorted order, and it is: of the two candidates, only one has a directory to
// give.
func TestAnInferredSeasonAndItsDirectoryAreOneItem(t *testing.T) {
	plan := resolveShowFiles(t,
		"The Series/The Series - S02E01 - No Season Directory.mkv",
		"The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4",
	)

	seasons := itemsOfKind(plan, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want 1: %q", len(seasons), namesOf(seasons))
	}
	if seasons[0].Path != "The Series/Season 02" {
		t.Errorf("season path = %q, want %q", seasons[0].Path, "The Series/Season 02")
	}

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 2 {
		t.Fatalf("resolved %d episodes, want 2", len(episodes))
	}
	for _, episode := range episodes {
		if episode.ParentID != seasons[0].ID {
			t.Errorf("%q is parented to %q, not to the one season %q",
				episode.Path, episode.ParentID, seasons[0].ID)
		}
	}
}

// ---------------------------------------------------------------------------
// §3.8 — unplaceable is not skipped
// ---------------------------------------------------------------------------

// TestAStrayFileInASeasonDirectoryIsUnplaceableAndNotSkipped is spec §3.8's
// distinction, over the one file in the fixture tree that exercises it.
//
// §3.8 counts the two apart "because an operator told that both were skipped
// would go looking for something that is not missing". So `blob.mkv` is an
// **item**: it is in the library, it has an identifier, it is under its season,
// and what it does not have is a number. The assertion is made in both
// directions — it is in [Plan.Unplaceable] and it is **not** in [Plan.Skipped]
// — because a build that dropped it entirely satisfies "it is not skipped".
func TestAStrayFileInASeasonDirectoryIsUnplaceableAndNotSkipped(t *testing.T) {
	const stray = "The Series/Season 01/blob.mkv"

	plan := resolveShowFiles(t,
		"The Series/Season 01/The Series - S01E01 - Pilot.mkv",
		stray,
	)

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 2 {
		t.Fatalf("resolved %d episodes, want 2 — the stray file was dropped: %q",
			len(episodes), namesOf(episodes))
	}

	var blob *ports.ScannedItem
	for i := range episodes {
		if episodes[i].Path == stray {
			blob = &episodes[i]
		}
	}
	if blob == nil {
		t.Fatalf("no item for %q", stray)
	}

	if !blob.Unplaceable {
		t.Error("the item is not marked unplaceable")
	}
	if blob.IndexNumber != nil {
		t.Errorf("IndexNumber = %s, want absent", number(blob.IndexNumber))
	}
	if got, want := delimiter+blob.Name+delimiter, delimiter+"blob"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
	if blob.ID == "" {
		t.Error("the item has no identifier, so nothing can key anything on it")
	}

	if len(plan.Skipped) != 0 {
		t.Errorf("skipped = %+v, want none: an unplaceable file is in the library", plan.Skipped)
	}
	if len(plan.Unplaceable) != 1 || plan.Unplaceable[0].Path != stray {
		t.Fatalf("unplaceable = %+v, want the one stray file", plan.Unplaceable)
	}
	if got, want := plan.Unplaceable[0].Reason, ReasonNoEpisodeNumber; got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}

	// And it is still placed: the season directory says where it sits even
	// though its name says nothing about which episode it is.
	seasons := itemsOfKind(plan, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want 1", len(seasons))
	}
	if blob.ParentID != seasons[0].ID {
		t.Errorf("the stray file's parent is %q, want the season %q", blob.ParentID, seasons[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Date-based naming, and a number beyond any real count
// ---------------------------------------------------------------------------

// TestADateNamedEpisodeResolvesByDate is spec §3.4's daily show.
//
// It has a premiere date, **no** episode number, and is not unplaceable: a date
// says which episode this is as well as a number does. It also keeps its whole
// name, because an episode is called what follows the numbering and nothing
// follows a date that ends the stem — see [episodeName], and note that a build
// naming an episode after the last hyphenated segment agrees with every other
// row in this file and calls this one `2024-01-31`.
func TestADateNamedEpisodeResolvesByDate(t *testing.T) {
	plan := resolveShowFiles(t, "The Daily Show/Season 2024/The Daily Show - 2024-01-31.mkv")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	episode := episodes[0]

	if episode.PremiereDate == nil {
		t.Fatal("PremiereDate is absent")
	}
	want := units.At(time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC))
	if !episode.PremiereDate.Equal(want) {
		t.Errorf("PremiereDate = %s, want %s", episode.PremiereDate, want)
	}
	if episode.IndexNumber != nil {
		t.Errorf("IndexNumber = %s, want absent: a date is not an episode number", number(episode.IndexNumber))
	}
	if episode.Unplaceable {
		t.Error("a date-named episode is marked unplaceable")
	}
	if got, want := delimiter+episode.Name+delimiter, delimiter+"The Daily Show - 2024-01-31"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
	if episode.ProductionYear != nil {
		t.Errorf("ProductionYear = %s, want absent — the date's own year was read as a production year",
			number(episode.ProductionYear))
	}

	seasons := itemsOfKind(plan, KindSeason)
	if len(seasons) != 1 {
		t.Fatalf("resolved %d seasons, want 1", len(seasons))
	}
	if got := seasons[0].IndexNumber; got == nil || *got != 2024 {
		t.Errorf("season IndexNumber = %s, want 2024", number(seasons[0].IndexNumber))
	}
}

// TestAnEpisodeNumberBeyondAnyRealCountIsNotAnError is spec §3.4's closing
// paragraph: "an episode whose number exceeds any real count… None of these is
// an error; all of them appear in real libraries."
func TestAnEpisodeNumberBeyondAnyRealCountIsNotAnError(t *testing.T) {
	plan := resolveShowFiles(t, "The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	if got := episodes[0].IndexNumber; got == nil || *got != 99 {
		t.Fatalf("IndexNumber = %s, want 99", number(episodes[0].IndexNumber))
	}
	if episodes[0].Unplaceable {
		t.Error("the item is marked unplaceable")
	}
	if len(plan.Unplaceable) != 0 {
		t.Errorf("unplaceable = %+v, want none", plan.Unplaceable)
	}
	if got, want := delimiter+episodes[0].Name+delimiter, delimiter+"Beyond Any Real Count"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// The rules under the resolver
// ---------------------------------------------------------------------------

// TestTheNumberingFamilyIsTheOneTheSpecificationNames walks 003 §3.4's family —
// "`S01E02` and its separators, `1x02`, `E02`/`EP02`, and date-based naming" —
// and the shapes each rule has to refuse.
func TestTheNumberingFamilyIsTheOneTheSpecificationNames(t *testing.T) {
	type want struct {
		season, episode, end *int
		date                 string
	}
	for _, row := range []struct {
		stem string
		want want
	}{
		{"The Series - S01E02", want{season: intPointer(1), episode: intPointer(2)}},
		{"The Series - s01e02", want{season: intPointer(1), episode: intPointer(2)}},
		{"The Series - S01 - E02", want{season: intPointer(1), episode: intPointer(2)}},
		{"The.Series.S01.E02", want{season: intPointer(1), episode: intPointer(2)}},
		{"The Series S01xE02", want{season: intPointer(1), episode: intPointer(2)}},
		{"Series Season 1 Episode 2", want{season: intPointer(1), episode: intPointer(2)}},
		{"The Series - 1x02", want{season: intPointer(1), episode: intPointer(2)}},
		{"The Series - EP02", want{episode: intPointer(2)}},
		{"The Series - ep_02", want{episode: intPointer(2)}},
		{"The Series - E02", want{episode: intPointer(2)}},
		{"The Series - 2024-01-31", want{date: "2024-01-31"}},
		{"The Series - 2024.01.31", want{date: "2024-01-31"}},
		{"The Series - 31-01-2024", want{date: "2024-01-31"}},

		// What the family refuses. Each is a file the reference numbers and
		// this server leaves unplaceable, or one neither numbers.
		{"blob", want{}},
		{"01 - Pilot", want{}},
		{"The Series - E02 - Pilot", want{}},
		{"Series Special (1920x1080)", want{}},
		{"The Series - 2024-13-31", want{}},
		{"The Series - 2024-02-31", want{}},
	} {
		got := parseEpisodeNumbering(row.stem)

		if number(got.season) != number(row.want.season) {
			t.Errorf("%q: season = %s, want %s", row.stem, number(got.season), number(row.want.season))
		}
		if number(got.episode) != number(row.want.episode) {
			t.Errorf("%q: episode = %s, want %s", row.stem, number(got.episode), number(row.want.episode))
		}
		if number(got.episodeEnd) != number(row.want.end) {
			t.Errorf("%q: ending episode = %s, want %s", row.stem, number(got.episodeEnd), number(row.want.end))
		}

		date := ""
		if got.premiere != nil {
			date = got.premiere.Instant().Format("2006-01-02")
		}
		if date != row.want.date {
			t.Errorf("%q: date = %q, want %q", row.stem, date, row.want.date)
		}
	}
}

// TestASeasonDirectorysNumberIsReadTheWayTheReferenceReadsIt is
// [parseSeasonDirectoryName]'s table.
//
// The last three rows are the ones with a failure behind them: `24` is a series
// directory whose name would be a season number anywhere else, `Extras` and
// `Featurettes` must not be seasons, and `Season 1 1080p` is the resolution
// guard the reference's own non-greedy number carries
// `[source: Emby.Naming/TV/SeasonPathParser.cs:18 @ v10.11.11]`.
func TestASeasonDirectorysNumberIsReadTheWayTheReferenceReadsIt(t *testing.T) {
	for _, row := range []struct {
		name, series string
		want         *int
	}{
		{name: "Season 01", want: intPointer(1)},
		{name: "Season 2024", want: intPointer(2024)},
		{name: "season 3", want: intPointer(3)},
		{name: "S01", want: intPointer(1)},
		{name: "Specials", want: intPointer(0)},
		{name: "specials", want: intPointer(0)},
		{name: "07", want: intPointer(7)},
		{name: "Staffel 2", want: intPointer(2)},
		{name: "Temporada 2", want: intPointer(2)},
		{name: "1 Season", want: intPointer(1)},
		{name: "1st Season", want: intPointer(1)},
		{name: "The Series Season 5", series: "The Series", want: intPointer(5)},
		{name: "Season 1 1080p", want: intPointer(1)},

		{name: "24", series: "", want: intPointer(24)},
		{name: "Extras", want: nil},
		{name: "Featurettes", want: nil},
		{name: "Behind The Scenes", want: nil},
		{name: "S01E01", want: nil},
		{name: "Season 1 E01", want: nil},
		{name: "The Series", want: nil},
		{name: "Sherlock", want: nil},
	} {
		got, ok := parseSeasonDirectoryName(row.name, row.series)
		if !ok {
			if row.want != nil {
				t.Errorf("%q under %q: no season, want %s", row.name, row.series, number(row.want))
			}
			continue
		}
		if row.want == nil {
			t.Errorf("%q under %q: season %d, want none", row.name, row.series, got)
			continue
		}
		if got != *row.want {
			t.Errorf("%q under %q: season %d, want %d", row.name, row.series, got, *row.want)
		}
	}
}

// TestAnEpisodeIsNamedAfterWhatFollowsItsNumberingAndNothingElse is the naming
// rule on a tree where **nothing else can repair a wrong answer**, which is
// T5's finding applied rather than repeated: over the fixture tree an episode's
// name is also its last hyphenated segment, also what a year rule would leave,
// and also what a directory could supply.
//
// Here the numbering sits in the middle of a name whose two halves are both
// plausible titles and whose tail contains its own hyphen. A build that took
// the text **before** the numbering answers `Interview`; one that took the last
// hyphenated segment answers `A Sequel`; one that took the whole stem answers
// all of it. Only "what follows the numbering" answers the expected value.
func TestAnEpisodeIsNamedAfterWhatFollowsItsNumberingAndNothingElse(t *testing.T) {
	plan := resolveShowFiles(t, "Interview/Season 01/Interview - S01E07 - Aftermath - A Sequel.mkv")

	episodes := itemsOfKind(plan, KindEpisode)
	if len(episodes) != 1 {
		t.Fatalf("resolved %d episodes, want 1", len(episodes))
	}
	if got, want := delimiter+episodes[0].Name+delimiter, delimiter+"Aftermath - A Sequel"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}

	// And the sort key is 003 §3.7.2's, which is where the name actually
	// shows up in an order a client sees: season padded to three, episode to
	// four, then the **raw** name.
	if got, want := episodes[0].SortKey, "001 - 0007 - Aftermath - A Sequel"; got != want {
		t.Errorf("sort key = %q, want %q", got, want)
	}
}

// TestACandidateWithNoSeriesAboveItIsUnplaceableRatherThanAnEpisodeOfNothing
// covers the other reason a `tvshows` candidate cannot be placed: it sits
// directly under a library root, so no directory names its series.
//
// The reference has the same two cases and answers the first the same way — a
// flat `tvshows` root is resolved by taking the series name out of the filename
// `[source: Emby.Naming/Common/NamingOptions.cs:324,
// Emby.Server.Implementations/Library/Resolvers/TV/EpisodeResolver.cs:52-54 @ v10.11.11]`.
func TestACandidateWithNoSeriesAboveItIsUnplaceableRatherThanAnEpisodeOfNothing(t *testing.T) {
	plan := resolveShowFiles(t, "The Show - S01E02 - Named By Its File.mkv", "S03E04.mkv")

	series := itemsOfKind(plan, KindSeries)
	if len(series) != 1 {
		t.Fatalf("resolved %d series, want 1: %q", len(series), namesOf(series))
	}
	if got, want := delimiter+series[0].Name+delimiter, delimiter+"The Show"+delimiter; got != want {
		t.Errorf("series name = %s, want %s", got, want)
	}
	if series[0].Path != "" {
		t.Errorf("the series carries path %q; it has no directory", series[0].Path)
	}

	if len(plan.Unplaceable) != 1 || plan.Unplaceable[0].Path != "S03E04.mkv" {
		t.Fatalf("unplaceable = %+v, want the one file that names no series", plan.Unplaceable)
	}
	if got, want := plan.Unplaceable[0].Reason, ReasonNoSeries; got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}
}

// TestTwoReadingsOfOneTreeInOppositeOrdersResolveIdentically is Principle VII
// over the shapes in this file that a map or a first-writer-wins rule could
// make order-dependent: the season path, the series path, and which of two
// series a shared season number belongs to.
func TestTwoReadingsOfOneTreeInOppositeOrdersResolveIdentically(t *testing.T) {
	paths := []string{
		"24/24 - S01E01 - 12-00 AM.mkv",
		"The Series/Season 01/The Series - S01E01 - Pilot.mkv",
		"The Series/Season 01/blob.mkv",
		"The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4",
		"The Series/Specials/The Series - S00E01 - A Special.mkv",
		"The Series/The Series - S02E01 - No Season Directory.mkv",
	}
	reversed := make([]string, len(paths))
	for i, p := range paths {
		reversed[len(paths)-1-i] = p
	}

	forward := resolveShowFiles(t, paths...)
	backward := resolveShowFiles(t, reversed...)

	if len(forward.Items) != len(backward.Items) {
		t.Fatalf("%d items forward and %d backward", len(forward.Items), len(backward.Items))
	}
	for i := range forward.Items {
		a, b := forward.Items[i], backward.Items[i]
		if a.ID != b.ID || a.Name != b.Name || a.Path != b.Path || a.ParentID != b.ParentID ||
			a.SortKey != b.SortKey || number(a.IndexNumber) != number(b.IndexNumber) {
			t.Errorf("item %d differs by entry order:\n forward  %+v\n backward %+v", i, a, b)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-1's `tvshows` third
// ---------------------------------------------------------------------------

// TestTheFixturesShowsResolveToTheExpectedRows compares this resolver against
// the item set 003 plan §8.5 keeps as a **literal**.
//
// It is compared as a **set** and not positionally: the literal is grouped by
// type and [Resolve] orders by path, and a positional comparison would have
// been a test of two orderings agreeing rather than of two item sets. The
// reading is a literal too, for the reason `resolve_test.go` gives — deriving
// it from `libraryfixture.Libraries()` would make this a test of arithmetic —
// and it is not a scan: the fixture's `.mka` file and its empty
// `The Series/Season 03` directory are absent because a **walk** drops them and
// the walk is not this task's.
func TestTheFixturesShowsResolveToTheExpectedRows(t *testing.T) {
	reading := aReading(0,
		"24/Season 01/24 - S01E01 - 12-00 AM.mkv",
		"The Daily Show/Season 2024/The Daily Show - 2024-01-31.mkv",
		"The Series/Season 01/The Series - S01E01 - Pilot.mkv",
		"The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv",
		"The Series/Season 01/The Series - S01E04 - Old Transfer.avi",
		"The Series/Season 01/blob.mkv",
		"The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4",
		"The Series/Specials/The Series - S00E01 - A Special.mkv",
		"The Series/The Series - S02E01 - No Season Directory.mkv",
	)

	plan, err := Resolve(aShowsLibrary(), []Reading{reading})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	var want []libraryfixture.ExpectedItem
	for _, row := range libraryfixture.ExpectedItems() {
		if row.Library == "Shows" {
			want = append(want, row)
		}
	}
	if len(plan.Items) != len(want) {
		t.Fatalf("resolved %d items, want %d:\n got  %q\n want %q",
			len(plan.Items), len(want), describeItems(plan.Items), describeExpected(want))
	}

	// The parent of every expected row is named by its **path**, so the
	// identifier a row expects is looked up from the resolved item that has
	// that path — which is what makes "the expected parent-child structure" an
	// assertion about the tree rather than about a string.
	byPath := map[string]ports.ScannedItem{}
	for _, item := range plan.Items {
		if Kind(item.Type) != KindCollectionFolder {
			byPath[item.Path] = item
		}
	}

	got := describeItems(plan.Items)
	sort.Strings(got)
	expected := describeExpected(want)
	sort.Strings(expected)
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("row %d:\n got  %s\n want %s", i, got[i], expected[i])
		}
	}

	for _, row := range want {
		if row.Parent == "" {
			continue
		}
		item, ok := byPath[row.Path]
		if !ok {
			continue
		}
		wantParent := plan.Items[0].ID
		if row.Parent != libraryfixture.LibraryRoot {
			parent, ok := byPath[row.Parent]
			if !ok {
				t.Errorf("%q expects parent %q and nothing resolved to that path", row.Path, row.Parent)
				continue
			}
			wantParent = parent.ID
		}
		if item.ParentID != wantParent {
			t.Errorf("%q: parent %q, want %q (%q)", row.Path, item.ParentID, wantParent, row.Parent)
		}
	}

	// The library's own row is the only item with no path, and it is what the
	// three series hang from.
	if plan.Items[0].Type != string(KindCollectionFolder) {
		t.Errorf("the first item is %q, want the library's own row", plan.Items[0].Type)
	}
}

// describeItems renders the four fields the expected set carries, between
// delimiters so that a lost or gained space is visible in a failure message.
func describeItems(items []ports.ScannedItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = delimiter + item.Type + delimiter + item.Name + delimiter + item.Path +
			delimiter + strconv.FormatBool(item.Unplaceable) + delimiter
	}
	return out
}

func describeExpected(rows []libraryfixture.ExpectedItem) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = delimiter + row.Type + delimiter + row.Name + delimiter + row.Path +
			delimiter + strconv.FormatBool(row.Unplaceable) + delimiter
	}
	return out
}
