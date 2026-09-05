package app

// The five verbs that configure a library, which 003 plan §6.7 declares
// alongside the `scan` that shipped at T13.
//
// # These commands are the only interface an operator has, so their output is a
// # contract of a sort
//
// 003 registers no route (plan §3: *"internal/httpapi is untouched, and that is
// the sentence a reader should check first"*), so there is no request that
// declares a library and none that lists one. What is here is what an operator
// has, and `conformance/` too — that package may import nothing of ours, so
// running these commands is the only way it can put this feature's state into an
// installation (plan §6.7, §8.1).
//
// The consequence, stated rather than discovered: `list` and `scan` report
// through `--format json` as well as through a table, and **it is the document a
// test parses**. A test that parsed the human table would start constraining
// prose, and the prose is the half that is meant to change when an operator
// finds it unreadable.
//
// # Editing a library and recreating one are visibly different acts
//
// 003 §3.6 makes a library's collection type, its case sensitivity and its own
// identity frozen at creation, and that is what `rename` and `roots` being
// separate verbs is for: they are free, and every identifier under the library
// survives them. `remove` followed by `add` is **not** the same library — the
// identity is allocated afresh and every item under it derives a new identifier
// — and one `set` verb that quietly recreated would make that an accident
// instead of a decision.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
)

// --- add ----------------------------------------------------------------------

// addLibraryOptions is everything `library add` was told.
type addLibraryOptions struct {
	dataDirectory  string
	name           string
	collectionType string
	caseSensitive  bool
	roots          repeatedFlag
}

// newLibraryAddFlags builds `library add`'s flag set, unparsed.
//
// It returns the set unparsed for `newUserAddFlags`' reason: 002 T7's assertion
// that no flag carries a password is over the *flags themselves* rather than
// over the source, and the same shape is what lets this feature assert that no
// verb but this one can write a frozen column.
//
// **This is the only flag set in the program that declares --type or
// --case-sensitive.** See the constant block in library.go.
func newLibraryAddFlags(output io.Writer) (*flag.FlagSet, *addLibraryOptions) {
	options := &addLibraryOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryAdd, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "",
		"what to call the library; required, and two names differing only in case are one name")
	fs.StringVar(&options.collectionType, flagCollectionType, "",
		"which resolution rules apply: "+strings.Join(collectionTypeNames(), ", ")+
			"; required, and settled for the life of the library (003 §3.6)")
	fs.BoolVar(&options.caseSensitive, flagCaseSensitive, false,
		"treat two paths differing only in capitalisation as two items; "+
			"settled for the life of the library (003 §3.6)")
	fs.Var(&options.roots, flagRoot,
		"a directory to scan; required, repeatable, and kept in the order given")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryAdd+
			" — declare a library over one or more directories.\n\n"+
			"The collection type and --"+flagCaseSensitive+" are settled here and cannot be\n"+
			"changed afterwards: both decide every identifier under the library, and nothing\n"+
			"stores the old ones to undo with. Renaming a library and moving its roots are\n"+
			"free; removing one and declaring it again is not (003 §3.6).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runLibraryAdd declares one library.
func runLibraryAdd(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryAddFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	name, err := checkLibraryName(libraryAdd, options.name)
	if err != nil {
		return err
	}
	// The refusal that lists the three, and the list comes from the domain
	// rather than from a literal here: a fourth collection type would appear in
	// this message without anybody remembering to add it.
	collectionType, known := library.ParseCollectionType(options.collectionType)
	if !known {
		return fmt.Errorf("%s %s: --%s %q is not a collection type: it is one of %s",
			LibraryCommand, libraryAdd, flagCollectionType, options.collectionType,
			strings.Join(collectionTypeNames(), ", "))
	}
	roots, err := checkRoots(libraryAdd, options.roots)
	if err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	// Looked up first for the message, exactly as `user add` does. The unique
	// index on name_folded is what actually forbids the second row, and this is
	// the difference between a sentence naming the library that holds the name
	// and a constraint violation.
	folded := library.FoldName(name)
	if existing, found, err := store.LibraryByFoldedName(ctx, folded); err != nil {
		return err
	} else if found {
		return fmt.Errorf("%s %s: the name %q folds to %q, which the library %q (%s) already holds",
			LibraryCommand, libraryAdd, name, folded, existing.Name, existing.ID)
	}

	// Allocated, never derived from the name or the roots (003 §3.6). It is
	// what makes this verb, run after `remove` with the same arguments, a
	// *different* library — see library.NewID, which carries the argument.
	id, err := library.NewID()
	if err != nil {
		return err
	}

	declared := ports.Library{
		ID:             id,
		Name:           name,
		NameFolded:     folded,
		CollectionType: string(collectionType),
		CaseSensitive:  options.caseSensitive,
		Roots:          roots,
		CreatedAt:      SystemClock().Now(),
	}
	if err := store.CreateLibrary(ctx, declared); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", declared.ID, declared.Name)
	return nil
}

