// Package system is the domain half of feature 001: what this installation is,
// what it calls itself, whether its setup is finished, and which address it
// advertises to a requester.
//
// It imports no HTTP. architecture 2 puts the measured semantics in a layer
// that may reach for Ports and the unit types and for nothing else, because a
// rule that reaches for a request, a header or a status code can only be tested
// by issuing a request. The address choice of plan 3.4 arrives here as
// RequestFacts, filled in by the edge; the store arrives as the narrow
// interface plan 5 declares.
//
// What is here so far is the installation identity — the one piece of this
// package's state that lives in a file rather than in the store, and plan 4
// argues why — and the three-tier LocalAddress choice, which is a pure function
// of RequestFacts and AddressConfig and reaches for nothing else.
package system
