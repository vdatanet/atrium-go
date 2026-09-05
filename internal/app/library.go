package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/scan"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
)

// LibraryCommand is the first argument that selects the library subcommands.
//
// It is a constant for [UserCommand]'s reason: the dispatch in cmd/atrium is
// one branch and nothing else (002 plan §3), and a word spelled in two places
// is a word that can be spelled two ways.
const LibraryCommand = "library"

// The six verbs of 003 plan §6.7.
//
// `scan` shipped first, at T13, because that task's criteria are asserted
// *through the subcommand* rather than through the function: AC-12 and AC-16
// are about what the store ends up holding after an operator ran a scan. The
// other five arrive here, with the arm on cmd/atrium's dispatch that makes any
// of them reachable from a shell.
const (
	libraryAdd    = "add"
	libraryList   = "list"
	libraryRename = "rename"
	libraryRoots  = "roots"
	libraryRemove = "remove"
	libraryScan   = "scan"
)

// libraryVerbSet is every verb and the flag set it parses, unparsed, in the
// order the usage text lists them.
//
// **It is one table read by the dispatch and by the tests**, and that is the
// whole reason it exists. 003 T14's definition of done asserts that no verb but
// `add` offers a way to write a library's collection type or its case
// sensitivity, *"over the parsed flag sets rather than by reading the source"* —
// so a verb whose flags this table did not carry would be a verb that assertion
// never looked at, and a sixth arm added to [RunLibrary] without a row here is
// caught by `TestEveryVerbTheDispatchAcceptsHasARowInTheFlagTable`.
//
// The options each constructor returns are dropped: a caller that wants to
// *run* a verb calls its constructor directly. What a caller of this wants is
// the shape of the flags, and nothing else.
func libraryVerbSet(output io.Writer) []struct {
	Verb  string
	Flags *flag.FlagSet
} {
	add, _ := newLibraryAddFlags(output)
	list, _ := newLibraryListFlags(output)
	rename, _ := newLibraryRenameFlags(output)
	roots, _ := newLibraryRootsFlags(output)
	remove, _ := newLibraryRemoveFlags(output)
	scan, _ := newLibraryScanFlags(output)
	return []struct {
		Verb  string
		Flags *flag.FlagSet
	}{
		{libraryAdd, add},
		{libraryList, list},
		{libraryRename, rename},
		{libraryRoots, roots},
		{libraryRemove, remove},
		{libraryScan, scan},
	}
}

// RunLibrary runs `atrium library <subcommand>`.
//
// It is in internal/app and not in cmd/atrium for the reason
// [RunUser] is: architecture §3 allows cmd no branch a test would want to
// reach, and every branch below is one.
//
// Nothing this command takes is a secret, so there is no standard-input rule to
// inherit from 002 (003 plan §6.7). stdout carries what an operator asked to
// see, and stderr carries the usage text and the log.
func RunLibrary(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if len(args) == 0 {
		writeLibraryUsage(stderr)
		return fmt.Errorf("%s: a subcommand is required", LibraryCommand)
	}

	switch args[0] {
	case libraryAdd:
		return runLibraryAdd(ctx, args[1:], getenv, stdout, stderr)
	case libraryList:
		return runLibraryList(ctx, args[1:], getenv, stdout, stderr)
	case libraryRename:
		return runLibraryRename(ctx, args[1:], getenv, stdout, stderr)
	case libraryRoots:
		return runLibraryRoots(ctx, args[1:], getenv, stdout, stderr)
	case libraryRemove:
		return runLibraryRemove(ctx, args[1:], getenv, stdout, stderr)
	case libraryScan:
		return runLibraryScan(ctx, args[1:], getenv, stdout, stderr)
	default:
		writeLibraryUsage(stderr)
		return fmt.Errorf("%s: %q is not a subcommand", LibraryCommand, args[0])
	}
}

