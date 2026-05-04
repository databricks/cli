package cmdio

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SGR (Select Graphic Rendition) escapes; see
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR
const (
	ansiReset     = "\x1b[0m"
	ansiGreen     = "\x1b[32m"
	ansiBoldGreen = "\x1b[32;1m"
	ansiRed       = "\x1b[31m"
	ansiCyan      = "\x1b[36m"
	ansiMagenta   = "\x1b[35m"
	ansiBoldBlue  = "\x1b[34;1m"
)

// marshalJSON returns indented JSON, optionally colorized for TTY output.
func marshalJSON(v any, colorize bool) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	if !colorize {
		return b, nil
	}
	return colorizeJSON(b), nil
}

// colorizeJSON wraps each JSON token in ANSI color escapes. The input must
// already be valid indented JSON (e.g. from json.MarshalIndent); the walker
// trusts that structure and does not re-validate it.
func colorizeJSON(b []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(b) + len(b)/2)

	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == '"':
			end := scanString(b, i)
			tok := b[i:end]
			if isObjectKey(b, end) {
				writeColored(&out, ansiBoldBlue, tok)
			} else {
				writeColored(&out, ansiGreen, tok)
			}
			i = end
		case c == 't' && hasLiteral(b, i, "true"):
			writeColored(&out, ansiBoldGreen, b[i:i+4])
			i += 4
		case c == 'f' && hasLiteral(b, i, "false"):
			writeColored(&out, ansiRed, b[i:i+5])
			i += 5
		case c == 'n' && hasLiteral(b, i, "null"):
			writeColored(&out, ansiMagenta, b[i:i+4])
			i += 4
		case c == '-' || (c >= '0' && c <= '9'):
			end := scanNumber(b, i)
			writeColored(&out, ansiCyan, b[i:end])
			i = end
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.Bytes()
}

func writeColored(out *bytes.Buffer, code string, tok []byte) {
	out.WriteString(code)
	out.Write(tok)
	out.WriteString(ansiReset)
}

// scanString returns the index just past the closing quote of a JSON string
// that begins at b[i] (which must be '"').
func scanString(b []byte, i int) int {
	j := i + 1
	for j < len(b) {
		switch b[j] {
		case '\\':
			j += 2
		case '"':
			return j + 1
		default:
			j++
		}
	}
	return len(b)
}

// scanNumber returns the index just past the JSON number that begins at b[i].
// It assumes b is valid JSON and does not validate the number itself.
func scanNumber(b []byte, i int) int {
	j := i
	for j < len(b) {
		c := b[j]
		if (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' {
			j++
			continue
		}
		break
	}
	return j
}

// isObjectKey reports whether the token ending at b[end] is followed (after
// optional whitespace) by a ':', i.e. it is an object key rather than a value.
func isObjectKey(b []byte, end int) bool {
	for j := end; j < len(b); j++ {
		switch b[j] {
		case ' ', '\t', '\n', '\r':
			continue
		case ':':
			return true
		default:
			return false
		}
	}
	return false
}

func hasLiteral(b []byte, i int, lit string) bool {
	return i+len(lit) <= len(b) && string(b[i:i+len(lit)]) == lit
}
