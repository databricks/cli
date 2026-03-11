# Implementation Context for Variable Interpolation Improvements

This file captures the context needed to implement the two design proposals in
this directory. Read the design docs first, then use this as a map of the
codebase.

## Two Proposals

1. **`interpolation-parser.md`** — Replace regex-based `${...}` parsing with a
   proper two-mode character scanner. Adds escape support (`$$`), error messages
   with byte positions, and cleaner token-based interpolation.

2. **`variable-lookup-suggestions.md`** — Add "did you mean?" suggestions when
   a variable reference path doesn't exist. Uses Levenshtein distance and a
   fuzzy walk of the config tree to suggest corrections, including for
   multi-level typos.

These are independent and can be implemented in either order.

## Key Files to Modify

### Proposal 1: Parser

| File | What to do |
|------|------------|
| `libs/dyn/dynvar/parse.go` | **New**. Two-mode scanner: TEXT mode accumulates literals, REFERENCE mode scans `${...}`. Returns `[]Token`. See design doc for full pseudocode. |
| `libs/dyn/dynvar/parse_test.go` | **New**. Test valid refs, escapes (`$$`), errors (unterminated, empty, invalid chars). |
| `libs/dyn/dynvar/ref.go` | Replace regex + `Matches [][]string` with `Tokens []Token`. Update `NewRef`, `IsPure`, `References`. Remove the `re` regex variable. Keep `IsPureVariableReference` and `ContainsVariableReference` working. |
| `libs/dyn/dynvar/ref_test.go` | Update to match new internals. All existing behavioral tests should still pass. |
| `libs/dyn/dynvar/resolve.go` | Update `resolveRef` to use token concatenation instead of `strings.Replace` loop on regex matches. |
| `libs/dyn/dynvar/resolve_test.go` | Add tests for escape sequences. All existing tests must pass unchanged. |
| `python/databricks/bundles/core/_transform.py` | Has a Python regex that must stay in sync (separate PR). Comment on line 11-12 of `ref.go` references this. |

### Proposal 2: Suggestions

| File | What to do |
|------|------------|
| `libs/dyn/dynvar/suggest.go` | **New**. `levenshtein()`, `closestKeyMatch()`, `SuggestPath()`. |
| `libs/dyn/dynvar/suggest_test.go` | **New**. Unit tests for distance, matching, path suggestion (single typo, multi-level, with indices, no match). |
| `libs/dyn/mapping.go` | Add `StringKeys() []string` helper on `Mapping`. |
| `libs/dyn/dynvar/resolve.go` | Add `SuggestFn` field to `resolver`, `WithSuggestFn` option, use in `resolveKey` when `NoSuchKeyError`. |
| `libs/dyn/dynvar/resolve_test.go` | Integration tests with suggestions. |
| `bundle/config/mutator/resolve_variable_references.go` | Pass `WithSuggestFn` closure that calls `SuggestPath(normalized, p)`. |

## Current Code Structure

### `libs/dyn/dynvar/ref.go` — Variable Reference Detection
- `re` regex at line 14-15 matches `${path.to.var[0]}` patterns
- `baseVarDef = [a-zA-Z]+([-_]*[a-zA-Z0-9]+)*` — segment pattern
- `Ref` struct holds: `Value dyn.Value`, `Str string`, `Matches [][]string`
- `NewRef(v)` → uses `re.FindAllStringSubmatch(s, -1)`
- `IsPure()` → single match equals entire string (enables type retention)
- `References()` → extracts variable paths from match groups (`m[1]`)
- Must stay in sync with Python regex in `python/databricks/bundles/core/_transform.py`

### `libs/dyn/dynvar/resolve.go` — Resolution Pipeline
- 3-phase pipeline: collect → resolve → replace
- `collectVariableReferences()` — walks tree, calls `NewRef` on each string value
- `resolveVariableReferences()` — resolves refs in sorted key order (deterministic cycle detection)
- `resolveRef()` — resolves all deps, then either:
  - Pure substitution: single ref, retain original type
  - String interpolation: `strings.Replace(ref.Str, ref.Matches[j][0], s, 1)` for each match
- `resolveKey()` — catches `NoSuchKeyError` → `"reference does not exist: ${%s}"`. **This is where suggestion logic hooks in.**
- `ErrSkipResolution` — leaves variable reference in place for multi-pass resolution

### `libs/dyn/dynvar/lookup.go` — Lookup Interface
- `type Lookup func(path dyn.Path) (dyn.Value, error)`
- `ErrSkipResolution` sentinel error
- `DefaultLookup(in)` → creates lookup against a `dyn.Value`

### `libs/dyn/path_string.go` — Path Parsing
- `NewPathFromString("foo.bar[0].baz")` → `Path{Key("foo"), Key("bar"), Index(0), Key("baz")}`
- Already handles dots, brackets, indices
- **Reuse this for path validation** in the new parser (don't reimplement)

### `libs/dyn/visit.go` — Tree Traversal
- `noSuchKeyError{p Path}` — raised when map key doesn't exist
- At `visit.go:132-137`: has the map `m` with all sibling keys, but doesn't expose them in the error
- For suggestions, we do NOT modify this. Instead, `SuggestPath()` does its own walk.

### `libs/dyn/mapping.go` — Map Type
- `Mapping` struct with `pairs []Pair` and `index map[string]int`
- `GetByString(key)` → `(Value, bool)`
- `Keys()` → `[]Value`
- **Need to add**: `StringKeys() []string`

### `bundle/config/mutator/resolve_variable_references.go` — Bundle-Level Resolution
- Multi-round resolution (up to 11 rounds)
- Creates `normalized` tree via `convert.Normalize(b.Config, root, convert.IncludeMissingFields)`
- Rewrites `${var.foo}` → `${variables.foo.value}` before lookup
- Lookup function: `dyn.GetByPath(normalized, path)`
- `prefixes` control which paths are resolved vs skipped
- **This is where `WithSuggestFn` gets wired in**, passing `normalized` to `SuggestPath`

### `libs/dyn/walk.go` — Tree Walking
- `Walk(v, fn)` — walks all nodes, calls fn on each
- `CollectLeafPaths(v)` — returns all leaf paths as strings

## Important Invariants

1. **`IsPure()` must work identically** — when a string is exactly `${...}` with
   no surrounding text, the resolved value retains its original type (int, bool,
   map, etc.). The parser must preserve this semantic.

2. **`ErrSkipResolution` must work** — skipped variables are left as literal
   `${...}` text in the output. The token-based interpolation must handle this.

3. **Regex sync with Python** — the Python regex in `_transform.py` must
   eventually match whatever the parser accepts. For now, the parser should
   accept the same language as the regex (not broader).

4. **Sorted resolution order** — keys are resolved in sorted order for
   deterministic cycle detection errors. Don't change this.

5. **`var` shorthand** — `${var.X}` is rewritten to `${variables.X.value}`
   before lookup. Suggestions should reverse this for user-facing messages.

## Build & Test

```bash
# Run all unit tests
make test

# Run specific package tests
go test ./libs/dyn/dynvar/...

# Run acceptance tests (after changes)
go test ./acceptance/...

# Lint
make lintfull

# Format
make fmtfull
```