func writeLibraryUsage(output io.Writer) {
	fmt.Fprint(output, "atrium "+LibraryCommand+" — configure and scan this installation's libraries.\n\n"+
		"Usage:\n"+
		"  atrium "+LibraryCommand+" "+libraryAdd+"    --"+flagDataDirectory+" <directory> --"+flagName+" <name> --"+
		flagCollectionType+" "+strings.Join(collectionTypeNames(), "|")+" --"+flagRoot+" <path> [--"+flagRoot+" <path> …] [--"+flagCaseSensitive+"]\n"+
		"  atrium "+LibraryCommand+" "+libraryList+"   --"+flagDataDirectory+" <directory> [--"+flagFormat+" "+formatTable+"|"+formatJSON+"]\n"+
		"  atrium "+LibraryCommand+" "+libraryRename+" --"+flagDataDirectory+" <directory> --"+flagName+" <name> --"+flagTo+" <name>\n"+
		"  atrium "+LibraryCommand+" "+libraryRoots+"  --"+flagDataDirectory+" <directory> --"+flagName+" <name> --"+flagRoot+" <path> [--"+flagRoot+" <path> …]\n"+
		"  atrium "+LibraryCommand+" "+libraryRemove+" --"+flagDataDirectory+" <directory> --"+flagName+" <name>\n"+
		"  atrium "+LibraryCommand+" "+libraryScan+"   --"+flagDataDirectory+" <directory> [flags]\n\n"+
		"A library's collection type and its case sensitivity are settled when it is\n"+
		"declared and there is no verb that changes either: they decide every identifier\n"+
		"under the library, and nothing stores the old ones to undo with (003 §3.6).\n")
}

// The flags the six verbs take. Every one of them is a name spelled once.
//
// `--type` and `--case-sensitive` are declared by exactly one verb, and the
// declaration is 003 §3.6's refusal rather than a convenience: *"an attempt to
// change it afterwards is refused, not accepted with a warning"*, and the way
// this design refuses it is that **there is nothing to type**. The assertion
// that no other verb offers either lives over these flag sets — see
// `TestNoVerbButAddCanWriteAFrozenColumn`, which is this refusal's operator-
// facing half; the store-facing half is `internal/ports`'
// `TestNoMethodOfTheLibraryStoreCanCarryAFrozenColumn`, and neither implies the
// other.
const (
	flagFull           = "full"
	flagAllowEmptyRoot = "allow-empty-root"
	flagFormat         = "format"
	flagCollectionType = "type"
	flagCaseSensitive  = "case-sensitive"
	flagRoot           = "root"
	flagTo             = "to"
)

// The two spellings of --format.
//
// `table` is what an operator reads and `json` is what a test parses, and the
// split is deliberate: 003 plan §6.7 says the commands' output is a contract of
// a sort, and *"parsing a human table in a test is how a test starts
// constraining prose"*.
const (
	formatTable = "table"
	formatJSON  = "json"
)

// scanOptions is everything `library scan` was told.
type scanOptions struct {
	dataDirectory string
	name          string
	logLevel      string
	format        string
	options       scan.Options
}

// newLibraryScanFlags builds `library scan`'s flag set, unparsed.
//
// It is a function of its own so that a test can assert over the flags
// themselves rather than over this file's text, which is the shape 002 T7
// established for the absence of `--password` and which 003 T14 needs for the
// absence of `--collection-type` on every verb but `add`.
func newLibraryScanFlags(output io.Writer) (*flag.FlagSet, *scanOptions) {
	options := &scanOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryScan, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "",
		"scan only the library with this name; every library is scanned when it is absent")
	fs.StringVar(&options.logLevel, flagLogLevel, levelName(DefaultLogLevel),
		"lowest level to log: "+strings.Join(levelNames(), ", ")+
			"; debug names every skipped path and its reason (env "+EnvLogLevel+")")
	fs.StringVar(&options.format, flagFormat, formatTable,
		"how to write the summary: "+formatTable+" or "+formatJSON)

	fs.BoolVar(&options.options.Full, flagFull, false,
		"re-examine every file instead of believing the size and time of change (003 §3.8)")
	fs.BoolVar(&options.options.AllowEmptyRoot, flagAllowEmptyRoot, false,
		"proceed over a root that reads as holding no candidate file, removing everything that was under it")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryScan+
			" — read every root of every library and record what is there.\n\n"+
			"A scan is incremental by default and changes nothing when a root cannot be read.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runLibraryScan scans one library, or every library.
