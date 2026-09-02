// Command check_conformance_imports asserts that nothing under conformance/
// depends on this module's internal packages.
//
// architecture §3 states the rule and why it needs a check rather than a
// promise: conformance/ speaks HTTP and imports nothing of ours, but Go will
// not enforce that — internal/ restricts imports across module paths, not
// within a module, so the compiler happily lets a conformance test reach inside
// and assert on a value the wire never carried.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const (
	module      = "github.com/vdatanet/atrium-go"
	forbidden   = module + "/internal/"
	conformance = "conformance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "check_conformance_imports:", err)
		os.Exit(1)
	}
}

func run() error {
	// The directory does not exist until the first feature ships a conformance
	// test. That is not a failure; it is the state before 001's T19.
	if info, err := os.Stat(conformance); err != nil || !info.IsDir() {
		fmt.Println("no conformance/ directory yet — nothing to check")
		return nil
	}

	offenders, err := offendingPackages()
	if err != nil {
		return err
	}
	if len(offenders) == 0 {
		fmt.Println("conformance/ imports nothing under internal/")
		return nil
	}

	sort.Strings(offenders)
	var b strings.Builder
	b.WriteString("conformance/ reaches into internal/, which architecture §3 forbids:\n")
	for _, line := range offenders {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	b.WriteString("A test that can reach inside can assert on a value the wire never carried.")
	return fmt.Errorf("%s", b.String())
}

// offendingPackages returns "<package> imports <internal package>" for every
// direct import of an internal package from a conformance package, counting
// test imports, which is where the temptation actually lives.
func offendingPackages() ([]string, error) {
	const format = `{{$p := .ImportPath}}` +
		`{{range .Imports}}{{$p}} imports {{.}}` + "\n" + `{{end}}` +
		`{{range .TestImports}}{{$p}} imports {{.}}` + "\n" + `{{end}}` +
		`{{range .XTestImports}}{{$p}} imports {{.}}` + "\n" + `{{end}}`

	cmd := exec.Command("go", "list", "-f", format, "./"+conformance+"/...")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	var offenders []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		_, imported, ok := strings.Cut(line, " imports ")
		if ok && strings.HasPrefix(imported, forbidden) {
			offenders = append(offenders, line)
		}
	}
	return offenders, nil
}
