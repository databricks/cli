# Proposal: "Did You Mean?" Suggestions for Invalid Variable References

## Status: Draft

## Problem

When a user writes a variable reference like `${bundle.git.origin_urlx}` (typo)
or `${var.my_clster_id}` (misspelling), the error message today is:

```
reference does not exist: ${bundle.git.origin_urlx}
```

That's it. No suggestions, no list of valid keys, no indication of what went
wrong. The user has to go back to the docs or mentally diff against the schema
to figure out the correct name.

This is a common source of frustration: a single character typo in a long path
like `${resources.jobs.my_pipeline.tasks[0].task_key}` can take minutes to spot.

Variable references are multi-level paths (e.g., `${resources.jobs.my_job.id}`).
A typo can occur in **any** component — or even in **multiple** components at
once. A good suggestion system must handle all of these cases.

## What We Have Today

### The Error Path

When resolution fails, the call chain is:

```
resolve_variable_references.go:  dynvar.Resolve(v, lookupFn)
  resolve.go:                    r.resolveKey(dep, seen)
    resolve.go:                  r.fn(p)                     // calls the lookup function
      resolve_variable_references.go:  m.lookupFn(normalized, path, b)
        resolve_variable_references.go:  dyn.GetByPath(v, path)
          visit.go:                      m.GetByString(c.key)   // FAILS HERE
                                         return noSuchKeyError{path}
```

At the point of failure in `visit.go:135-137`:
```go
m := v.MustMap()
ev, ok := m.GetByString(c.key)
if !ok {
    return InvalidValue, noSuchKeyError{path}
}
```

The map `m` contains **all sibling keys** — i.e., the valid alternatives. But
that information is discarded. The error only carries the failed `path`.

Back in `resolve.go:200-201`, the error is rewrapped into a flat string:
```go
if dyn.IsNoSuchKeyError(err) {
    err = fmt.Errorf("reference does not exist: ${%s}", key)
}
```

**Crucially**, `visit.go` stops at the **first** non-existent key. For a path
like `${resources.jbs.my_jb.id}` with typos in both `jbs` and `my_jb`:
1. `resources` — exists, traverse into it
2. `jbs` — **does not exist** → `NoSuchKeyError` immediately

The traversal never reaches `my_jb`, so we can only suggest fixing `jbs`. The
user has to fix that, re-run, and discover the second typo. This round-trip
is exactly the frustration we want to avoid.

### The Normalized Config Tree

In `resolve_variable_references.go:203`, before resolution begins:
```go
normalized, _ := convert.Normalize(b.Config, root, convert.IncludeMissingFields)
```

This `normalized` tree includes:
- All struct-defined fields (from Go types), even if unset (via `IncludeMissingFields`)
- All user-defined map keys (resource names, variable names, etc.)

This is the right source of truth for suggestions. It contains everything the
user could validly reference.

## Design

### Approach: Fuzzy Path Walk Against the Config Tree

Rather than relying on the error from `visit.go` (which only tells us about the
first failing component), we do a **separate fuzzy walk** of the config tree when
a lookup fails. This walk processes every component in the reference path and
can fix typos in **multiple** components simultaneously.

The flow:
1. Lookup fails with `NoSuchKeyError`
2. We walk the reference path component by component against the normalized tree
3. At each component, if the exact key exists, we follow it
4. If not, we fuzzy-match against sibling keys and follow the best match
5. If all components are resolved (some via fuzzy matching), we suggest the
   corrected full path
6. If any component can't be fuzzy-matched (too far from all candidates), we
   give up on the suggestion

### Implementation

#### 1. Levenshtein Distance Utility

```go
// File: libs/dyn/dynvar/suggest.go

package dynvar

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
    if len(a) == 0 {
        return len(b)
    }
    if len(b) == 0 {
        return len(a)
    }

    // Use single-row DP to save memory.
    prev := make([]int, len(b)+1)
    for j := range prev {
        prev[j] = j
    }

    for i := range len(a) {
        curr := make([]int, len(b)+1)
        curr[0] = i + 1
        for j := range len(b) {
            cost := 1
            if a[i] == b[j] {
                cost = 0
            }
            curr[j+1] = min(
                curr[j]+1,      // insertion
                prev[j+1]+1,    // deletion
                prev[j]+cost,   // substitution
            )
        }
        prev = curr
    }

    return prev[len(b)]
}
```

