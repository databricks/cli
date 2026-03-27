# Variable Interpolation: Character Scanner Parser

Author: Shreyas Goenka
Date: 12 March 2026

## Motivation

DABs variable interpolation (`${...}`) was regex-based. This caused:

1. **Silent failures** — `${foo.bar-}` silently treated as literal text with no warning.
2. **No suggestions** — `${bundle.nme}` produces "reference does not exist" with no hint.
3. **No extensibility** — cannot support structured path features like key-value references `tasks[task_key="x"]` that exist in `libs/structs/structpath`.

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
proper error reporting without the complexity.

## Design Decisions

### Two-mode character scanner

A two-mode scanner (TEXT / REFERENCE) that produces a flat list of tokens.
No AST, no recursive descent. Easy to port to the Python implementation.

See `libs/interpolation/parse.go`.

### Nested `${` support

Nested `${...}` inside a reference (e.g., `${var.foo_${var.tail}}`) is
supported. When the parser encounters a nested `${` inside an outer
reference, it treats the outer `${...` prefix as literal text so the inner
reference is resolved first. Multi-round resolution (up to 11 rounds in
`resolve_variable_references.go`) then progressively resolves from inside
out. For example, `${a_${b_${c}}}` resolves in 3 rounds.

### No escape sequences

Escape sequences (e.g. `\$` → `$`) are intentionally omitted. Variable
resolution uses multi-round processing (up to 11 rounds) where each round
re-parses all string values. Escape sequences consumed in round N would
produce bare `${...}` text that round N+1 would incorrectly resolve as a
real variable reference. A safe escape mechanism would require either
deferred consumption (escapes survive all rounds and are consumed in a
final pass) or sentinel characters, adding significant complexity. Since
there is no existing user demand for literal `${` in output, we defer
escape support to a follow-up if needed.

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

PyDABs uses a regex in `python/databricks/bundles/core/_transform.py` to
detect pure variable references (entire string is a single `${...}`). This
regex must stay in sync with the Go parser's key/path validation. Shared
test cases in `libs/interpolation/testdata/variable_references.json` are
consumed by both Go (`TestParsePureVariableReferences`) and Python
(`test_pure_variable_reference`) to verify agreement.
