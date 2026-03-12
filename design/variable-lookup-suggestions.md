# "Did You Mean?" Suggestions for Invalid Variable References

Author: Shreyas Goenka
Date: 12 March 2026

## Problem

`${bundle.git.origin_urlx}` produces `reference does not exist` with no hint.
A single character typo in a long path can take minutes to spot.

## Design: Fuzzy Path Walk

When a lookup fails with `NoSuchKeyError`, we do a separate fuzzy walk of the
normalized config tree (which includes all struct-defined fields via
`IncludeMissingFields` + all user-defined map keys).

The walk processes every component independently:
1. Exact key match → follow it
2. No exact match → Levenshtein fuzzy match against siblings → follow best match
3. Index components (`[0]`) → pass through verbatim
4. Any component unfixable (all candidates too far) → give up, no suggestion

This corrects **multiple** typos simultaneously (e.g., `${resources.jbs.my_jb.id}`
→ `did you mean ${resources.jobs.my_job.id}?`).

Distance threshold: `min(3, max(1, len(key)/2))`.

See `libs/dyn/dynvar/suggest.go`.

## Wiring

The suggestion callback is passed via `dynvar.WithSuggestFn(...)` into
`dynvar.Resolve`. On `NoSuchKeyError` in `resolveKey`, the suggestion is
appended to the error message.

## `var` Shorthand

`${var.foo}` is rewritten to `${variables.foo.value}` before lookup. The
`SuggestFn` in `resolve_variable_references.go` handles this bidirectionally:
rewrite `var.X` → `variables.X.value` for the fuzzy walk, then convert the
suggestion back to `var.X` form for the user-facing message.

## Scope

**Covered**: typos in struct fields, user-defined names, resource types/instances,
multi-level typos.

**Not covered**: malformed references (handled by the parser), cross-section
suggestions (user writes `${bundle.X}` meaning `${var.X}`), array index
out of bounds.
