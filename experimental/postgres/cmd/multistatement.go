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
// statement-terminating ';' is found anywhere except trailing whitespace.
//
// The scanner ignores ';' inside:
//   - single-quoted strings ('a;b', SQL standard doubled-quote escape)
//   - double-quoted identifiers ("col;name")
//   - line comments (-- ... \n)
//   - block comments (/* ... */, non-nesting)
//   - dollar-quoted bodies ($tag$ ... $tag$, optional tag)
//
// Over-rejection on weird syntactic edge cases is acceptable: users get a
// clear error and can split into multiple input units. v2 may swap this for
// a real Postgres tokenizer.
func checkSingleStatement(sql string) error {
	s := sql
	// Trim trailing whitespace once so a single trailing ';' is allowed.
	end := len(strings.TrimRight(s, " \t\r\n"))

	i := 0
	for i < end {
		c := s[i]

		switch c {
		case ';':
			// A ';' that's not at end-of-trimmed-input is a separator.
			if i < end-1 {
				return errMultipleStatements
			}
			// Trailing ';' is fine.
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

func scanLineComment(s string, start, end int) int {
	i := start + 2
	for i < end && s[i] != '\n' {
		i++
	}
	return i
}

func scanBlockComment(s string, start, end int) int {
	i := start + 2
	for i+1 < end {
		if s[i] == '*' && s[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return end
}

// readDollarTag inspects s[start] (which must be '$') and returns the tag
// between the two dollar signs and the index right after the closing first
// '$' of $tag$. If the construct doesn't look like a valid dollar-quote
// opener, returns ("", start) so the caller can fall through.
//
// Tag rule: starts after '$', runs to the next '$'. Per the Postgres docs a
// dollar-quote tag must not start with a digit, so we reject `$1`, `$2`,
// etc. as tags and let the scanner treat them as ordinary bytes (this is
// what `$1`-style parameter placeholders look like, even though `QueryExecModeExec`
// can't bind them in this command). Empty tag is valid: $$ is a marker,
// $$body$$ is the body.
func readDollarTag(s string, start, end int) (string, int) {
	i := start + 1
	for i < end {
		if s[i] == '$' {
			tag := s[start+1 : i]
			return tag, i + 1
		}
		// Reject `$<digit>...` early: it can't be a valid tag.
		if i == start+1 && s[i] >= '0' && s[i] <= '9' {
			return "", start
		}
		// Stop at characters that can't be in a tag.
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == ';' {
			return "", start
		}
		i++
	}
	return "", start
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
