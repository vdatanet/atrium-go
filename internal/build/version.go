// Package build carries the version this binary was stamped with.
//
// It is a leaf on purpose: the edge needs the version for the Server header
// (behaviours 4.1) and the entry layer needs it for a startup line, so it may
// not live in either.
//
// architecture 5 says why a version is not cosmetic here: a differential run
// refuses to start unless the two servers differ on the Server header, because
// ProductName is "Jellyfin Server" on both. "Server: Atrium/<version>" is the
// one thing that tells them apart, and "a binary that cannot state its version
// cannot be measured".
package build

import (
	"runtime/debug"
	"strings"
)

// version is set at link time:
//
//	go build -ldflags "-X github.com/vdatanet/atrium-go/internal/build.version=1.2.3" ./cmd/atrium
//
// It is deliberately not a constant edited by hand: architecture 4 puts the
// Server header's version in "a build-stamped version" and never in "a constant
// edited by hand", because a stale constant misidentifies a measured run.
var version string

// developmentVersion is what an unstamped build reports. It sorts below any
// real release and says plainly that nothing stamped it.
const developmentVersion = "0.0.0-dev"

// revisionLength is how much of a commit hash the fallback keeps: enough to
// find the commit, short enough for a header.
const revisionLength = 12

// Version returns the version this binary reports. It is never empty, never
// contains a space, and is always a valid HTTP token, because it is used as the
// product version of a Server header.
func Version() string {
	info, _ := debug.ReadBuildInfo()
	return versionFrom(version, info)
}

// versionFrom is Version's decision, separated from the two things it cannot
// control: the link-time stamp and the toolchain's build information.
func versionFrom(stamped string, info *debug.BuildInfo) string {
	if v := tokenOrEmpty(stamped); v != "" {
		return v
	}

	revision, modified := vcsStamp(info)
	if revision == "" {
		return developmentVersion
	}
	if len(revision) > revisionLength {
		revision = revision[:revisionLength]
	}
	v := developmentVersion + "+" + revision
	if modified {
		v += ".dirty"
	}
	return v
}

// vcsStamp reads the revision the toolchain records for a build made inside a
// repository, so that a plain `go build` still produces a binary a report can
// be traced back to.
func vcsStamp(info *debug.BuildInfo) (revision string, modified bool) {
	if info == nil {
		return "", false
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = tokenOrEmpty(setting.Value)
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return revision, modified
}

// tokenOrEmpty returns s trimmed if every remaining byte may appear in an HTTP
// token, and the empty string otherwise. A value that cannot go in a header is
// treated as no value at all rather than sanitised into something that no
// longer names the build it came from.
func tokenOrEmpty(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if !isTokenByte(s[i]) {
			return ""
		}
	}
	return s
}

// isTokenByte reports whether c is one of RFC 9110's tchar.
func isTokenByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}