// --- list ---------------------------------------------------------------------

// listLibraryOptions is everything `library list` was told.
type listLibraryOptions struct {
	dataDirectory string
	format        string
}

func newLibraryListFlags(output io.Writer) (*flag.FlagSet, *listLibraryOptions) {
	options := &listLibraryOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryList, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.format, flagFormat, formatTable,
		"how to write the list: "+formatTable+" or "+formatJSON)

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryList+
			" — print every library this installation holds.\n\n"+
			"The roots are printed in the order they were configured, which is the order\n"+
			"an item's recorded root ordinal indexes.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// libraryReport is one library as `list` reports it.
//
// The roots are a list rather than a joined string because their **order** is
// what a reader has to be able to check: an item's path is relative to one of
// them and the recorded ordinal indexes this list.
type libraryReport struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	CollectionType string   `json:"collectionType"`
	CaseSensitive  bool     `json:"caseSensitive"`
	Roots          []string `json:"roots"`
}

// runLibraryList prints what libraries this installation holds.
func runLibraryList(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryListFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	if err := checkFormat(libraryList, options.format); err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	libraries, err := store.Libraries(ctx)
	if err != nil {
		return err
	}

	reports := make([]libraryReport, 0, len(libraries))
	for _, lib := range libraries {
		reports = append(reports, libraryReport{
			ID:             lib.ID,
			Name:           lib.Name,
			CollectionType: lib.CollectionType,
			CaseSensitive:  lib.CaseSensitive,
			// An installation with a library and no roots is a state the
			// store can hold, and `[]` says so where `null` would make a
			// parser decide what an absent list means.
			Roots: nonEmptyList(lib.Roots),
		})
	}

	if options.format == formatJSON {
		// A named list rather than a bare array, which is the shape `scan`
		// already reports in: a later verb can then report something beside
		// the libraries without changing what is already parsed.
		document, err := json.Marshal(struct {
			Libraries []libraryReport `json:"libraries"`
		}{Libraries: reports})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%s\n", document)
		return err
	}

	table := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tNAME\tTYPE\tCASE-SENSITIVE\tROOTS")
	for _, report := range reports {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			report.ID, report.Name, report.CollectionType,
			yesNo(report.CaseSensitive), strings.Join(report.Roots, ", "))
	}
	return table.Flush()
}

// --- rename -------------------------------------------------------------------

// renameLibraryOptions is everything `library rename` was told.
type renameLibraryOptions struct {
	dataDirectory string
	name          string
	to            string
}

func newLibraryRenameFlags(output io.Writer) (*flag.FlagSet, *renameLibraryOptions) {
	options := &renameLibraryOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryRename, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "", "the library to rename; required")
	fs.StringVar(&options.to, flagTo, "", "what to call it instead; required")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryRename+
			" — change what a library is called.\n\n"+
			"Free: a library's name is not an input to any identifier, so nothing under it\n"+
			"changes (003 §3.6).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runLibraryRename renames one library.
func runLibraryRename(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryRenameFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	wanted, err := checkLibraryName(libraryRename, options.name)
	if err != nil {
		return err
	}
	target, err := checkNewLibraryName(libraryRename, options.to)
	if err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	lib, err := libraryNamed(ctx, store, libraryRename, wanted)
	if err != nil {
		return err
	}

	// The fold is the domain's, and the same one the unique index is enforcing.
	// A library renamed to a different capitalisation of its own name folds to
	// what it already holds, and that is a rename rather than a collision —
	// which is why the identifier is compared and not only the fold.
	folded := library.FoldName(target)
	if existing, found, err := store.LibraryByFoldedName(ctx, folded); err != nil {
		return err
	} else if found && existing.ID != lib.ID {
		return fmt.Errorf("%s %s: the name %q folds to %q, which the library %q (%s) already holds",
			LibraryCommand, libraryRename, target, folded, existing.Name, existing.ID)
	}

	if err := store.RenameLibrary(ctx, lib.ID, target, folded); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", lib.ID, target)
	return nil
}

