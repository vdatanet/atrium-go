package build

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	cases := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name:    "a stamped version is reported verbatim",
			stamped: "10.11.11-atrium.1",
			want:    "10.11.11-atrium.1",
		},
		{
			name:    "surrounding whitespace is trimmed, because a linker flag collects it",
			stamped: "  1.2.3\n",
			want:    "1.2.3",
		},
		{
			name:    "a stamp that cannot go in a header is no stamp at all",
			stamped: "1.2.3 (built by hand)",
			want:    developmentVersion,
		},
		{
			name: "nothing stamped and no build information",
			want: developmentVersion,
		},
		{
			name: "an unstamped build names the revision it was built from",
			info: buildInfo(map[string]string{"vcs.revision": "0123456789abcdef0123456789abcdef01234567"}),
			want: developmentVersion + "+0123456789ab",
		},
		{
			name: "a dirty working tree says so",
			info: buildInfo(map[string]string{
				"vcs.revision": "0123456789abcdef",
				"vcs.modified": "true",
			}),
			want: developmentVersion + "+0123456789ab.dirty",
		},
		{
			name:    "the stamp wins over the revision",
			stamped: "1.2.3",
			info:    buildInfo(map[string]string{"vcs.revision": "0123456789abcdef"}),
			want:    "1.2.3",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := versionFrom(c.stamped, c.info); got != c.want {
				t.Errorf("versionFrom(%q, %v) = %q, want %q", c.stamped, c.info, got, c.want)
			}
		})
	}
}

// TestVersionIsAHeaderToken guards the reason this package exists: the value
// goes into "Server: Atrium/<version>", so a version carrying a space or a
// quote would produce a header no client can parse.
func TestVersionIsAHeaderToken(t *testing.T) {
	got := Version()
	if got == "" {
		t.Fatal("Version() is empty; a binary that cannot state its version cannot be measured")
	}
	if tokenOrEmpty(got) != got {
		t.Errorf("Version() = %q, which is not an HTTP token", got)
	}
	if strings.ContainsAny(got, " \t/") {
		t.Errorf("Version() = %q, which cannot follow the slash in a Server header", got)
	}
}

func buildInfo(settings map[string]string) *debug.BuildInfo {
	info := &debug.BuildInfo{}
	for key, value := range settings {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: key, Value: value})
	}
	return info
}
