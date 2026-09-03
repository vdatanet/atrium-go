package system

// What this installation tells a client it is.
//
// Both constants are wire values rather than facts about this program, and
// that is the whole of behaviours 4.1: this server identifies as Jellyfin on
// the two fields clients parse, and as Atrium everywhere a human looks — the
// Server header, the ServerName field, the logs and the project page.
// reference-target.md 4 is where the argument is settled, and it is settled
// there rather than here because Principle I (zero delta) and Principle X
// (honest about lineage) pull against each other on exactly these two strings.
//
// They live in this package because they are what this installation *calls
// itself*, which is this package's subject, and because three responses need
// the same two answers: /System/Info/Public and /System/Info report both, and
// /System/Ping is the product name on its own (spec 3.3). A literal repeated
// at three handlers is three places to be wrong.
const (
	// ProductName is what /System/Info/Public, /System/Info and both spellings
	// of /System/Ping report, and it is exactly "Jellyfin Server".
	//
	// It is the documented discriminator a multi-server client reads to decide
	// whether it is talking to Emby or to Jellyfin
	// [probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]. A
	// client that reads "Atrium" here takes an unknown-server path and never
	// reaches the rest of the API, so Principle I is broken at the very first
	// request. spec 3.1 spells the requirement "exactly Jellyfin Server".
	ProductName = "Jellyfin Server"

	// ReportedVersion is what the Version field carries: the pinned reference
	// version, not this binary's own.
	//
	// internal/build is the other version in this program and the two are
	// deliberately different things — Version tells a client which API dialect
	// it is speaking, and clients gate capabilities on it, while
	// Server: Atrium/<version> tells a person which server answered. A
	// differential run depends on the second being unlike the reference's and
	// on this one being exactly like it.
	//
	// The value is the project's pin, from reference-target.md's table
	// ("Version Atrium reports: 10.11.11", settled 2026-09-01). It is
	// deliberately *not* read from surface.Table.Reference(): that field
	// records which OpenAPI document the route table was generated against,
	// and its own documentation says it is not the project's pin. Two pins
	// that happen to agree today are not one pin.
	ReportedVersion = "10.11.11"

	// OperatingSystem is what the field of that name carries: nothing.
	//
	// The reference marks the property obsolete and never assigns it, so it
	// serialises as its default empty string
	// [source: MediaBrowser.Model/System/PublicSystemInfo.cs:37-38 @ v10.11.11]
	// — and an empty string is not a null, so behaviours 1.7's omit-when-null
	// rule does not reach it and the field is present and empty. spec 3.1
	// spells it "always the empty string".
	//
	// It is a named constant rather than "" at the handler because the empty
	// string is the one value a reader assumes is an oversight.
	OperatingSystem = ""
)