// --- roots --------------------------------------------------------------------

// rootsLibraryOptions is everything `library roots` was told.
type rootsLibraryOptions struct {
	dataDirectory string
	name          string
	roots         repeatedFlag
}

func newLibraryRootsFlags(output io.Writer) (*flag.FlagSet, *rootsLibraryOptions) {
	options := &rootsLibraryOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryRoots, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "", "the library whose roots to replace; required")
	fs.Var(&options.roots, flagRoot,
		"a directory to scan; required, repeatable, and it replaces the roots configured now")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryRoots+
			" — replace the directories a library is scanned from.\n\n"+
			"Free: an item's identity is derived from its path relative to its root and\n"+
			"never from the root itself, so a library that moved between mounts keeps every\n"+
			"identifier under it (003 §3.6, AC-10).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runLibraryRoots replaces one library's roots.
func runLibraryRoots(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryRootsFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	wanted, err := checkLibraryName(libraryRoots, options.name)
	if err != nil {
		return err
	}
	roots, err := checkRoots(libraryRoots, options.roots)
	if err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	lib, err := libraryNamed(ctx, store, libraryRoots, wanted)
	if err != nil {
		return err
	}
	if err := store.ReplaceRoots(ctx, lib.ID, roots); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", lib.ID, strings.Join(roots, "\t"))
	return nil
}

// --- remove -------------------------------------------------------------------

// removeLibraryOptions is everything `library remove` was told.
type removeLibraryOptions struct {
	dataDirectory string
	name          string
}

func newLibraryRemoveFlags(output io.Writer) (*flag.FlagSet, *removeLibraryOptions) {
	options := &removeLibraryOptions{}

	fs := flag.NewFlagSet("atrium "+LibraryCommand+" "+libraryRemove, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "", "the library to remove; required")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+LibraryCommand+" "+libraryRemove+
			" — forget a library and everything scanned under it.\n\n"+
			"Not free, and not the opposite of "+libraryAdd+": a library's identity is\n"+
			"allocated rather than derived, so declaring one again with the same name and\n"+
			"the same roots is a **different** library and every item under it gets a new\n"+
			"identifier — which is a favourite, a resume position and a client's cache\n"+
			"(003 §3.6).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runLibraryRemove forgets one library.
//
// **The items go first, and that is a decision this verb had to take.** Nothing
// in the schema removes them: `items.library_id` is a string and deliberately
// not a foreign key, because the derived half may not reference the precious one
// (derived/library.sql's header), so `RemoveLibrary` on its own leaves every row
// this library ever scanned behind — invisible, because every reader reaches
// items through a library that no longer exists, and permanent, because a
// library identifier is allocated afresh and never reused.
//
// What survives is the library's `scan_state` row, and that is stated rather
// than hidden: no port method deletes one, adding one is an amendment to
// `ports.ItemStore`, and the row can never be read again because nothing will
// ever hold that library identifier. It is a debt in the handover rather than a
// method invented here.
func runLibraryRemove(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs, options := newLibraryRemoveFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	wanted, err := checkLibraryName(libraryRemove, options.name)
	if err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	lib, err := libraryNamed(ctx, store, libraryRemove, wanted)
	if err != nil {
		return err
	}

	items, err := store.ItemsForLibrary(ctx, lib.ID)
	if err != nil {
		return err
	}
	identifiers := make([]string, 0, len(items))
	for _, item := range items {
		identifiers = append(identifiers, item.ID)
	}
	if err := store.RemoveItems(ctx, identifiers); err != nil {
		return err
	}
	if err := store.RemoveLibrary(ctx, lib.ID); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", lib.ID, lib.Name)
	return nil
}

// --- What the verbs share -----------------------------------------------------

// repeatedFlag is a flag that may be given more than once, keeping the order.
//
// `--root` is the only one, and the order is the whole reason it is not a
// comma-separated string: `ports.ScannedItem.RootOrdinal` indexes this list, so
// two roots that swapped places would leave every item's recorded ordinal
// naming the root its path is *not* relative to. A separator would also make a
// directory whose name contains that separator unconfigurable.
type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, ", ") }

