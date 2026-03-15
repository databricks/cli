# Variable Interpolation: Character Scanner Parser

Author: Shreyas Goenka
Date: 12 March 2026

## Motivation

DABs variable interpolation (`${...}`) was regex-based. This caused:

1. **Silent failures** — `${foo.bar-}` silently treated as literal text with no warning.
2. **No suggestions** — `${bundle.nme}` produces "reference does not exist" with no hint.
3. **No escape mechanism** — no way to produce a literal `${` in output.
4. **No extensibility** — cannot support structured path features like key-value references `tasks[task_key="x"]` that exist in `libs/structs/structpath`.

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

See `libs/interpolation/parse.go`.

### Nested `${` rejection

Nested `${...}` inside a reference (e.g., `${var.foo_${var.tail}}`) is
rejected as an error. This construct is ambiguous and was never intentionally
supported — the old regex happened to match only the innermost pair by
coincidence.

### `\$` escape sequence

`\$` produces a literal `$`, and `\\` produces a literal `\`. This follows
the same convention used by Bash for escaping `$` and is the least
surprising option for users working in shell environments.

A standalone `\` before any character other than `$` or `\` is passed
through as a literal backslash, so existing configurations that happen to
contain backslashes are not affected.

### Malformed reference warnings

A standalone `WarnMalformedReferences` mutator walks the config tree once
before variable resolution. It only checks values with source locations
(`len(v.Locations()) > 0`) to avoid false positives from synthesized values
(e.g., normalized/computed paths).

### Token-based resolution

The resolver's string interpolation changed from `strings.Replace` (with
count=1 to avoid double-replacing duplicate refs) to a token concatenation
loop. Each `TokenRef` maps 1:1 to a resolved value, eliminating the ambiguity.

## Python sync

The Python regex in `python/databricks/bundles/core/_transform.py` needs a
corresponding update in a follow-up PR.
