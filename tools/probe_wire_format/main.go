// Command probe_wire_format measures the reference's wire format: how dates are
// serialised, whether tick fields are integers, and whether property names are
// PascalCase.
//
// It discharges the register debt "Dates carry seven fractional digits", which
// behaviours 1.2 has carried as a prior-probe since 2026-06-19.
//
// Read-only. It issues GETs and one sign-in, writes nothing, and needs no
// fixture and no second identity.
//
//	go run ./tools/probe_wire_format
//	go run ./tools/probe_wire_format -url http://192.168.1.39:7096
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/vdatanet/atrium-go/tools/internal/probe"
)

// A value that looks like a date at all: the shape the reference emits before
// the fractional part is examined.
var dateLike = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

// The exact shape behaviours 1.2 claims: seven fractional digits and a Z.
var sevenDigitsZ = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{7}Z$`)

var fractional = regexp.MustCompile(`\.(\d+)`)

// A dictionary key that is an identifier rather than a property name.
var hexKey = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Routes with dated, ticked and named properties. All reads.
var routes = []string{
	"/System/Info/Public",
	"/System/Info",
	"/Users/Me",
	"/UserViews",
	"/Items?Recursive=true&Limit=60&SortBy=DateCreated&SortOrder=Descending",
	"/Items?Recursive=true&IncludeItemTypes=Audio&Limit=60",
	"/Items?Recursive=true&IncludeItemTypes=Movie,Episode,Series&Limit=60",
	"/Items/Latest?Limit=30",
	"/Sessions",
	"/Artists?Limit=30",
	"/Genres?Limit=30",
}

type finding struct {
	key, value, route string
}

func main() {
	openSession := probe.Flags()
	flag.Parse()
	s, err := openSession()
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe_wire_format:", err)
		os.Exit(2)
	}
	fmt.Printf("probe_wire_format against %s\n\n", s.BaseURL)

	// Observations, keyed by property name so the report is per field.
	dateKeys := map[string]map[int]int{}      // key -> fractional digit count -> times seen
	dateSample := map[string]map[int]string{} // key -> digit count -> one example value
	dateNoZ := []finding{}
	tickKeys := map[string]int{}
	tickNonInteger := []finding{}
	names := map[string]bool{}
	bodies := 0

	for _, route := range routes {
		status, body, err := s.Get(route)
		if err != nil {
			fmt.Printf("  %-70s ERROR %v\n", route, err)
			continue
		}
		if status != 200 {
			fmt.Printf("  %-70s %d, skipped\n", route, status)
			continue
		}
		doc, err := probe.Decode(body)
		if err != nil {
			fmt.Printf("  %-70s undecodable: %v\n", route, err)
			continue
		}
		bodies++
		probe.Walk(doc, "", func(key string, val any) {
			names[key] = true
			switch v := val.(type) {
			case string:
				if !dateLike.MatchString(v) {
					return
				}
				if dateKeys[key] == nil {
					dateKeys[key] = map[int]int{}
				}
				digits := 0
				if m := fractional.FindStringSubmatch(v); m != nil {
					digits = len(m[1])
				}
				dateKeys[key][digits]++
				if dateSample[key] == nil {
					dateSample[key] = map[int]string{}
				}
				if _, seen := dateSample[key][digits]; !seen {
					dateSample[key][digits] = v
				}
				if !sevenDigitsZ.MatchString(v) {
					dateNoZ = append(dateNoZ, finding{key, v, route})
				}
			case json.Number:
				if !strings.HasSuffix(key, "Ticks") {
					return
				}
				tickKeys[key]++
				if strings.ContainsAny(v.String(), ".eE") {
					tickNonInteger = append(tickNonInteger, finding{key, v.String(), route})
				}
			}
		})
	}

	fmt.Printf("\n%d bodies read, %d distinct property names seen\n", bodies, len(names))

	// --- The claim under test -----------------------------------------------
	fmt.Println("\n== dates: do they carry seven fractional digits and a Z? ==")
	total, conforming := 0, 0
	for _, key := range sorted(dateKeys) {
		counts := dateKeys[key]
		n := 0
		for _, d := range sortedInts(counts) {
			n += counts[d]
			if d == 7 {
				conforming += counts[d]
			}
			fmt.Printf("  %-24s %d digits x%-4d %s\n", key, d, counts[d], dateSample[key][d])
		}
		total += n
	}
	if total == 0 {
		fmt.Println("  no date-shaped values found — the claim cannot be settled from these routes")
		os.Exit(1)
	}
	fmt.Printf("\n  %d of %d date values match the exact shape (seven digits and a Z)\n", conforming, total)
	for _, f := range first(dateNoZ, 10) {
		fmt.Printf("    counterexample: %s = %q  (%s)\n", f.key, f.value, f.route)
	}

	// --- What the stated L1 sweep would and would not cover ------------------
	// conformance.md's unit sweep says "every field whose name ends in Date".
	fmt.Println("\n== which date fields does \"ends in Date\" actually reach? ==")
	var reached, missed []string
	for _, key := range sorted(dateKeys) {
		if strings.HasSuffix(key, "Date") {
			reached = append(reached, key)
		} else {
			missed = append(missed, key)
		}
	}
	fmt.Printf("  reached (%d): %s\n", len(reached), strings.Join(reached, ", "))
	fmt.Printf("  MISSED  (%d): %s\n", len(missed), strings.Join(missed, ", "))
	if len(missed) > 0 {
		fmt.Println("  A sweep keyed on the suffix alone does not see these. They are date values by")
		fmt.Println("  measurement, and the rule as written in conformance.md would not check them.")
	}

	// --- The two other cross-cutting facts, free from the same bodies --------
	fmt.Println("\n== ticks: is every *Ticks value an integer? ==")
	n := 0
	for _, k := range sortedCount(tickKeys) {
		fmt.Printf("  %-28s x%d\n", k, tickKeys[k])
		n += tickKeys[k]
	}
	fmt.Printf("  %d values across %d fields; %d non-integer\n", n, len(tickKeys), len(tickNonInteger))
	for _, f := range first(tickNonInteger, 10) {
		fmt.Printf("    counterexample: %s = %s  (%s)\n", f.key, f.value, f.route)
	}

	fmt.Println("\n== property names: is every one PascalCase? ==")
	// A dictionary's KEYS are data, not schema. ImageBlurHashes is keyed by image
	// tag, so a sweep that treats every map key as a property name reports a few
	// hundred hex strings as casing failures. The L1 casing sweep has to make the
	// same distinction, which is why this is measured rather than filtered quietly.
	var notPascal, dataKeys []string
	for k := range names {
		if k == "" {
			continue
		}
		if hexKey.MatchString(k) {
			dataKeys = append(dataKeys, k)
			continue
		}
		if c := k[0]; c < 'A' || c > 'Z' {
			notPascal = append(notPascal, k)
		}
	}
	fmt.Printf("  %d of %d keys are 32-hex dictionary keys (data, not property names) and are excluded\n",
		len(dataKeys), len(names))
	sort.Strings(notPascal)
	if len(notPascal) == 0 {
		fmt.Printf("  every remaining name begins with an upper-case letter\n")
	} else {
		fmt.Printf("  %d do not: %s\n", len(notPascal), strings.Join(first(notPascal, 20), ", "))
	}

	if conforming != total || len(tickNonInteger) > 0 {
		os.Exit(1)
	}
}

func sorted[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCount(m map[string]int) []string { return sorted(m) }

func sortedInts(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func first[T any](s []T, n int) []T {
	if len(s) < n {
		return s
	}
	return s[:n]
}
