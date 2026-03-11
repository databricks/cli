# Proposal: Replace Regex-Based Variable Interpolation with a Proper Parser

## Status: Draft

## Background

DABs variable interpolation (`${...}` syntax) is currently implemented via regex
matching in `libs/dyn/dynvar/ref.go`:

```go
baseVarDef = `[a-zA-Z]+([-_]*[a-zA-Z0-9]+)*`
re = regexp.MustCompile(
    fmt.Sprintf(`\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}`, baseVarDef, baseVarDef),
)
```

This regex is used in `NewRef()` via `FindAllStringSubmatch` to extract all
`${...}` references from a string. The matched substrings are then resolved by
the pipeline in `resolve.go` (collect → resolve → replace).

### Problems with the Regex Approach

1. **No error reporting.** A non-match produces zero information — the user
   gets no feedback about *why* `${foo.bar-}` or `${foo..bar}` is silently
   ignored. Invalid references are indistinguishable from literal text.

2. **No position information.** Errors cannot point to the character where
   parsing fails. When resolution *does* fail, the error messages refer to the
   matched string but not its location within the original value.

3. **Hard to extend.** Adding new syntax (e.g., default values like
   `${var.name:-default}`, or function calls like `${upper(var.name)}`) requires
   modifying a regex that is already at the edge of readability.

4. **No escape mechanism.** There is no way to produce a literal `${` in a
   string. Users who need `${` in their output (e.g., shell scripts, ARM
   templates) have no workaround.

5. **Dual maintenance burden.** The regex must be kept in sync with a Python
   regex in `python/databricks/bundles/core/_transform.py` — a fragile
   arrangement with no automated enforcement.

6. **Silent acceptance of ambiguous input.** The regex approach cannot
   distinguish between "this string has no variable references" and "this string
   has a malformed variable reference that should be reported."

## Research: How Other Systems Parse `${...}`

| System | Strategy | Escape | Nesting | Error Quality |
|--------|----------|--------|---------|---------------|
| Go `text/template` | State-function lexer | None | Paren depth | Line + template name |
| HCL2 (Terraform) | Ragel FSM + recursive descent | `$${` → literal `${` | Brace depth stack | Source range + suggestions |
| Python f-strings (PEP 701) | Mode-stack tokenizer | `{{` → `{` | Mode stack | Line/column |
| Rust `format!` | Iterator-based descent | `{{`/`}}` | N/A | Spans + suggestions |
| Bash | Char-by-char + depth tracking | `\$` | Full recursive | Line number |

**Key insight from the research:** For a syntax as simple as `${path.to.var[0]}`
(no nested expressions, no function calls, no operators), a full recursive
descent parser is overkill. The right tool is a **two-mode character scanner** —
the same pattern used by Go's `text/template` lexer and HCL's scanner at their
core. This gives us proper error reporting, escape support, and extensibility
without the complexity of a full parser generator.

## Proposed Design

### Architecture: Two-Phase Scanner

Replace the regex with a small, explicit scanner that operates in two modes:

```
Mode 1: TEXT      — accumulate literal characters
Mode 2: REFERENCE — accumulate variable path characters inside ${...}
```

The scanner produces a flat list of tokens. No AST, no recursive descent — just
a linear scan that cleanly separates literal text from variable references.

### Token Types

```go
// TokenKind represents the type of a parsed token.
type TokenKind int

const (
    TokenLiteral TokenKind = iota // Literal text (no interpolation)
    TokenRef                       // Variable reference: content between ${ and }
)
```

### Core Data Structure

```go
// Token represents a parsed segment of an interpolation string.
type Token struct {
    Kind  TokenKind
    Value string // For Literal: the text. For Ref: the variable path (e.g., "a.b[0].c").
    Start int    // Byte offset of the start of this token in the original string.
    End   int    // Byte offset past the end of this token.
}
```

### Scanner Implementation

