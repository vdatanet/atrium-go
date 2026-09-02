---
feature: NNN-kebab-case-name
title: <Human-readable title> — tasks
status: Draft
created: YYYY-MM-DD
updated: YYYY-MM-DD
plan_status_required: Accepted
---

# NNN — Tasks

Ordered. Each task is a reviewable change on its own, and states how you know it worked.

No task may say "implement the feature". If one does, it needs breaking down.

## Legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked (say by what)

---

## T1 — <Short imperative title>

- [ ] **Changes:** which files, what they do afterwards
- **Depends on:** —
- **Verified by:** the exact command or test, and what its output must show
- **Spec reference:** §N of `spec.md`

## T2 — …

---

## Definition of done

The feature is done when **all** of these hold:

- [ ] Every acceptance criterion in `spec.md` §5 has a passing test.
- [ ] Every endpoint reaches the conformance level declared in `spec.md` §6.
- [ ] `docs/compatibility/surface.yaml` lists every route added, and no route exists outside it.
- [ ] Anything learned during implementation is back in `spec.md`, in this same change.
- [ ] Any new measured Jellyfin behaviour is in `docs/compatibility/behaviours.md` with provenance.
- [ ] `spec.md`, `plan.md` and `tasks.md` are all marked `Implemented`.
