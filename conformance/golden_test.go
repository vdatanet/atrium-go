package conformance_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updateGolden rewrites the golden files from the responses this run received.
//
// architecture 8: goldens are reviewed, never blindly regenerated, and a diff
// in one is a contract change read like one. So this flag does two things and
// the second is the important one — it writes the file, and then it **fails the
// test**. No single run can both rewrite a golden and report green, which means
// the new bytes have to be looked at and the suite re-run without the flag
// before anything is called passing.
//
// That is the difference between an update flag and a "make the test pass"
// button. The button is what turns a golden into a record of whatever the code
// last did, which is a test that cannot fail.
var updateGolden = flag.Bool("update-golden", false,
	"rewrite the golden files from this run's responses, then fail so the diff is reviewed")

// goldenDirectory is where the recorded responses live.
const goldenDirectory = "testdata/golden"

// assertGolden compares one response body against its recorded bytes.
//
// The comparison is on bytes and nothing else. A golden compared after parsing
// would agree about a body whose property names had changed case, whose numbers
// had become strings, or whose keys had been reordered — and all three are
// differences a client sees (Principle VIII, conformance L3).
//
// The recorded file holds exactly what the wire carried, with no trailing
// newline, because internal/wire sends none. An editor that adds one to the
// file is a failing test rather than a silently forgiving one, which is the
// right way round for a file whose whole job is to be exact.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join(goldenDirectory, name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("making the golden directory: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		t.Fatalf("%s was rewritten from this run.\n"+
			"A golden is a record of what a client receives, so read the diff as a contract change, "+
			"then re-run without -update-golden.\n%s", path, got)
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v\n"+
			"If this response is new, create the file with -update-golden and review what it recorded.",
			path, err)
	}

	if bytes.Equal(got, want) {
		return
	}

	t.Errorf("the response does not match %s.\n got %s\nwant %s\n%s",
		path, got, want, firstDifference(got, want))
}

// statedMember is one member of a golden body whose value derives from the run
// and which the golden therefore states rather than records.
//
// name is the placeholder's name inside the file — `{{AccessToken}}` — and
// value is what this run's response carried at that position.
type statedMember struct {
	name  string
	value string
}

// assertGoldenWithStatedMembers compares a response against a golden that
// states the members deriving from the run, at the positions they occupy.
//
// # Why a golden needs this at all
//
// 001 T16's rule: a byte-compared golden needs the response to stop deriving
// from the run, and the honest fix is to state the input rather than blank the
// output. Every derived member of 001's bodies could be stated through an
// input — the identity by writing the file, the address by sending a Host
// header — so 001 needed no placeholder.
//
// 002's authentication result carries three that cannot be. The access token
// is read from the system's randomness and the two dates from the wall clock,
// and the binary offers an operator no way to state either: there is no flag,
// no environment variable and no request that fixes them. The choice is
// therefore between a golden that names them and no golden at all.
//
// # This is not "normalising away"
//
// Blanking would delete the member from the response and compare what is left,
// which agrees with a body that moved the member, renamed it, changed its JSON
// type or dropped the quotes around it. Here the golden holds the member's
// name, its position among the other bytes and its quoting, and the value that
// goes into it is one this run's caller has already asserted against a rule of
// its own — a shape, or a window the request happened inside. A member the
// caller does not state is compared as bytes like everything else.
//
// A stated member whose placeholder is not in the file is a fatal error rather
// than a no-op, because a substitution that matches nothing is a caller
// asserting something and a file not being asked anything.
func assertGoldenWithStatedMembers(t *testing.T, name string, got []byte, stated []statedMember) {
	t.Helper()

	path := filepath.Join(goldenDirectory, name)

	if *updateGolden {
		// The file records the *template*, so the run's own token and dates
		// never reach it. First occurrence only, and in the order the caller
		// gave: two members can hold the same string — two clock reads a
		// microsecond apart usually do not, but nothing stops them — and
		// replacing every occurrence would give both slots one name.
		template := string(got)
		for _, member := range stated {
			if !strings.Contains(template, member.value) {
				t.Fatalf("the response does not carry %s's stated value %q, so %s cannot be recorded:\n%s",
					member.name, member.value, path, got)
			}
			template = strings.Replace(template, member.value, placeholder(member.name), 1)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("making the golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		t.Fatalf("%s was rewritten from this run.\n"+
			"A golden is a record of what a client receives, so read the diff as a contract change, "+
			"then re-run without -update-golden.\n%s", path, template)
	}

	template, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v\n"+
			"If this response is new, create the file with -update-golden and review what it recorded.",
			path, err)
	}

	want := string(template)
	for _, member := range stated {
		slot := placeholder(member.name)
		if !strings.Contains(want, slot) {
			t.Fatalf("%s does not state %s: there is no %s in it, so stating a value for it asserts nothing",
				path, member.name, slot)
		}
		want = strings.ReplaceAll(want, slot, member.value)
	}

	if bytes.Equal(got, []byte(want)) {
		return
	}

	t.Errorf("the response does not match %s with its stated members filled in.\n got %s\nwant %s\n%s",
		path, got, want, firstDifference(got, []byte(want)))
}

// placeholder is how a stated member is spelled inside a golden file.
func placeholder(name string) string {
	return "{{" + name + "}}"
}

// firstDifference names the byte the two bodies first disagree on, because a
// two-hundred-byte JSON object printed twice is not a diff a reader can see.
func firstDifference(got, want []byte) string {
	limit := min(len(got), len(want))
	for i := range limit {
		if got[i] != want[i] {
			return "They first differ at byte " + itoa(i) + ": got " +
				quoteByte(got[i]) + ", want " + quoteByte(want[i]) + "."
		}
	}
	switch {
	case len(got) > len(want):
		return "They agree for " + itoa(limit) + " bytes and the response then carries " +
			itoa(len(got)-limit) + " more: " + string(got[limit:])
	case len(want) > len(got):
		return "They agree for " + itoa(limit) + " bytes and the response then stops; the golden carries " +
			itoa(len(want)-limit) + " more: " + string(want[limit:])
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func quoteByte(b byte) string {
	if b >= 0x20 && b < 0x7f {
		return "'" + string(rune(b)) + "'"
	}
	const hex = "0123456789abcdef"
	return "0x" + string([]byte{hex[b>>4], hex[b&0x0f]})
}
