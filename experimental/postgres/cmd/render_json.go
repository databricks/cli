package postgrescmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
)

// jsonSink streams query results as JSON.
//
// For rows-producing statements the output is a top-level array of row
// objects. We use the separator-before-element pattern to avoid the
// "rewrite the trailing comma" trick and keep the JSON parseable even when
// iteration ends with a partial result (caller closes the array on OnError).
//
// For command-only statements the output is a single object describing the
// command tag.
type jsonSink struct {
	out    io.Writer
	stderr io.Writer

	// columns are the disambiguated column names: duplicates beyond the first
	// occurrence are renamed to "<name>__2", "<name>__3", etc. Postgres
	// allows duplicate output names (`SELECT 1, 1`, joins with two unaliased
	// `id` columns) but JSON consumers usually want unique keys; we rename
	// deterministically and warn once on stderr.
	columns []string
	oids    []uint32

	// hasOpenedArray is true once the leading `[\n` has been written. Used
	// by OnError to decide whether to emit the closing `]\n` to keep stdout
	// well-formed.
	hasOpenedArray bool
	// rowsWritten counts emitted rows so the separator decision is trivial:
	// 0 means "first row, no separator", anything else means "separator first".
	rowsWritten int
}

func newJSONSink(out, stderr io.Writer) *jsonSink {
	return &jsonSink{out: out, stderr: stderr}
}

func (s *jsonSink) Begin(fields []pgconn.FieldDescription) error {
	if len(fields) == 0 {
		// Command-only; we wait until End to emit the {"command": ...} object.
		return nil
	}

	s.columns = make([]string, len(fields))
	s.oids = make([]uint32, len(fields))
	seen := make(map[string]int, len(fields))
	dupes := false
	for i, f := range fields {
		s.oids[i] = f.DataTypeOID
		name := f.Name
		seen[name]++
		if seen[name] > 1 {
			dupes = true
			name = fmt.Sprintf("%s__%d", f.Name, seen[name])
		}
		s.columns[i] = name
	}
	if dupes {
		fmt.Fprintln(s.stderr, "Warning: query returned duplicate column names; renamed duplicates to <name>__N. Use AS aliases for stable names.")
	}

	if _, err := io.WriteString(s.out, "[\n"); err != nil {
		return err
	}
	s.hasOpenedArray = true
	return nil
}

func (s *jsonSink) Row(values []any) error {
	if s.rowsWritten > 0 {
		if _, err := io.WriteString(s.out, ",\n"); err != nil {
			return err
		}
	}

	// Emit keys in column order. json.Marshal on a map sorts keys
	// alphabetically; SELECT order is what consumers expect, so we write
	// `{`, walk columns, encode key:value pairs ourselves, then `}`.
	if _, err := io.WriteString(s.out, "{"); err != nil {
		return err
	}
	for i, name := range s.columns {
		if i > 0 {
			if _, err := io.WriteString(s.out, ","); err != nil {
				return err
			}
		}
		key, err := marshalJSON(name)
		if err != nil {
			return fmt.Errorf("encode column name %q: %w", name, err)
		}
		if _, err := s.out.Write(key); err != nil {
			return err
		}
		if _, err := io.WriteString(s.out, ":"); err != nil {
			return err
		}
		val, err := marshalJSON(jsonValueWithOID(values[i], s.oids[i]))
		if err != nil {
			return fmt.Errorf("encode value for %q: %w", name, err)
		}
		if _, err := s.out.Write(val); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(s.out, "}"); err != nil {
		return err
	}
	s.rowsWritten++
	return nil
}

// marshalJSON encodes v with HTML escaping disabled (so jsonb values like
// {"url":"<a>"} round-trip without `&lt;` rewrites). encoding/json's Encoder
// is the only path that exposes SetEscapeHTML, so we route through it and
// strip the trailing newline it always appends.
func marshalJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func (s *jsonSink) End(commandTag string) error {
	if s.hasOpenedArray {
		if s.rowsWritten == 0 {
			// Empty result: collapse to "[]\n" rather than "[\n\n]\n".
			_, err := io.WriteString(s.out, "]\n")
			return err
		}
		_, err := io.WriteString(s.out, "\n]\n")
		return err
	}
	// Command-only path: emit a single ordered object.
	if _, err := io.WriteString(s.out, `{"command":`); err != nil {
		return err
	}
	verbBytes, err := marshalJSON(commandTagVerb(commandTag))
	if err != nil {
		return fmt.Errorf("encode command tag verb: %w", err)
	}
	if _, err := s.out.Write(verbBytes); err != nil {
		return err
	}
	if rows, ok := commandTagRowCount(commandTag); ok {
		if _, err := fmt.Fprintf(s.out, `,"rows_affected":%d`, rows); err != nil {
			return err
		}
	}
	_, err = io.WriteString(s.out, "}\n")
	return err
}

// OnError closes the array cleanly so stdout remains parseable JSON. The
// caller still propagates the original error, which the command writes to
// stderr.
func (s *jsonSink) OnError(err error) {
	if !s.hasOpenedArray {
		return
	}
	// Best-effort; if this Write fails the stream is already corrupted
	// and there is nothing more we can do.
	if s.rowsWritten == 0 {
		_, _ = io.WriteString(s.out, "]\n")
		return
	}
	_, _ = io.WriteString(s.out, "\n]\n")
}

// commandTagVerb extracts the leading verb from a Postgres command tag (e.g.
// "INSERT 0 5" -> "INSERT"). Returns the input unchanged if there is no space.
func commandTagVerb(tag string) string {
	for i, r := range tag {
		if r == ' ' {
			return tag[:i]
		}
	}
	return tag
}

// commandTagRowCount extracts the trailing row count from a Postgres command
// tag. INSERT tags have the shape "INSERT <oid> <rows>"; UPDATE/DELETE/SELECT
// have "VERB <rows>". Returns ok=false for tags without a trailing integer
// (e.g. "CREATE DATABASE", "SET").
func commandTagRowCount(tag string) (int64, bool) {
	for i := len(tag) - 1; i >= 0; i-- {
		if tag[i] == ' ' {
			n, err := strconv.ParseInt(tag[i+1:], 10, 64)
			if err != nil {
				return 0, false
			}
			return n, true
		}
	}
	return 0, false
}
