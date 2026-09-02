// Package units holds the two types the wire measures things in: the tick and
// the date.
//
// It is a leaf. It imports nothing of ours and no HTTP, because
// architecture 2 has both the domain and the store holding these types —
// behaviours 1.3 puts ticks in *storage*, not only on the wire, "so no
// conversion can be forgotten at a boundary" — and a domain that had to import
// the serialiser to hold a duration would have inverted the whole diagram to
// get a number.
//
// # Why this package exists before anything sends a tick
//
// 001 sends neither a tick nor a date: the four routes it serves carry a
// string identity, a name, a version and a handful of booleans. The package is
// here anyway because 001 delivers the two cross-cutting L1 sweeps (spec 6),
// and the unit sweep needs a type to recognise (plan 3).
//
// That is also why the two types are named types rather than an int64 and a
// time.Time with a formatting helper beside them. The sweep's rule is not
// "every field whose name ends in Ticks", which is a heuristic:
// conformance L1's own correction is that three of nine date-valued fields
// observed in the reference — DateCreated, DateLastMediaAdded and
// LastPlaybackCheckIn — do not end in Date, so a sweep keyed on the suffix
// checks six of nine `[probe: tools/probe_wire_format, Jellyfin 10.11.11,
// 2026-09-02]`. A date has to be recognised by what it is, and there are two
// ways to ask that question of this package: a response model's field has type
// Time, or a value in a body satisfies IsTime.
package units
