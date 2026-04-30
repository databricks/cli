package postgrescmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// renderTextMulti renders a sequence of unit results as plain text. Each
// per-unit block follows the single-input layout (table for rows-producing,
// command tag for command-only); successive blocks are separated by a blank
// line, mirroring `psql -c "...; ..."` shape.
//
// errIndex/errResult identifies the unit that errored (-1 if none); we still
// render any successful prefix. The error itself is surfaced by the caller
// via cobra's default error rendering.
func renderTextMulti(out io.Writer, results []*unitResult) error {
	for i, r := range results {
		if i > 0 {
			if _, err := io.WriteString(out, "\n"); err != nil {
				return err
			}
		}
		if err := renderTextResult(out, r); err != nil {
			return err
		}
	}
	return nil
}

// renderTextResult renders a single buffered unitResult in the same shape as
// textSink would for a streamed result.
func renderTextResult(out io.Writer, r *unitResult) error {
	if !r.IsRowsProducing() {
		_, err := fmt.Fprintln(out, r.CommandTag)
		return err
	}

	// Reuse textSink for the table layout so single-input and multi-input
	// share the same alignment and footer logic.
	sink := newTextSink(out)
	if err := sink.Begin(r.Fields); err != nil {
		return err
	}
	for _, row := range r.Rows {
		if err := sink.Row(row); err != nil {
			return err
		}
	}
	return sink.End(r.CommandTag)
}

// renderJSONMulti emits the wrapped multi-input JSON shape: a top-level
// array of result objects, one per input unit. Per-unit objects are buffered
// to completion before write; the outer array uses separator-before-element
// streaming. CSV multi-input is rejected pre-flight, so this function is
// only used for json.
//
// Per-unit shape:
//
//	{"sql": "...", "kind": "rows", "elapsed_ms": N, "rows": [...]}
//	{"sql": "...", "kind": "command", "elapsed_ms": N, "command": "...", "rows_affected": N}
//	{"sql": "...", "kind": "error", "elapsed_ms": N, "error": {...}}
//
// kind discriminates which fields are present so consumers don't have to
// branch on key presence.
func renderJSONMulti(out, stderr io.Writer, results []*unitResult, errIndex int, errMessage string) error {
	if _, err := io.WriteString(out, "[\n"); err != nil {
		return err
	}

	for i, r := range results {
		if i > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		var unitBuf bytes.Buffer
		if err := renderJSONUnit(&unitBuf, stderr, r); err != nil {
			return err
		}
		if _, err := out.Write(unitBuf.Bytes()); err != nil {
			return err
		}
	}

	if errIndex >= 0 {
		// The errored unit follows the last successful unit; write a comma
		// separator and the error envelope for it.
		if len(results) > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		errSQL := ""
		errSource := ""
		// errIndex points to the input *unit* index; since we render
		// successful units in order, the errored unit's SQL came from the
		// caller's units slice. The caller embeds it in errMessage so we
		// don't need separate plumbing here.
		obj := jsonErrorObject(errSource, errSQL, errMessage)
		if _, err := out.Write(obj); err != nil {
			return err
		}
	}

	_, err := io.WriteString(out, "\n]\n")
	return err
}

// renderJSONUnit writes one buffered result object to buf, using the
// existing single-input json rendering for the rows array so the value
// mapping stays consistent across single- and multi-input shapes.
func renderJSONUnit(buf *bytes.Buffer, stderr io.Writer, r *unitResult) error {
	if !r.IsRowsProducing() {
		// Command-only unit.
		if _, err := fmt.Fprintf(buf, `{"sql":`); err != nil {
			return err
		}
		sqlJSON, err := marshalJSON(r.SQL)
		if err != nil {
			return err
		}
		buf.Write(sqlJSON)
		fmt.Fprintf(buf, `,"kind":"command","elapsed_ms":%d`, r.Elapsed.Milliseconds())
		fmt.Fprintf(buf, `,"command":"%s"`, jsonEscapeShort(commandTagVerb(r.CommandTag)))
		if rows, ok := commandTagRowCount(r.CommandTag); ok {
			fmt.Fprintf(buf, `,"rows_affected":%d`, rows)
		}
		buf.WriteString(`}`)
		return nil
	}

	// Rows-producing unit. We reuse jsonSink for the rows array body so
	// the per-row encoding (column order, type mapping) stays in one place.
	if _, err := fmt.Fprintf(buf, `{"sql":`); err != nil {
		return err
	}
	sqlJSON, err := marshalJSON(r.SQL)
	if err != nil {
		return err
	}
	buf.Write(sqlJSON)
	fmt.Fprintf(buf, `,"kind":"rows","elapsed_ms":%d,"rows":`, r.Elapsed.Milliseconds())

	rowsBuf := &bytes.Buffer{}
	sink := newJSONSink(rowsBuf, stderr)
	if err := sink.Begin(r.Fields); err != nil {
		return err
	}
	for _, row := range r.Rows {
		if err := sink.Row(row); err != nil {
			return err
		}
	}
	// Use a no-op tag for End so jsonSink's success path emits the closing
	// bracket. The trailing newline gets trimmed below.
	if err := sink.End(""); err != nil {
		return err
	}
	rowsTrimmed := bytes.TrimRight(rowsBuf.Bytes(), "\n")
	buf.Write(rowsTrimmed)
	buf.WriteString(`}`)
	return nil
}

// jsonErrorObject builds the per-unit error envelope used in the multi-input
// JSON shape. message is the formatted error message (already includes
// SQLSTATE / hint / detail when applicable).
func jsonErrorObject(source, sql, message string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"source":`)
	if b, err := marshalJSON(source); err == nil {
		buf.Write(b)
	} else {
		buf.WriteString(`""`)
	}
	buf.WriteString(`,"sql":`)
	if b, err := marshalJSON(sql); err == nil {
		buf.Write(b)
	} else {
		buf.WriteString(`""`)
	}
	buf.WriteString(`,"kind":"error","error":{"message":`)
	if b, err := marshalJSON(message); err == nil {
		buf.Write(b)
	} else {
		buf.WriteString(`""`)
	}
	buf.WriteString(`}}`)
	return buf.Bytes()
}

// jsonEscapeShort is a fast path for short ASCII strings (command tag verbs)
// that need backslash escapes for ", \, and control bytes only. Falls back
// to a string-escaped value if the input contains anything unusual.
func jsonEscapeShort(s string) string {
	if !strings.ContainsAny(s, "\"\\\n\r\t") {
		return s
	}
	out, err := marshalJSON(s)
	if err != nil {
		return s
	}
	// marshalJSON returns the value with surrounding quotes; strip them so
	// the caller can wrap with its own quoting.
	return string(bytes.Trim(out, `"`))
}
