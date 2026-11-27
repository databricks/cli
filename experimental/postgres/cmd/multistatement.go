package postgrescmd

import (
	"errors"
	"strings"
)

// errMultipleStatements is the typed error returned by checkSingleStatement
// when the input contains more than one ';'-separated statement. The runQuery
// path catches this with errors.Is to attach the multi-input workaround
// pointer in the user-visible message.
var errMultipleStatements = errors.New("input contains multiple statements (a ';' separates two or more statements)")

// checkSingleStatement walks sql and returns errMultipleStatements if a
// statement-terminating ';' is followed by anything other than whitespace,
// comments and further ';'. Postgres drops the empty statements those form.
//
// The scanner ignores ';' inside:
//   - single-quoted strings ('a;b', SQL standard doubled-quote escape)
//   - double-quoted identifiers ("col;name")
//   - line comments (-- ... up to \n or \r)
//   - block comments (/* ... */, nesting as in Postgres)
//   - dollar-quoted bodies ($tag$ ... $tag$, optional tag)
//
// Over-rejection on weird syntactic edge cases is acceptable: users get a
// clear error and can split into multiple input units. v2 may swap this for
// a real Postgres tokenizer.
func checkSingleStatement(sql string) error {
	s := sql
	end := len(s)

	terminated := false
	i := 0
	for i < end {
		c := s[i]

		if terminated && !isSQLSpace(c) && c != ';' && !startsComment(s, i, end) {
			return errMultipleStatements
		}

		switch c {
		case ';':
			terminated = true
			i++

		case '\'':
			// Single-quoted string. SQL standard escape is '' (doubled).
			i = scanQuoted(s, i, end, '\'')

		case '"':
			// Double-quoted identifier. Same '"' doubling escape rule.
			i = scanQuoted(s, i, end, '"')

		case '-':
			// Line comment "--" runs to next newline.
			if i+1 < end && s[i+1] == '-' {
				i = scanLineComment(s, i, end)
			} else {
				i++
			}

		case '/':
			// Block comment "/* ... */".
			if i+1 < end && s[i+1] == '*' {
				i = scanBlockComment(s, i, end)
			} else {
				i++
			}

		case '$':
			// Dollar-quoted body: $tag$ ... $tag$ (tag may be empty).
			tag, end2 := readDollarTag(s, i, end)
			if tag != "" || end2 > i {
				i = scanDollarBody(s, end2, end, tag)
			} else {
				i++
			}

		default:
			i++
		}
	}

	return nil
}

// scanQuoted advances past a quoted string or identifier opened at s[start]
// with the given quote character. SQL standard doubles the quote to escape
// (e.g. doubling the quote inside the string). Returns the index of the byte AFTER the closing quote, or
// end if the string is unterminated (over-permissive: an unterminated string
// at EOF means there's no ';' inside it anyway).
func scanQuoted(s string, start, end int, quote byte) int {
	i := start + 1
	for i < end {
		if s[i] == quote {
			if i+1 < end && s[i+1] == quote {
				i += 2 // doubled-quote escape
				continue
			}
			return i + 1
		}
		i++
	}
	return end
}

// isSQLSpace reports whether c is whitespace to the Postgres lexer.
func isSQLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// startsComment reports whether a "--" or "/*" comment starts at s[i].
func startsComment(s string, i, end int) bool {
	return i+1 < end && ((s[i] == '-' && s[i+1] == '-') || (s[i] == '/' && s[i+1] == '*'))
}

// scanLineComment advances past a "--" comment opened at s[start]. Like the
// Postgres lexer, it ends the comment at either '\n' or '\r'.
func scanLineComment(s string, start, end int) int {
	i := start + 2
	for i < end && s[i] != '\n' && s[i] != '\r' {
		i++
	}
	return i
}

// scanBlockComment advances past a "/*" comment opened at s[start]. Postgres
// nests block comments, so the comment ends at the "*/" that closes the
// outermost "/*". Returns end if unterminated.
func scanBlockComment(s string, start, end int) int {
	depth := 1
	i := start + 2
	for i+1 < end {
		switch {
		case s[i] == '/' && s[i+1] == '*':
			depth++
			i += 2
		case s[i] == '*' && s[i+1] == '/':
			depth--
			i += 2
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return end
}

// readDollarTag inspects s[start] (which must be '$') and returns the tag
// between the two dollar signs and the index right after the closing first
// '$' of $tag$. If the construct doesn't look like a valid dollar-quote
// opener, returns ("", start) so the caller can fall through.
//
// Tag rule: a Postgres dollar-quote tag is an unquoted identifier — first
// char is ASCII letter or underscore, subsequent chars are letter/digit/
// underscore. We reject anything outside that grammar so e.g. `$1`
// (parameter placeholder) and `$foo-bar$ ... $foo-bar$` (PG parses as two
// statements because `-` is not a tag char) are NOT treated as dollar
// quotes by this scanner. Empty tag is valid: `$$` is a marker, `$$body$$`
// is the body.
func readDollarTag(s string, start, end int) (string, int) {
	i := start + 1
	for i < end {
		c := s[i]
		if c == '$' {
			return s[start+1 : i], i + 1
		}
		// First tag char must be a letter or underscore.
		if i == start+1 {
			if !isTagStart(c) {
				return "", start
			}
			i++
			continue
		}
		// Subsequent tag chars: letter, digit, or underscore.
		if !isTagCont(c) {
			return "", start
		}
		i++
	}
	return "", start
}

func isTagStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isTagCont(c byte) bool {
	return isTagStart(c) || (c >= '0' && c <= '9')
}

// scanDollarBody advances past a $tag$...$tag$ body starting at start (the
// byte right after the opening tag's closing '$'). Returns the index of the
// byte AFTER the closing tag, or end if unterminated.
func scanDollarBody(s string, start, end int, tag string) int {
	close := "$" + tag + "$"
	idx := strings.Index(s[start:end], close)
	if idx < 0 {
		return end
	}
	return start + idx + len(close)
}
