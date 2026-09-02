# Constitution

These are the principles that do not bend. When a specification, a plan, a task or a pull request
conflicts with anything here, the conflict is resolved in favour of this document — or this
document is amended first, deliberately, with the reasoning written down.

Each principle states what it forbids, because a principle that forbids nothing decides nothing.

---

## I. Zero delta — the client must not be able to tell

Atrium implements the Jellyfin API. It does not extend it, improve it, or offer "a better way"
alongside it.

A client that works against Jellyfin must work against Atrium **with no branch, no capability
probe, and no configuration**. If a client needs to know which server it is talking to in order to
behave correctly, Atrium has failed, regardless of how good the reason was.

**Forbidden:**
- New endpoints that have no Jellyfin counterpart.
- New fields in responses that Jellyfin does not send.
- Different names, casings, types or units for anything Jellyfin already names.
- "Optional" extensions gated behind a header or query flag. An extension a client can discover is
  still a delta.

**Consequence:** every idea that would be genuinely nicer done differently gets written down in
`docs/compatibility/behaviours.md` as a *deliberate non-improvement*, and then not done.

> This principle is not an aesthetic preference. It is the entire value of the project: the moment
> Atrium speaks its own dialect, it stops being a drop-in server and becomes a third protocol that
> every client author has to care about.

---

## II. Behaviour is measured, not assumed

The reference is not the documentation, and not the OpenAPI schema. **The reference is what a
running Jellyfin actually does.**

Every compatibility claim in this repository must be traceable to one of:
- a probe script under `tools/` that was run against a real Jellyfin server, with the version and
  date recorded; or
- a **prior measurement** against a real Jellyfin server, carried forward with its version and
  date but without a reproducible script in this repository; or
- a cited line in the Jellyfin source (`file:line`, plus the version tag it was read at); or
- the OpenAPI document for the pinned version.

The second is weaker than the first and is marked differently on purpose. A prior measurement is a
real observation of a real server, but nobody can re-run it from this repository — so each one is a
standing debt, to be discharged by writing the probe script that reproduces it. They are listed in
`docs/compatibility/reference-target.md`.

**Forbidden:**
- "Jellyfin probably returns…" in a specification.
- A field, status code, header or unit stated without one of the three sources above.
- Copying a claim from a third-party wiki, blog or client library without re-measuring it.

An unverified claim is marked `⚠️ UNVERIFIED` inline and blocks the specification from leaving
draft status. It never silently becomes true by being repeated.

---

## III. Specification before implementation

This project practises Spec-Driven Development. The order is:

```
spec.md  (WHAT and WHY — no technology)
   ↓
plan.md  (HOW — architecture, stack, data model, contracts)
   ↓
tasks.md (verifiable steps, each with its acceptance check)
   ↓
code
```

**Forbidden:**
- Writing code for a feature whose `spec.md` is still in draft.
- Naming a library, framework, table or Python module inside `spec.md`. The specification
  describes observable behaviour; if it mentions a technology, it has stopped being a
  specification.
- Discovering a requirement during implementation and leaving it only in the code. It goes back
  into the spec, in the same change.

Documentation and code move **in the same commit**. A commit that changes behaviour without
updating the specification it came from is incomplete, not "to be documented later".

---

## IV. No forked code

Atrium is written from scratch. Jellyfin's source is read as a **behavioural reference**, the way
one reads a specification — never as a source of code to translate.

**Forbidden:**
- Transliterating a Jellyfin C# method into Python.
- Copying identifier names, comments, file structure or algorithm implementations from Jellyfin,
  Emby, or any of their plugins.
- Vendoring Jellyfin source or assets into this repository.

What *is* allowed, and necessary: reading Jellyfin to learn **what it does** — the shape of a
response, the unit of a field, the status code on an error path — and citing where that was
observed.

The distinction is between **interface** and **implementation**. We reimplement the interface. We
never carry over the implementation.

---

