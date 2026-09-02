package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// dataDirectoryMode is the permission a data directory is created with.
//
// Nothing but this process reads it. The precious half of the store holds
// credentials, tokens and sessions (ADR-0003), so the directory is the
// process's own and not a shared one — 0700 rather than the usual 0755.
const dataDirectoryMode fs.FileMode = 0o700

// EnsureDataDirectory creates the data directory if it is not there, and
// reports what is wrong when it cannot.
//
// It creates the final component only, and deliberately not its parents. The
// two failures this sits between are both real: a first start that has to be
// preceded by a mkdir is an awkward install and an awkward container image,
// and a server that invents every directory on a mistyped path answers a typo
// with an empty installation that looks like a fresh one — where an operator's
// data is sitting untouched under the path they meant. Creating one component
// under a parent that already exists makes a first start work and makes
// --data-dir /var/lbi/atrium fail while naming /var/lbi.
//
// This is the entry layer's and not the store's because both things in the
// data directory need it and neither owns it: the installation identity is
// read before the store is opened (plan 7 refuses to start on either), so a
// directory created by the store would be created after the first thing that
// needed it had already failed.
func EnsureDataDirectory(directory string) error {
	err := os.Mkdir(directory, dataDirectoryMode)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("--%s %s: %s does not exist (env %s)",
				flagDataDirectory, directory, filepath.Dir(directory), EnvDataDirectory)
		}
		return fmt.Errorf("--%s: %w", flagDataDirectory, err)
	}

	// Reached when the path was already there, which includes the case where
	// it is there as a file. Stat rather than trusting Mkdir's error, because
	// EEXIST says a name is taken and not what took it.
	info, statErr := os.Stat(directory)
	if statErr != nil {
		return fmt.Errorf("--%s %s: %w", flagDataDirectory, directory, statErr)
	}
	if !info.IsDir() {
		return fmt.Errorf("--%s %s: not a directory (env %s)",
			flagDataDirectory, directory, EnvDataDirectory)
	}
	return nil
}