#### 2. Single-Key Match Function

```go
// closestKeyMatch finds the closest matching key from a list of candidates.
// Returns the best match and its edit distance.
// Returns ("", -1) if no candidate is within the distance threshold.
func closestKeyMatch(key string, candidates []string) (string, int) {
    // Threshold: allow up to 3 edits, but no more than half the key length.
    // This avoids suggesting wildly different strings for short keys.
    maxDist := min(3, max(1, len(key)/2))

    bestMatch := ""
    bestDist := maxDist + 1

    for _, c := range candidates {
        d := levenshtein(key, c)
        if d < bestDist {
            bestDist = d
            bestMatch = c
        }
    }

    if bestMatch == "" {
        return "", -1
    }
    return bestMatch, bestDist
}
```

#### 3. Fuzzy Path Walk

This is the core new function. It walks the reference path against the config
tree, fuzzy-matching at each level:

```go
// suggestPath walks the reference path against root and tries to find the
// closest valid path. At each component, it first tries an exact match; if
// that fails, it fuzzy-matches against available keys.
//
// Returns the suggested path as a string, or "" if no reasonable suggestion
// can be made.
func suggestPath(root dyn.Value, refPath dyn.Path) string {
    current := root
    suggested := make(dyn.Path, 0, len(refPath))

    for _, component := range refPath {
        if component.IsIndex() {
            // For index components (e.g., [0]), we can't fuzzy-match.
            // Just check if the index is valid and pass through.
            s, ok := current.AsSequence()
            if !ok || component.Index() >= len(s) {
                return ""
            }
            suggested = append(suggested, component)
            current = s[component.Index()]
            continue
        }

        key := component.Key()
        m, ok := current.AsMap()
        if !ok {
            // Expected a map but got something else — can't suggest.
            return ""
        }

        // Try exact match first.
        if v, exists := m.GetByString(key); exists {
            suggested = append(suggested, component)
            current = v
            continue
        }

        // Exact match failed — try fuzzy match.
        candidates := m.StringKeys()
        match, _ := closestKeyMatch(key, candidates)
        if match == "" {
            // No close match — can't suggest beyond this point.
            return ""
        }

        suggested = append(suggested, dyn.Key(match))
        v, _ := m.GetByString(match)
        current = v
    }

    return suggested.String()
}
```

**Key properties:**
- Handles typos at **any** level: first, middle, last, or multiple levels
- Index components (`[0]`) are passed through verbatim — no fuzzy matching
- Stops suggesting as soon as any component can't be matched (no partial guesses)
- Each component is matched independently, so two typos in different components
  are both corrected

#### 4. Wire It Into Resolution

The suggestion logic needs access to the normalized config tree that the lookup
function uses. Today, the `Lookup` function type is:

```go
type Lookup func(path dyn.Path) (dyn.Value, error)
```

The `resolve.go` resolver doesn't have direct access to the underlying tree —
it only has the lookup function. We add the suggestion logic at the layer above,
in `resolve_variable_references.go`, which has access to the `normalized` tree.

**Option A: Pass a suggest function into the resolver**

Add an optional suggest callback to the resolver:

```go
// SuggestFn takes a failed reference path and returns a suggested correction,
// or "" if no suggestion can be made.
type SuggestFn func(path dyn.Path) string

func Resolve(in dyn.Value, fn Lookup, opts ...ResolveOption) (out dyn.Value, err error) {
    r := resolver{in: in, fn: fn}
    for _, opt := range opts {
        opt(&r)
    }
    return r.run()
}

type ResolveOption func(*resolver)

func WithSuggestFn(fn SuggestFn) ResolveOption {
    return func(r *resolver) {
        r.suggestFn = fn
    }
}
```

