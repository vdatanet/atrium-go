// Command build_library_fixture writes the scanning fixture tree of
// conformance §L2 into a directory.
//
// It exists so that `conformance/` can have the tree without importing
// anything of ours (architecture §3, 003 plan §3). That package speaks HTTP and
// its import boundary is enforced over `go list -deps` by
// tools/check_conformance_imports — so it runs
//
//	go run ./tools/build_library_fixture -into <dir>
//
// as a subprocess, exactly as it already runs `go build` for the server binary.
// A subprocess is not an import, and the check reads a dependency graph rather
// than a process tree.
//
// There is one declaration and two ways to reach it: this program and
// internal/libraryfixture. Two readers of one document cannot agree by
// construction, which is the same trade 001 makes for surface.yaml.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
)

func main() {
	into := flag.String("into", "", "the directory to write the fixture tree into; it must not already hold one")
	flag.Parse()

	if err := run(*into); err != nil {
		fmt.Fprintln(os.Stderr, "build_library_fixture:", err)
		os.Exit(1)
	}
}

func run(into string) error {
	if into == "" {
		return fmt.Errorf("-into is required: the tree is generated per run and there is no default directory to put one in")
	}
	if err := libraryfixture.Build(into); err != nil {
		return err
	}

	fmt.Printf("wrote the fixture tree into %s\n", into)
	for _, library := range libraryfixture.Libraries() {
		fmt.Printf("  %s\n", library)
	}
	return nil
}
