// Command probe_sort_vocabulary measures which SortBy tokens the reference
// actually honours.
//
// It discharges the register debt "The SortBy vocabulary", open since
// 2026-06-13. What behaviours 2.5 records is that eight tokens order rows; what
// nobody had measured is the CLOSURE of the set — whether a token outside those
// eight orders anything. The reference's own enumeration names thirty
// [source: Jellyfin.Data/Enums/ItemSortBy.cs @ v10.11.11], an unrecognised token
// is ignored rather than refused (behaviours 1.12), and a shipping music client
// sends three that are not among the eight: ParentIndexNumber, IndexNumber and
// ProductionYear (client-embeat-mobile 5.8).
//
// Method: for each token, ask the same query ascending and descending. A token
// that orders answers two different sequences; a token that is ignored answers
// the default order both times, which is also what a token that cannot exist
// answers. The control is a token no enumeration contains.
//
// Read-only.
//
//	go run ./tools/probe_sort_vocabulary
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vdatanet/atrium-go/tools/internal/probe"
)

// The reference's whole enumeration, read at the pinned tag.
var enumeration = []string{
	"Default", "AiredEpisodeOrder", "Album", "AlbumArtist", "Artist", "DateCreated",
	"OfficialRating", "DatePlayed", "PremiereDate", "StartDate", "SortName", "Name",
	"Random", "Runtime", "CommunityRating", "ProductionYear", "PlayCount", "CriticRating",
	"IsFolder", "IsUnplayed", "IsPlayed", "SeriesSortName", "VideoBitRate", "AirTime",
	"Studio", "IsFavoriteOrLiked", "DateLastContentAdded", "SeriesDatePlayed",
	"ParentIndexNumber", "IndexNumber",
}

// The eight behaviours 2.5 records as supported.
var recorded = map[string]bool{
	"SortName": true, "DateCreated": true, "PremiereDate": true, "PlayCount": true,
	"DatePlayed": true, "Random": true, "AlbumArtist": true, "Artist": true,
}

// A token no enumeration contains. Whatever this answers is what "ignored" looks like.
const control = "AtriumNotASortToken"

func main() {
	openSession := probe.Flags()
	query := flag.String("query", "/Items?Recursive=true&IncludeItemTypes=Audio&Limit=40", "the query to order")
	flag.Parse()
	s, err := openSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe_sort_vocabulary:", err)
		os.Exit(2)
	}
	fmt.Printf("probe_sort_vocabulary against %s\n  query: %s\n\n", s.BaseURL, *query)

	ids := func(token, order string) (string, error) {
		_, body, err := s.Get(fmt.Sprintf("%s&SortBy=%s&SortOrder=%s", *query, token, order))
		if err != nil {
			return "", err
		}
		doc, err := probe.Decode(body)
		if err != nil {
			return "", err
		}
		var out []string
		root, _ := doc.(map[string]any)
		items, _ := root["Items"].([]any)
		for _, it := range items {
			if m, ok := it.(map[string]any); ok {
				if id, ok := m["Id"].(string); ok {
					out = append(out, id)
				}
			}
		}
		return strings.Join(out, ","), nil
	}

	baseline, err := ids(control, "Ascending")
	if err != nil || baseline == "" {
		fmt.Fprintln(os.Stderr, "  the control query returned no items; pick another -query:", err)
		os.Exit(2)
	}
	baselineAgain, _ := ids(control, "Descending")
	if baseline != baselineAgain {
		fmt.Println("  NOTE: the control token's order changes with SortOrder, so the default order")
		fmt.Println("  is itself direction-sensitive. Classification below still holds.")
	}
	fmt.Printf("  control %q returned %d rows; that sequence is what \"ignored\" looks like\n\n",
		control, strings.Count(baseline, ",")+1)

	var honoured, ignored, random []string
	fmt.Printf("  %-22s %-10s %s\n", "TOKEN", "VERDICT", "NOTE")
	for _, token := range enumeration {
		asc, err := ids(token, "Ascending")
		if err != nil {
			fmt.Printf("  %-22s %-10s %v\n", token, "ERROR", err)
			continue
		}
		desc, _ := ids(token, "Descending")
		ascAgain, _ := ids(token, "Ascending")

		mark := " "
		if !recorded[token] {
			mark = "*" // outside the eight behaviours 2.5 records
		}
		switch {
		case asc != ascAgain:
			random = append(random, token)
			fmt.Printf(" %s%-22s %-10s two identical requests differ\n", mark, token, "RANDOM")
		case asc != desc:
			honoured = append(honoured, token)
			fmt.Printf(" %s%-22s %-10s ascending and descending differ\n", mark, token, "ORDERS")
		case asc == baseline:
			ignored = append(ignored, token)
			fmt.Printf(" %s%-22s %-10s same sequence as the control\n", mark, token, "IGNORED")
		default:
			honoured = append(honoured, token)
			fmt.Printf(" %s%-22s %-10s differs from the control, direction-insensitive\n", mark, token, "ORDERS")
		}
	}

	fmt.Printf("\n  * = outside the eight behaviours 2.5 records\n")
	fmt.Printf("\n  orders: %d   ignored: %d   random: %d   (of %d in the reference's enumeration)\n",
		len(honoured), len(ignored), len(random), len(enumeration))

	// The question the register asks: is the recorded set closed?
	var extra []string
	for _, t := range append(append([]string{}, honoured...), random...) {
		if !recorded[t] {
			extra = append(extra, t)
		}
	}
	var missing []string
	for t := range recorded {
		found := false
		for _, h := range append(append([]string{}, honoured...), random...) {
			if h == t {
				found = true
			}
		}
		if !found {
			missing = append(missing, t)
		}
	}
	fmt.Println("\n== is the recorded vocabulary closed? ==")
	if len(extra) == 0 {
		fmt.Println("  YES — no token outside the recorded eight ordered anything.")
	} else {
		fmt.Printf("  NO — %d tokens outside the recorded eight order rows: %s\n", len(extra), strings.Join(extra, ", "))
		fmt.Println("  behaviours 2.5 records a subset, not the vocabulary.")
	}
	if len(missing) > 0 {
		fmt.Printf("  And %d of the recorded eight did NOT order this query: %s\n", len(missing), strings.Join(missing, ", "))
		fmt.Println("  (a sort key absent from these rows cannot order them; try -query with another type)")
	}
	fmt.Println("\n  The three a shipping music client sends, specifically:")
	for _, t := range []string{"ParentIndexNumber", "IndexNumber", "ProductionYear"} {
		verdict := "IGNORED"
		for _, h := range honoured {
			if h == t {
				verdict = "ORDERS"
			}
		}
		fmt.Printf("    %-20s %s\n", t, verdict)
	}
}