```go
// Parse parses a string that may contain ${...} variable references.
// It returns a slice of tokens representing literal text and variable references.
//
// Escape sequences:
//   - "$$" produces a literal "$" (only when followed by "{")
//
// Examples:
//   - "hello"                → [Literal("hello")]
//   - "${a.b}"               → [Ref("a.b")]
//   - "pre ${a.b} post"      → [Literal("pre "), Ref("a.b"), Literal(" post")]
//   - "$${a.b}"              → [Literal("${a.b}")]
//   - "${a.b} ${c[0]}"       → [Ref("a.b"), Literal(" "), Ref("c[0]")]
func Parse(s string) ([]Token, error) {
    var tokens []Token
    i := 0
    buf := strings.Builder{} // accumulates literal text

    flushLiteral := func(end int) {
        if buf.Len() > 0 {
            tokens = append(tokens, Token{
                Kind:  TokenLiteral,
                Value: buf.String(),
                Start: end - buf.Len(),
                End:   end,
            })
            buf.Reset()
        }
    }

    for i < len(s) {
        if s[i] != '$' {
            buf.WriteByte(s[i])
            i++
            continue
        }

        // We see '$'. Look ahead.
        if i+1 >= len(s) {
            // Trailing '$' — treat as literal.
            buf.WriteByte('$')
            i++
            continue
        }

        switch s[i+1] {
        case '$':
            // Escape: "$$" → literal "$".
            buf.WriteByte('$')
            i += 2

        case '{':
            // Start of variable reference.
            flushLiteral(i)
            refStart := i
            i += 2 // skip "${"

            // Scan the variable path until we find '}'.
            pathStart := i
            for i < len(s) && s[i] != '}' {
                i++
            }

            if i >= len(s) {
                return nil, fmt.Errorf(
                    "unterminated variable reference at position %d",
                    refStart,
                )
            }

            path := s[pathStart:i]
            i++ // skip '}'

            if path == "" {
                return nil, fmt.Errorf(
                    "empty variable reference at position %d",
                    refStart,
                )
            }

            // Validate the path content.
            if err := validatePath(path, refStart); err != nil {
                return nil, err
            }

            tokens = append(tokens, Token{
                Kind:  TokenRef,
                Value: path,
                Start: refStart,
                End:   i,
            })

        default:
            // '$' not followed by '$' or '{' — treat as literal.
            buf.WriteByte('$')
            i++
        }
    }

    flushLiteral(i)
    return tokens, nil
}
```

### Path Validation

Rather than encoding path rules in the regex, validate path contents explicitly
after extraction. This function reuses the existing `dyn.NewPathFromString` but
adds character-level error reporting:

```go
// validatePath checks that a variable path is well-formed.
// It wraps dyn.NewPathFromString with position-aware error messages.
func validatePath(path string, refStart int) error {
    _, err := dyn.NewPathFromString(path)
    if err != nil {
        return fmt.Errorf(
            "invalid variable reference ${%s} at position %d: %w",
            path, refStart, err,
        )
    }
    return nil
}
```

We should also add validation for the character set used in path segments. The
current regex implicitly enforces `[a-zA-Z]` start and `[a-zA-Z0-9_-]`
continuation. This should move to an explicit check inside path validation:

```go
func validatePathSegment(seg string) error {
    if len(seg) == 0 {
        return fmt.Errorf("empty path segment")
    }
    if seg[0] < 'A' || (seg[0] > 'Z' && seg[0] < 'a') || seg[0] > 'z' {
        return fmt.Errorf("path segment must start with a letter, got %q", seg[0])
    }
    // ... check continuation characters ...
}
```

### Updated Ref Type

The `Ref` struct changes from storing raw regex match groups to storing parsed
tokens:

```go
type Ref struct {
    Value  dyn.Value // Original dyn.Value.
    Str    string    // Original string content.
    Tokens []Token   // Parsed tokens (literals and references).
}

func NewRef(v dyn.Value) (Ref, bool) {
    s, ok := v.AsString()
    if !ok {
        return Ref{}, false
    }

    tokens, err := Parse(s)
    if err != nil {
        // Return error through a new error-aware API (see Migration section).
        return Ref{}, false
    }

    // Check if any token is a reference.
    hasRef := false
    for _, t := range tokens {
        if t.Kind == TokenRef {
            hasRef = true
            break
        }
    }
    if !hasRef {
        return Ref{}, false
    }

    return Ref{Value: v, Str: s, Tokens: tokens}, true
}
```

### Updated Resolution Logic

The string interpolation in `resolveRef` simplifies from a regex-replacement
loop to a token-concatenation loop:

```go
func (r *resolver) resolveRef(ref Ref, seen []string) (dyn.Value, error) {
    deps := ref.References()
    resolved := make([]dyn.Value, len(deps))
    complete := true

    // ... resolve deps (unchanged) ...

    // Pure substitution (single ref, no literals).
    if ref.IsPure() && complete {
        return dyn.NewValue(resolved[0].Value(), ref.Value.Locations()), nil
    }

    // String interpolation: concatenate tokens.
    var buf strings.Builder
    refIdx := 0
    for _, tok := range ref.Tokens {
        switch tok.Kind {
        case TokenLiteral:
            buf.WriteString(tok.Value)
        case TokenRef:
            if !resolved[refIdx].IsValid() {
                // Skipped — write original ${...} back.
                buf.WriteString("${")
                buf.WriteString(tok.Value)
                buf.WriteByte('}')
            } else {
                s, err := valueToString(resolved[refIdx])
                if err != nil {
                    return dyn.InvalidValue, err
                }
                buf.WriteString(s)
            }
            refIdx++
        }
    }

    return dyn.NewValue(buf.String(), ref.Value.Locations()), nil
}
```

This is cleaner than the current approach which uses `strings.Replace` with a
count of 1 — a trick needed to avoid double-replacing when the same variable
appears multiple times.

### Helper Methods on Ref

```go
// IsPure returns true if the string is a single variable reference with no
// surrounding text (e.g., "${a.b}" but not "x ${a.b}" or "${a} ${b}").
func (v Ref) IsPure() bool {
    return len(v.Tokens) == 1 && v.Tokens[0].Kind == TokenRef
}

// References returns the variable paths referenced in this string.
func (v Ref) References() []string {
    var out []string
    for _, t := range v.Tokens {
        if t.Kind == TokenRef {
            out = append(out, t.Value)
        }
    }
    return out
}
```

