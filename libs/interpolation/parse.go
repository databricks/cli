package interpolation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ParseError is returned by [Parse] when the input contains a malformed
// variable reference. Pos is the byte offset in the original string where
// the problematic reference starts.
type ParseError struct {
	Msg string
	Pos int
}

func (e *ParseError) Error() string {
	return e.Msg
}

// TokenKind represents the type of a parsed token.
type TokenKind int

const (
	TokenLiteral TokenKind = iota // Literal text (no interpolation)
	TokenRef                      // Variable reference: content between ${ and }
)

// Token represents a parsed segment of an interpolation string.
type Token struct {
	Kind  TokenKind
	Value string // For TokenLiteral: the literal text; For TokenRef: the path string (without ${})
	Start int    // Start position in original string
	End   int    // End position in original string (exclusive)
}

const (
	dollarChar = '$'
	openBrace  = '{'
	closeBrace = '}'
)

// keyPattern validates a single key segment in a variable path.
// Matches: [a-zA-Z]+([-_]*[a-zA-Z0-9]+)*
// Examples: "foo", "my-job", "a_b_c", "abc123"
//
// PyDABs uses a duplicate regex to detect pure variable references
// (python/databricks/bundles/core/_transform.py). The patterns must stay in
// sync. Cross-language test cases live in testdata/variable_references.json
// and are run by both Go (TestParsePureVariableReferences) and Python
// (test_pure_variable_reference). When changing key/index/path validation
// or reference syntax, add cases to that file so both languages are tested.
var keyPattern = regexp.MustCompile(`^[a-zA-Z]+([-_]*[a-zA-Z0-9]+)*$`)

// indexPattern matches one or more [N] index suffixes.
var indexPattern = regexp.MustCompile(`^(\[[0-9]+\])+$`)

// Parse parses a string that may contain ${...} variable references.
// It returns a slice of tokens representing literal text and variable references.
//
// Nested references like "${a.${b}}" are supported by treating the outer
// "${a." as literal text so that inner references are resolved first.
// After resolution the resulting string (e.g. "${a.x}") is re-parsed.
//
// Examples:
//   - "hello"            -> [Literal("hello")]
//   - "${a.b}"           -> [Ref("a.b")]
//   - "pre ${a.b} post"  -> [Literal("pre "), Ref("a.b"), Literal(" post")]
//   - "${a.${b}}"        -> [Literal("${a."), Ref("b"), Literal("}")]
func Parse(s string) ([]Token, error) {
	if len(s) == 0 {
		return nil, nil
	}

	var tokens []Token
	i := 0
	var buf strings.Builder
	litStart := 0 // tracks the start position in the original string for the current literal

	flushLiteral := func(end int) {
		if buf.Len() > 0 {
			tokens = append(tokens, Token{
				Kind:  TokenLiteral,
				Value: buf.String(),
				Start: litStart,
				End:   end,
			})
			buf.Reset()
		}
	}

	for i < len(s) {
		switch s[i] {
		case dollarChar:
			// We see '$'. Look ahead.
			if i+1 >= len(s) {
				// Trailing '$' at end of string: treat as literal.
				if buf.Len() == 0 {
					litStart = i
				}
				buf.WriteByte(dollarChar)
				i++
				continue
			}

			if s[i+1] != openBrace {
				// '$' not followed by '{': treat as literal.
				if buf.Len() == 0 {
					litStart = i
				}
				buf.WriteByte(dollarChar)
				i++
				continue
			}

			// Start of variable reference "${".
			refStart := i
			j := i + 2 // skip "${"

			// Scan until closing '}', handling nested '${'.
			pathStart := j
			nested := false
			for j < len(s) && s[j] != closeBrace {
				if s[j] == dollarChar && j+1 < len(s) && s[j+1] == openBrace {
					// Nested '${' found. Treat the outer "${..." prefix as
					// literal so inner references get resolved first.
					// E.g. "${a.${b}}" produces:
					//   [Literal("${a."), Ref("b"), Literal("}")]
					nested = true
					break
				}
				j++
			}

			if nested {
				if buf.Len() == 0 {
					litStart = refStart
				}
				buf.WriteString(s[refStart:j])
				i = j
				continue
			}

			if j >= len(s) {
				return nil, &ParseError{
					Msg: "unterminated variable reference",
					Pos: refStart,
				}
			}

			path := s[pathStart:j]
			j++ // skip '}'

			if path == "" {
				return nil, &ParseError{
					Msg: "empty variable reference",
					Pos: refStart,
				}
			}

			if err := validatePath(path); err != nil {
				return nil, &ParseError{
					Msg: fmt.Sprintf("invalid variable reference ${%s}: %s", path, err),
					Pos: refStart,
				}
			}

			flushLiteral(i)
			tokens = append(tokens, Token{
				Kind:  TokenRef,
				Value: path,
				Start: refStart,
				End:   j,
			})
			i = j

		default:
			if buf.Len() == 0 {
				litStart = i
			}
			buf.WriteByte(s[i])
			i++
		}
	}

	flushLiteral(i)
	return tokens, nil
}

// validatePath validates the path inside a ${...} reference by splitting on
// '.' and validating each segment individually.
func validatePath(path string) error {
	segments := strings.Split(path, ".")
	for _, seg := range segments {
		if seg == "" {
			return errors.New("invalid path")
		}

		// Strip trailing [N] index suffixes to get the key part.
		key, idx := splitKeyAndIndex(seg)

		if key == "" {
			return fmt.Errorf("invalid key %q", seg)
		}
		if !keyPattern.MatchString(key) {
			return fmt.Errorf("invalid key %q", key)
		}
		if idx != "" && !indexPattern.MatchString(idx) {
			return fmt.Errorf("invalid index in %q", seg)
		}
	}
	return nil
}

// splitKeyAndIndex splits "foo[0][1]" into ("foo", "[0][1]").
func splitKeyAndIndex(seg string) (string, string) {
	i := strings.IndexByte(seg, '[')
	if i < 0 {
		return seg, ""
	}
	return seg[:i], seg[i:]
}