func (r *repeatedFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

// collectionTypeNames is the three, spelled by the domain.
//
// Every message that lists them reads this rather than a literal, so a fourth
// collection type is listed by the refusal without anybody remembering to
// change it — which is the same cross-check `internal/store/sqlite`'s
// `TestAFourthCollectionTypeIsRefused` makes against the schema's CHECK.
func collectionTypeNames() []string {
	all := library.AllCollectionTypes()
	names := make([]string, 0, len(all))
	for _, collectionType := range all {
		names = append(names, string(collectionType))
	}
	return names
}

// libraryNamed finds the library an operator named, or refuses.
//
// **A name no library has is a refusal and never an empty run**, for every verb.
// An operator who mistyped and was told nothing happened would read it as
// *"there was nothing to do"*, which is what a successful run of the same
// command says.
//
// The match is on the domain's fold and not on bytes, because 003 §3.6 makes two
// library names differing only in case one name and the store's unique index is
// enforcing that fold.
func libraryNamed(ctx context.Context, store ports.LibraryStore, verb, name string) (ports.Library, error) {
	folded := library.FoldName(name)
	lib, found, err := store.LibraryByFoldedName(ctx, folded)
	if err != nil {
		return ports.Library{}, err
	}
	if !found {
		return ports.Library{}, fmt.Errorf("%s %s: there is no library named %q",
			LibraryCommand, verb, name)
	}
	return lib, nil
}

// checkLibraryName refuses an empty library name.
//
// It refuses only what makes the rest impossible, which is `checkName`'s rule
// for an account one command along: what else a library may be called is
// measured nowhere in this repository — ⚠️ UNVERIFIED — and a restriction
// invented here is one no operator could have predicted.
func checkLibraryName(verb, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%s %s: a library name is required: pass --%s",
			LibraryCommand, verb, flagName)
	}
	return trimmed, nil
}

// checkNewLibraryName is checkLibraryName for `rename`'s --to, which names the
// flag it is about rather than the one beside it.
func checkNewLibraryName(verb, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%s %s: a new name is required: pass --%s",
			LibraryCommand, verb, flagTo)
	}
	return trimmed, nil
}

// checkRoots turns what was typed into the absolute directories a scan walks,
// or refuses with the path.
//
// 003 plan §7's first row: a root that does not exist, or is not a directory, is
// refused **here** as well as at every scan. Refusing it at declaration is what
// turns a mistyped mount into a message an operator reads while they are still
// looking at the command, rather than into a library that refuses every scan
// afterwards.
//
// The path is made absolute for the same reason `--data-dir` is: this
// installation is read by a server started from somewhere else entirely, and a
// relative root would name a different directory for every process that opened
// it. Symbolic links are **not** resolved — an operator who configured a link
// configured the link, and resolving it would make the recorded root a path they
// never typed.
func checkRoots(verb string, given []string) ([]string, error) {
	if len(given) == 0 {
		return nil, fmt.Errorf("%s %s: at least one root is required: pass --%s",
			LibraryCommand, verb, flagRoot)
	}

	roots := make([]string, 0, len(given))
	for _, root := range given {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			return nil, fmt.Errorf("%s %s: --%s was given an empty path",
				LibraryCommand, verb, flagRoot)
		}
		absolute, err := filepath.Abs(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%s %s: --%s %q: %w", LibraryCommand, verb, flagRoot, root, err)
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, fmt.Errorf("%s %s: --%s %q: %w", LibraryCommand, verb, flagRoot, absolute, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s %s: --%s %q is not a directory",
				LibraryCommand, verb, flagRoot, absolute)
		}
		roots = append(roots, absolute)
	}
	return roots, nil
}

// checkFormat refuses a spelling of --format that is neither of the two.
func checkFormat(verb, format string) error {
	if format != formatTable && format != formatJSON {
		return fmt.Errorf("%s %s: --%s %q is not %s or %s",
			LibraryCommand, verb, flagFormat, format, formatTable, formatJSON)
	}
	return nil
}

// nonEmptyList answers `[]` where a nil slice would marshal as `null`.
//
// The same rule `scan`'s summary already keeps: a parser that has to decide what
// `null` means for a list is a parser two readers will decide differently.
func nonEmptyList(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