### Escape Sequence: `$$`

Following HCL2's precedent, `$$` before `{` produces a literal `$`. This is the
most natural escape for users already familiar with Terraform/HCL:

| Input | Output |
|-------|--------|
| `${a.b}` | *(resolved value of a.b)* |
| `$${a.b}` | `${a.b}` (literal) |
| `$$notbrace` | `$notbrace` (literal) |
| `$notbrace` | `$notbrace` (literal) |

This is backward compatible: `$$` is not a valid prefix today (the regex
requires `${`), so no existing config uses `$$` in a way that would change
meaning.

## File Changes

| File | Change |
|------|--------|
| `libs/dyn/dynvar/ref.go` | Replace regex + `Matches` with `Parse()` + `[]Token` |
| `libs/dyn/dynvar/ref_test.go` | Update tests: add parser tests, keep behavioral tests |
| `libs/dyn/dynvar/resolve.go` | Update `resolveRef` to use token concatenation |
| `libs/dyn/dynvar/resolve_test.go` | Add tests for escape sequences, error messages |
| `libs/dyn/dynvar/parse.go` | **New file**: scanner + token types |
| `libs/dyn/dynvar/parse_test.go` | **New file**: scanner unit tests |
| `python/databricks/bundles/core/_transform.py` | Update Python side to match (separate PR) |

## Migration Strategy

### Phase 1: Add Parser, Keep Regex

1. Implement `Parse()` in a new `parse.go` file with full test coverage.
2. Add a `NewRefWithDiagnostics(v dyn.Value) (Ref, diag.Diagnostics)` that
   uses the parser and can return warnings for malformed references.
3. Keep the existing `NewRef` as-is, calling the parser internally but falling
   back to the regex for any parse errors (belt-and-suspenders).
4. Add logging when the parser and regex disagree, to catch discrepancies.

### Phase 2: Switch Over

1. Remove the regex fallback — `NewRef` uses only the parser.
2. Update `Ref` to store `[]Token` instead of `[][]string`.
3. Update `resolveRef` to use token concatenation.
4. Remove the `re` variable.

### Phase 3: Add Escape Support

1. Enable `$$` escape handling in the parser.
2. Document the escape sequence.
3. Update the Python implementation.

## Compatibility

- **Forward compatible:** All strings that currently contain valid `${...}`
  references will parse identically. The parser accepts a strict superset of
  the regex (it can also report errors for malformed references).

- **Backward compatible escape:** `$$` before `{` is a new feature, not a
  breaking change. No existing valid config contains `$${` (the regex would not
  match it, and a literal `$${` in YAML has no special meaning today).

- **Error reporting is additive:** Strings that silently failed to match the
  regex will now produce actionable error messages. This is a UX improvement,
  not a breaking change, though it could surface new warnings for configs that
  previously "worked" by accident (e.g., a typo like `${foo.bar-}` was silently
  treated as literal text).

## Testing Plan

1. **Parser unit tests** (`parse_test.go`):
   - Valid references: single, multiple, with indices, with hyphens/underscores
   - Escape sequences: `$$`, `$` at end of string, `$` before non-`{`
   - Error cases: unterminated `${`, empty `${}`, invalid characters in path
   - Position tracking: verify `Start`/`End` offsets are correct

2. **Ref behavioral tests** (`ref_test.go`):
   - All existing tests pass unchanged
   - New tests for `IsPure()` and `References()` using token-based `Ref`

3. **Resolution tests** (`resolve_test.go`):
   - All existing tests pass unchanged
   - New tests for escape sequences in interpolation
   - New tests verifying improved error messages

4. **Acceptance tests**:
   - Add acceptance test with `$$` escape in `databricks.yml`
   - Verify existing acceptance tests pass without output changes

## Why Not a More Powerful Parser?

A recursive descent parser or parser combinator would allow richer syntax (nested
expressions, function calls, filters). We deliberately avoid this because:

1. **YAGNI.** The current `${path.to.var[0]}` syntax covers all use cases. There
   are no open feature requests for computed expressions inside `${...}`.

2. **Two implementations.** Any syntax change must be mirrored in the Python
   implementation. A simple scanner is easy to port; a recursive descent parser
   is not.

3. **Terraform alignment.** DABs variable references are conceptually similar to
   HCL variable references. Keeping the syntax simple avoids user confusion
   about what expressions are supported.

If we ever need richer expressions, the token-based architecture makes it easy to
add a parser layer on top of the scanner without changing the `Ref`/`Token` types
or the resolution pipeline.

## Summary

Replace the regex in `dynvar/ref.go` with a ~80-line character scanner that:
- Produces the same results for all valid inputs
- Reports actionable errors for invalid inputs (with byte positions)
- Supports `$$` escape for literal `${` output
- Is straightforward to read, test, and extend
- Simplifies the interpolation logic in `resolve.go` from regex-replacement to
  token concatenation
