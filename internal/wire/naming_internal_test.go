package wire

import (
	"encoding/json"
	"os"
	"testing"
)

// TestCamelNameIsTheReferencePolicy is the conversion on its own, one row per
// rule, so that a failure names the rule rather than an endpoint.
func TestCamelNameIsTheReferencePolicy(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		// The two names behaviours 1.13 turns on.
		{name: "UICulture", want: "uiCulture"},
		{name: "Id", want: "id"},

		// The ordinary case: one capital, and nothing after it moves.
		{name: "ServerName", want: "serverName"},
		{name: "ProductName", want: "productName"},
		{name: "StartupWizardCompleted", want: "startupWizardCompleted"},

		// The name that makes the wrong rule look right.
		{name: "ETag", want: "eTag"},

		// A run that ends the name has no following word to belong to, so all
		// of it lowers.
		{name: "ID", want: "id"},
		{name: "HTTP", want: "http"},
		{name: "A", want: "a"},

		// A run of three before a word: two lower, the third starts the word.
		{name: "XMLHttpFactory", want: "xmlHttpFactory"},

		{name: "", want: ""},
		{name: "Id2", want: "id2"},

		// ⚠️ UNVERIFIED, and unreachable from any name this server sends: no
		// property name of the pinned document begins with a lower-case
		// character or contains a space. Both rows are the reference's policy
		// carried over rather than a measurement.
		{name: "aBC", want: "aBC"},
		{name: "AB C", want: "ab C"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := camelName(c.name); got != c.want {
				t.Errorf("camelName(%q) = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// TestTheTwoRulesDisagreeOnExactlyOneName is behaviours 1.13's measurement,
// re-run here as a check.
//
// The section says that over the property names of the pinned document, the
// reference's policy and "lower the first letter" disagree exactly once, on
// `UICulture`. That number is what makes the wrong rule so easy to ship: a spot
// check almost certainly lands on one of the other names, where both rules
// agree.
//
// So the check is the disagreement itself rather than a sample of conversions.
// It fails on a conversion that lowers only the first letter — the whole set
// goes empty — and it fails on a conversion that lowers a whole leading run,
// because `ETag` would then join the set.
func TestTheTwoRulesDisagreeOnExactlyOneName(t *testing.T) {
	const path = "../../docs/compatibility/property-names.json"

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	var document struct {
		Count int      `json:"count"`
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(document.Names) != document.Count {
		t.Fatalf("%s holds %d names and claims %d", path, len(document.Names), document.Count)
	}

	var disagreed []string
	for _, name := range document.Names {
		if camelName(name) != lowerFirstLetter(name) {
			disagreed = append(disagreed, name)
		}
	}

	if len(disagreed) != 1 || disagreed[0] != "UICulture" {
		t.Fatalf("the two rules disagree on %v over %d names, want exactly [UICulture]",
			disagreed, len(document.Names))
	}
	if got, want := camelName("UICulture"), "uiCulture"; got != want {
		t.Errorf("camelName(%q) = %q, want %q", "UICulture", got, want)
	}
}

// lowerFirstLetter is the rule the reference does not use, written out so that
// the difference between the two can be measured rather than described.
func lowerFirstLetter(name string) string {
	chars := []rune(name)
	if len(chars) == 0 {
		return name
	}
	if chars[0] >= 'A' && chars[0] <= 'Z' {
		chars[0] += 'a' - 'A'
	}
	return string(chars)
}

// TestMarshalRefusesANamingPolicyItDoesNotHave keeps T6's check alive after T7
// moved the caller's choice up to a Profile.
//
// Write no longer takes a Naming, so nothing outside this package can reach the
// arm any more. That is not a reason to drop it: marshal is where the two
// policies are dispatched, and a third one arriving there — a policy added and
// not wired into profileAnswers, say — must be a refusal rather than a silent
// PascalCase body (behaviours 1.13).
func TestMarshalRefusesANamingPolicyItDoesNotHave(t *testing.T) {
	body, err := marshal(struct{ Id string }{Id: "3f9c"}, Naming(7))

	if err == nil {
		t.Fatalf("marshal(..., Naming(7)) = %s, want an error", body)
	}
	if body != nil {
		t.Errorf("marshal(..., Naming(7)) body = %s, want nothing", body)
	}
}