func runLibraryScan(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryScanFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	fallBackToEnv(&options.logLevel, options.logLevel != levelName(DefaultLogLevel), getenv(EnvLogLevel))
	level, err := parseLevel(options.logLevel)
	if err != nil {
		return err
	}
	if err := checkFormat(libraryScan, options.format); err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	libraries, err := selectedLibraries(ctx, store, options.name)
	if err != nil {
		return err
	}

	logger := NewLogger(stderr, level)
	scanner, err := scan.New(scan.Config{
		Items:     store,
		Clock:     SystemClock(),
		ClaimedBy: ScannerName(),
		Logger:    logger,
	})
	if err != nil {
		return err
	}

	// Every selected library is scanned, and one that fails does not stop the
	// next: 003 plan §6.5 and §7 both scope a failed reading to *that library*,
	// and a loop that gave up on the first would let an unmounted share on one
	// library leave three others unscanned with nothing said about them.
	reports := make([]scanReport, 0, len(libraries))
	var failures []error
	for _, lib := range libraries {
		changes, err := scanner.Scan(ctx, lib, options.options)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		reports = append(reports, scanReport{ID: lib.ID, Name: lib.Name, Changes: changes})
	}

	if err := writeScanReports(stdout, options.format, reports); err != nil {
		return err
	}
	return errors.Join(failures...)
}

// selectedLibraries is every library, or the one an operator named.
//
// A name that matches nothing is an error rather than an empty run. An operator
// who mistyped a library's name and was told *"0 libraries scanned"* would read
// it as *"nothing changed"*, which is the same sentence a successful scan of an
// unchanged tree produces.
func selectedLibraries(ctx context.Context, store ports.LibraryStore, name string) ([]ports.Library, error) {
	if strings.TrimSpace(name) == "" {
		return store.Libraries(ctx)
	}

	lib, err := libraryNamed(ctx, store, libraryScan, name)
	if err != nil {
		return nil, err
	}
	return []ports.Library{lib}, nil
}

// scanReport is one library's line of the summary.
//
// [scan.Changes] is embedded rather than held in a field, so that its six
// properties sit at the top level of the JSON document beside the library's
// identity — one shape for the summary wherever it is read, and the same one
// the store holds on the library's scan state.
type scanReport struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	scan.Changes
}

// writeScanReports writes the summary 003 §3.8 requires.
func writeScanReports(stdout io.Writer, format string, reports []scanReport) error {
	if format == formatJSON {
		// A document with a named list rather than a bare array, so that a
		// later verb can report something beside the libraries without
		// changing the shape of what is already parsed.
		document, err := json.Marshal(struct {
			Libraries []scanReport `json:"libraries"`
		}{Libraries: reports})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%s\n", document)
		return err
	}

	// Tab-separated with a header, which is what an operator can read and what
	// a shell can cut — `user list`'s shape, one command along.
	table := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "LIBRARY\tADDED\tUPDATED\tREMOVED\tEXAMINED\tSKIPPED\tUNPLACEABLE")
	for _, report := range reports {
		fmt.Fprintf(table, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			report.Name, len(report.Added), len(report.Updated), len(report.Removed),
			report.Examined, report.Skipped, report.Unplaceable)
	}
	return table.Flush()
}

// ScannerName is how this process names itself in a library's scanning claim.
//
// It is the host and the process identifier, because 003 plan §7 asks for two
// messages that print it — *"already being scanned"* by whom, and whose claim
// was broken on age — and both are read by an operator deciding whether the
// other scanner is a server on this machine, a colleague's shell, or a process
// that is no longer running. A name that were only the host could not answer
// the third.
func ScannerName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		// A host with no name is not a reason to refuse to scan, and the
		// process identifier alone still tells an operator whether the
		// claimant is alive.
		host = "unknown-host"
	}
	return fmt.Sprintf("%s/%d", host, os.Getpid())
}

// The store satisfies both halves of what a scan needs, and this is where that
// is required rather than assumed.
var (
	_ ports.LibraryStore = (*sqlite.Store)(nil)
	_ ports.ItemStore    = (*sqlite.Store)(nil)
)
