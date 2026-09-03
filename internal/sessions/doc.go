// Package sessions is the domain of sessions: the derived session identity,
// and what a caller may see of the session list.
//
// It imports no HTTP and no store. 002 plan 3 splits it from internal/users
// because a session outlives the credential that opened it and is read by
// features that never look at an account — 007 writes NowPlayingItem into a
// session row without touching a user, and 008's delivery routes hold a session
// identifier. The dependency runs one way: this package knows a user by
// identifier and never the reverse, so nothing here imports internal/users.
//
// Two things live here and they are not the same kind of thing. DeriveID is
// arithmetic over two strings. Visible is a rule with an order, and the order is
// the reference's — 002 plan 6.10 makes it one function precisely so that no
// caller can compose the halves the wrong way round.
package sessions
