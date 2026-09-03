package system

import "path/filepath"

// Paths is where this installation keeps the things spec 3.2 reports.
//
// # Why this exists at all
//
// GET /System/Info reports seven paths, and spec 3.2 says they are "real paths
// from the running configuration". 001 gives an operator exactly one path to
// configure — the data directory — so the other six are derived from it here,
// in one place, rather than at the handler that happens to need them.
//
// # These directories are not created, and that is deliberate
//
// 001 creates the data directory and nothing inside it (plan 4, at T3). A
// server that made six empty directories on every start would be inventing a
// layout for features that do not exist yet, and the one thing an operator
// would notice is the noise. So this answers *where a thing will live*, which
// is what the field on the wire is: an address, not a promise that something
// is at it. The feature that first needs one of these creates it, at the path
// this declares.
//
// Atrium logs to standard error rather than to a file (plan 3, at T1), so Log
// is the only member here that names a directory nothing is ever likely to
// write to. It is answered rather than left empty because a client reading
// seven paths and finding six is reading a different response shape, and
// because the field is where a file log would go the day one is added.
//
// # The layout follows the reference's own defaults
//
// Not for compatibility — no client reads these, and they differ per
// installation in any differential run — but because an operator who has run
// the reference already knows where to look:
//
//	metadata            [source: Emby.Server.Implementations/ServerApplicationPaths.cs:36 @ v10.11.11]
//	cache/transcodes    [source: MediaBrowser.Common/Configuration/EncodingConfigurationExtensions.cs:35 @ v10.11.11]
//
// ItemsByName is the same value as InternalMetadata rather than a directory of
// its own, which is the reference's own assignment: it fills both fields from
// InternalMetadataPath
// [source: Emby.Server.Implementations/SystemManager.cs:71-72 @ v10.11.11].
// Two fields carrying one value is a fact about the response, so it is written
// once here instead of twice at the handler.
type Paths struct {
	// ProgramData is the data directory itself — what --data-dir named.
	ProgramData string

	// Web is where a web client's files would be served from. Atrium serves
	// none: the v1 surface is 59 API routes and no static content
	// (Principle VI), so nothing reads this today.
	Web string

	// ItemsByName is where per-artist, per-genre and per-studio images live.
	// It is InternalMetadata, exactly.
	ItemsByName string

	// Cache is derived data that may be deleted between starts.
	Cache string

	// Log is where a file log would be written. Atrium logs to standard error.
	Log string

	// InternalMetadata is where downloaded artwork and metadata land — 004's,
	// when it arrives.
	InternalMetadata string

	// TranscodingTemp is where 008 would write a transcode's output.
	TranscodingTemp string
}

// PathsFor derives the layout from the one path an operator configures.
//
// The data directory is used exactly as it was given, not resolved to an
// absolute path or through its symbolic links. What the response reports is
// then what the operator wrote and what the "starting" log line already
// printed, and those three agreeing is worth more than an absolute path a
// reader would have to match up by hand.
func PathsFor(dataDirectory string) Paths {
	cache := filepath.Join(dataDirectory, "cache")
	metadata := filepath.Join(dataDirectory, "metadata")

	return Paths{
		ProgramData:      dataDirectory,
		Web:              filepath.Join(dataDirectory, "web"),
		ItemsByName:      metadata,
		Cache:            cache,
		Log:              filepath.Join(dataDirectory, "log"),
		InternalMetadata: metadata,
		TranscodingTemp:  filepath.Join(cache, "transcodes"),
	}
}