## V. Bug-for-bug where clients depend on it

Jellyfin has behaviours that are defects. Some of them clients have already worked around, and a
"correct" server breaks those workarounds.

When Atrium meets one, the choice is made by the procedure in `docs/compatibility/behaviours.md`
§3.0 and recorded there with three fields: **what Jellyfin does**, **whether any known client
depends on it**, and **what Atrium does**.

The procedure exists because "replicate unless you have a good argument" is a preference with a
disclaimer, not a rule. It turns on one question — *can a client have built something that being
correct would break?* — and it has produced opposite answers for two symptoms of a single upstream
bug, which is the clearest evidence that per-endpoint intuition is not good enough.

The default remains **replicate the defect**, because Principle I outranks correctness. Diverging
requires a written argument that no client can observe the difference, or that no client could have
worked around it in the first place.

**Forbidden, in addition to the above:**
- Treating a closed upstream pull request as a ruling on the behaviour. A PR can be closed for
  scope, process or bandwidth; none of those is a judgement.
- Deferring a decision on the grounds that upstream might fix it. Upstream is not a dependency.

**Forbidden:**
- Silently fixing a Jellyfin bug because it was obviously wrong.
- Replicating a bug without recording it, so that a later contributor "fixes" it.

---

## VI. Implement what is actually called

The Jellyfin API has 322 paths. Real clients call a small fraction of them. v1 implements the
fraction, derived by measurement from real client code, and documented in
`docs/compatibility/api-surface-v1.md`.

**Forbidden:**
- Adding an endpoint because it exists upstream, without a named consumer.
- Returning a plausible-looking stub from an endpoint that is not really implemented. An
  unimplemented endpoint returns the same thing Jellyfin returns when a feature is absent — or it
  is not routed at all. It never lies.

Growth of the surface is a scope decision recorded in the roadmap, not an implementation detail.

---

## VII. Determinism

The same library, scanned twice, produces the same identifiers, the same ordering and the same
results.

**Forbidden:**
- Item identifiers derived from insertion order, timestamps, or random values.
- Query results whose ordering depends on filesystem iteration order or dictionary ordering.
- Tests that depend on wall-clock time, network availability or the host's locale.

Identifiers are derived deterministically from stable inputs, so that a rescan — or a rebuild of
the database from an empty file — does not invalidate every client-side cache, favourite and
resume position.

---

## VIII. Every behaviour ships with a conformance check

A feature is done when a test asserts the behaviour **at the HTTP boundary**, in the same shape a
client sees it: status code, headers, and the exact JSON body.

**Forbidden:**
- Merging a route whose response shape is only checked by unit-testing the function behind it.
- Asserting on a parsed Python object where the client sees bytes. Casing, `null`-vs-absent and
  numeric type are part of the contract and only visible in the serialised form.

Where a real Jellyfin is reachable, the strongest check is **differential**: the same request
against both servers, with the responses compared field by field. See
`docs/compatibility/conformance.md`.

---

## IX. English, everywhere

All code, comments, identifiers, commit messages, branch names, issue text and documentation are
in English. The project is open source and intended to be readable by people who do not share the
author's first language.

**Forbidden:** any non-English text in the repository, including in commit messages and TODOs.

---

## X. Open source, and honest about its lineage

Atrium is published openly. Its relationship to Jellyfin is stated plainly in the README and never
obscured: it is an independent implementation of Jellyfin's API, unaffiliated with and not endorsed
by the Jellyfin project.

**Forbidden:**
- Presenting Atrium as a Jellyfin product, fork or successor.
- Using Jellyfin's name or marks in a way that suggests endorsement.
- Claiming compatibility that has not been measured.

---

## Amending this document

A principle is amended by a pull request that changes this file and nothing else, stating what
changed and what forced the change. Amendments are dated. The amendment log lives at the bottom of
this file.

### Amendment log

| Date | Change |
|---|---|
| 2026-08-26 | Initial ratification. |