Then in `resolveKey`:
```go
v, err := r.fn(p)
if err != nil {
    if dyn.IsNoSuchKeyError(err) {
        msg := fmt.Sprintf("reference does not exist: ${%s}", key)

        if r.suggestFn != nil {
            if suggestion := r.suggestFn(p); suggestion != "" {
                msg += fmt.Sprintf("; did you mean ${%s}?", suggestion)
            }
        }

        err = fmt.Errorf(msg)
    }

    r.lookups[key] = lookupResult{v: dyn.InvalidValue, err: err}
    return dyn.InvalidValue, err
}
```

And in `resolve_variable_references.go`, pass the suggest function:

```go
return dynvar.Resolve(v, lookupFn, dynvar.WithSuggestFn(func(p dyn.Path) string {
    return dynvar.SuggestPath(normalized, p)
}))
```

**Option B: Suggest at the `resolve_variable_references.go` level**

Instead of modifying `Resolve`'s signature, wrap the error after `Resolve`
returns. This is simpler but less clean:

```go
out, err := dynvar.Resolve(v, lookupFn)
if err != nil && dyn.IsNoSuchKeyError(err) {
    // Extract the failed path and suggest...
}
```

The problem with Option B is that by the time `Resolve` returns, the original
`dyn.Path` is lost — it's been formatted into the error string. We'd have to
re-parse it or change the error type. **Option A is cleaner.**

### Example Error Messages

| Reference | Typos | Error After |
|-----------|-------|-------------|
| `${bundle.git.origin_urlx}` | 1 (leaf) | `did you mean ${bundle.git.origin_url}?` |
| `${resources.jbs.my_job.id}` | 1 (middle) | `did you mean ${resources.jobs.my_job.id}?` |
| `${resources.jbs.my_jb.id}` | 2 (middle + middle) | `did you mean ${resources.jobs.my_job.id}?` |
| `${bundel.git.origin_urlx}` | 2 (root + leaf) | `did you mean ${bundle.git.origin_url}?` |
| `${workspace.root_paht}` | 1 (leaf) | `did you mean ${workspace.root_path}?` |
| `${var.my_clster_id}` | 1 (leaf) | `did you mean ${var.my_cluster_id}?` |
| `${completely.wrong.path}` | all | *(no suggestion — too far at first component)* |
| `${resources.jobs.my_jb.idd}` | 2 (deep) | `did you mean ${resources.jobs.my_job.id}?` |

### Walk-Through: Multi-Level Typo

For `${resources.jbs.my_jb.id}`, the fuzzy walk proceeds:

```
Component    Tree at this level            Exact?  Fuzzy match
─────────    ──────────────────            ──────  ───────────
resources    {bundle, resources, ...}      yes     —
jbs          {jobs, pipelines, ...}        no      "jobs" (dist=1)
my_jb        {my_job, other_job, ...}      no      "my_job" (dist=2)
id           {id, name, ...}              yes     —

Suggested path: resources.jobs.my_job.id
```

All four components resolved, so we suggest `${resources.jobs.my_job.id}`.

### Walk-Through: Unfixable Path

For `${zzz.yyy.xxx}`:

```
Component    Tree at this level            Exact?  Fuzzy match
─────────    ──────────────────            ──────  ───────────
zzz          {bundle, resources, ...}      no      none (all dist>3)

Suggested path: "" (give up)
```

No suggestion produced.

## Scope

### What This Covers

- Typos in struct field names: `${bundle.git.origin_urlx}` (keys from Go types)
- Typos in user-defined names: `${var.my_clster_id}` (keys from user config)
- Typos in resource type names: `${resources.jbs.my_job.id}`
- Typos in resource instance names: `${resources.jobs.my_jb.id}`
- **Multi-level typos**: `${resources.jbs.my_jb.id}` (typos at two levels)

### What This Does NOT Cover

- **Invalid path structure** (e.g., `${a..b}` or `${a[x]}`) — this is a parse
  error, not a lookup error, and would be handled by the parser proposal.
- **References to the wrong section** (e.g., user writes `${bundle.cluster_id}`
  when they mean `${var.cluster_id}`) — the prefix is valid so we'd only
  suggest keys within `bundle.*`. Cross-section suggestions would require
  searching the entire tree, which is a separate feature.
