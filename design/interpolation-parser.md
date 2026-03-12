# Variable Interpolation: Parser & "Did You Mean" Suggestions

Author: Shreyas Goenka
Date: 12 March 2026

## Motivation

DABs variable interpolation (`${...}`) was regex-based. This caused:

1. **Silent failures** — `${foo.bar-}` silently treated as literal text with no warning.
2. **No suggestions** — `${bundle.nme}` produces "reference does not exist" with no hint.
3. **No escape mechanism** — no way to produce a literal `${` in output.

## Background: How Other Systems Parse `${...}`

| System | Strategy | Escape | Error Quality |
|--------|----------|--------|---------------|
| Go `text/template` | State-function lexer | None | Line + template name |
| HCL2 (Terraform) | Ragel FSM + recursive descent | `$${` → literal `${` | Source range + suggestions |
| Python f-strings | Mode-stack tokenizer | `{{` → `{` | Line/column |
| Rust `format!` | Iterator-based descent | `{{`/`}}` | Spans + suggestions |
| Bash | Char-by-char + depth tracking | `\$` | Line number |

For a syntax as simple as `${path.to.var[0]}` (no nesting, no functions, no
operators), a full recursive descent parser is overkill. A **two-mode character
scanner** — the same core pattern used by Go's `text/template` and HCL — gives
proper error reporting and escape support without the complexity.

## Design Decisions

### Two-mode character scanner

A two-mode scanner (TEXT / REFERENCE) that produces a flat list of tokens.
No AST, no recursive descent. Easy to port to the Python implementation.

See `libs/dyn/dynvar/interpolation/parse.go`.

### Nested `${` handling

Existing configs use patterns like `${var.foo_${var.tail}}` where the inner
reference resolves first. The old regex matched only `${var.tail}` (the
innermost pair). The new parser preserves this: when scanning for `}` inside
a reference, if another `${` is encountered, the outer `${` is abandoned
(treated as literal) and scanning restarts from the inner `${`.

### `$$` escape sequence

Following HCL2's precedent, `$$` before `{` produces a literal `$`. This is
backward compatible — no existing config uses `$${` (the old regex wouldn't
match it).

### Malformed reference warnings

A standalone `WarnMalformedReferences` mutator walks the config tree once
before variable resolution. It only checks values with source locations
(`len(v.Locations()) > 0`) to avoid false positives from synthesized values
(e.g., normalized/computed paths).

### "Did you mean" suggestions

When a valid-syntax reference fails to resolve (`NoSuchKeyError`), the
resolver calls a `SuggestFn` that walks the config tree component by
component using Levenshtein distance. The suggestion is appended to the
existing error: `did you mean ${var.my_cluster_id}?`.

The `SuggestFn` receives the raw path from the reference (e.g., `var.X`),
rewrites it to `variables.X.value` for lookup, then converts the suggestion
back to `var.X` form for user-facing messages.

See `libs/dyn/dynvar/suggest.go`.

### Token-based resolution

The resolver's string interpolation changed from `strings.Replace` (with
count=1 to avoid double-replacing duplicate refs) to a token concatenation
loop. Each `TokenRef` maps 1:1 to a resolved value, eliminating the ambiguity.

## Python sync

The Python regex in `python/databricks/bundles/core/_transform.py` needs a
corresponding update in a follow-up PR.