- **Array index out of bounds** (e.g., `${resources.jobs.foo.tasks[99]}`) — this
  is an `indexOutOfBoundsError`, not a `noSuchKeyError`. No suggestions apply.

## `var` Shorthand

The `${var.foo}` shorthand is rewritten to `${variables.foo.value}` before
lookup (in `resolve_variable_references.go:209-222`). The suggestion function
receives the **rewritten** path. If we suggest a corrected path, we should
convert it back to the shorthand form for the user-facing message.

For example:
- User writes: `${var.my_clster_id}`
- Rewritten to: `${variables.my_clster_id.value}`
- Suggestion from fuzzy walk: `variables.my_cluster_id.value`
- User-facing message: `did you mean ${var.my_cluster_id}?`

This reverse mapping is straightforward: if the suggested path starts with
`variables.` and ends with `.value`, strip those and prefix with `var.`.

## File Changes

| File | Change |
|------|--------|
| `libs/dyn/mapping.go` | Add `StringKeys()` helper |
| `libs/dyn/dynvar/suggest.go` | **New**: `levenshtein()`, `closestKeyMatch()`, `SuggestPath()` |
| `libs/dyn/dynvar/suggest_test.go` | **New**: tests for distance, matching, and path suggestion |
| `libs/dyn/dynvar/resolve.go` | Add `SuggestFn` field, use it in `resolveKey` |
| `libs/dyn/dynvar/resolve_test.go` | Add tests for suggestion error messages |
| `bundle/config/mutator/resolve_variable_references.go` | Pass `WithSuggestFn` to `Resolve` |

Note: no changes to `libs/dyn/visit.go` — the suggestion logic is entirely
separate from the traversal error path.

## Testing

### Unit Tests for Levenshtein + Suggestions

```go
func TestLevenshtein(t *testing.T) {
    assert.Equal(t, 0, levenshtein("abc", "abc"))
    assert.Equal(t, 1, levenshtein("abc", "ab"))     // deletion
    assert.Equal(t, 1, levenshtein("abc", "abcd"))   // insertion
    assert.Equal(t, 1, levenshtein("abc", "adc"))    // substitution
    assert.Equal(t, 3, levenshtein("abc", "xyz"))    // all different
    assert.Equal(t, 3, levenshtein("", "abc"))        // empty vs non-empty
}

func TestClosestKeyMatch(t *testing.T) {
    candidates := []string{"origin_url", "branch", "commit"}

    match, dist := closestKeyMatch("origin_urlx", candidates)
    assert.Equal(t, "origin_url", match)
    assert.Equal(t, 1, dist)

    match, _ = closestKeyMatch("zzzzzzz", candidates)
    assert.Equal(t, "", match)
}
```

### Fuzzy Path Walk Tests

```go
func TestSuggestPathSingleTypo(t *testing.T) {
    root := dyn.V(map[string]dyn.Value{
        "bundle": dyn.V(map[string]dyn.Value{
            "git": dyn.V(map[string]dyn.Value{
                "origin_url": dyn.V(""),
                "branch":     dyn.V(""),
            }),
        }),
    })

    p := dyn.MustPathFromString("bundle.git.origin_urlx")
    assert.Equal(t, "bundle.git.origin_url", SuggestPath(root, p))
}

func TestSuggestPathMultiLevelTypo(t *testing.T) {
    root := dyn.V(map[string]dyn.Value{
        "resources": dyn.V(map[string]dyn.Value{
            "jobs": dyn.V(map[string]dyn.Value{
                "my_job": dyn.V(map[string]dyn.Value{
                    "id": dyn.V(""),
                }),
            }),
        }),
    })

    p := dyn.MustPathFromString("resources.jbs.my_jb.id")
    assert.Equal(t, "resources.jobs.my_job.id", SuggestPath(root, p))
}

func TestSuggestPathNoMatch(t *testing.T) {
    root := dyn.V(map[string]dyn.Value{
        "bundle": dyn.V(map[string]dyn.Value{
            "name": dyn.V(""),
        }),
    })

    p := dyn.MustPathFromString("zzzzz.yyyyy")
    assert.Equal(t, "", SuggestPath(root, p))
}

func TestSuggestPathWithIndex(t *testing.T) {
    root := dyn.V(map[string]dyn.Value{
        "resources": dyn.V(map[string]dyn.Value{
            "jobs": dyn.V(map[string]dyn.Value{
                "my_job": dyn.V(map[string]dyn.Value{
                    "tasks": dyn.V([]dyn.Value{
                        dyn.V(map[string]dyn.Value{
                            "task_key": dyn.V(""),
                        }),
                    }),
                }),
            }),
        }),
    })

    p := dyn.MustPathFromString("resources.jobs.my_job.tasks[0].tsk_key")
    assert.Equal(t, "resources.jobs.my_job.tasks[0].task_key", SuggestPath(root, p))
}
```

### Integration-Level Tests

```go
func TestResolveNotFoundWithSuggestion(t *testing.T) {
    in := dyn.V(map[string]dyn.Value{
        "bundle": dyn.V(map[string]dyn.Value{
            "name":   dyn.V("my-bundle"),
            "target": dyn.V("dev"),
        }),
        "ref": dyn.V("${bundle.nme}"),
    })

    _, err := dynvar.Resolve(in, dynvar.DefaultLookup(in),
        dynvar.WithSuggestFn(func(p dyn.Path) string {
            return dynvar.SuggestPath(in, p)
        }),
    )
    assert.ErrorContains(t, err, "reference does not exist: ${bundle.nme}")
    assert.ErrorContains(t, err, "did you mean ${bundle.name}?")
}

func TestResolveNotFoundMultiLevelTypo(t *testing.T) {
    in := dyn.V(map[string]dyn.Value{
        "resources": dyn.V(map[string]dyn.Value{
            "jobs": dyn.V(map[string]dyn.Value{
                "my_job": dyn.V(map[string]dyn.Value{
                    "id": dyn.V("123"),
                }),
            }),
        }),
        "ref": dyn.V("${resources.jbs.my_jb.id}"),
    })

    _, err := dynvar.Resolve(in, dynvar.DefaultLookup(in),
        dynvar.WithSuggestFn(func(p dyn.Path) string {
            return dynvar.SuggestPath(in, p)
        }),
    )
    assert.ErrorContains(t, err, "did you mean ${resources.jobs.my_job.id}?")
}
```

## Alternatives Considered

### A. Fix one component at a time (original proposal)

Only suggest a fix for the first failing component. After the user fixes that,
they re-run and discover the next typo.

**Rejected** because:
- Requires multiple round-trips for multi-level typos
- The fuzzy walk approach is barely more complex but gives a much better UX

### B. Enumerate all valid paths in the error

List all valid sibling keys:

```
reference does not exist: ${bundle.nme}; valid keys at "bundle" are: name, target, git
```

**Rejected** because for large maps (e.g., `resources.jobs` with dozens of jobs)
this would produce very noisy output. A single close match is more actionable.

### C. Search the entire tree for the closest leaf path

Walk the entire normalized tree and compute edit distance for every possible
leaf path against the full reference string.

**Rejected** because:
- Expensive for large configs (every leaf × string distance)
- Could suggest paths in completely unrelated sections
- The per-component walk is more predictable and faster (bounded by path depth)

### D. Do nothing — rely on docs/IDE support

**Rejected** because:
- Many users don't use an IDE for YAML editing
- The error happens at `databricks bundle validate` time, which is the right
  place for actionable feedback
- This is low-effort, high-value

## Relationship to Parser Proposal

This proposal is **independent** of the regex-to-parser migration. It can be
implemented with the current regex-based `NewRef` — the suggestion logic operates
at the resolution level, not the parsing level.

However, the two proposals complement each other:
- The parser proposal improves error reporting for **malformed** references
  (e.g., `${a..b}`, unterminated `${`)
- This proposal improves error reporting for **well-formed but invalid**
  references (e.g., `${bundle.nme}`)

Both can be implemented and shipped independently.
