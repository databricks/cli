package postgrescmd

import (
	"bytes"
	"fmt"
	"io"
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
// Every per-unit object shares the same canonical key order:
//
//	{"source", "sql", "kind", "elapsed_ms", payload...}
//
// where payload depends on kind:
//
//	"rows":    {..., "rows": [...]}
//	"command": {..., "command": "...", "rows_affected": N}
//	"error":   {..., "error": {"message": "..."}}
//
// elapsed_ms is present on errors too: it captures how long the failing
// statement ran before the error fired.
func renderJSONMulti(out, stderr io.Writer, results []*unitResult, errored *unitResult, errMessage string) error {
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

	if errored != nil {
		if len(results) > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		obj := jsonErrorObject(errored, errMessage)
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
	if err := writeJSONUnitHeader(buf, r); err != nil {
		return err
	}

	if !r.IsRowsProducing() {
		buf.WriteString(`,"kind":"command"`)
		fmt.Fprintf(buf, `,"elapsed_ms":%d`, r.Elapsed.Milliseconds())
		verbBytes, err := marshalJSON(commandTagVerb(r.CommandTag))
		if err != nil {
			return err
		}
		buf.WriteString(`,"command":`)
		buf.Write(verbBytes)
		if rows, ok := commandTagRowCount(r.CommandTag); ok {
			fmt.Fprintf(buf, `,"rows_affected":%d`, rows)
		}
		buf.WriteString(`}`)
		return nil
	}

	// Rows-producing unit. Reuse jsonSink for the rows array body so the
	// per-row encoding (column order, type mapping) stays in one place.
	buf.WriteString(`,"kind":"rows"`)
	fmt.Fprintf(buf, `,"elapsed_ms":%d,"rows":`, r.Elapsed.Milliseconds())

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
	if err := sink.End(""); err != nil {
		return err
	}
	rowsTrimmed := bytes.TrimRight(rowsBuf.Bytes(), "\n")
	buf.Write(rowsTrimmed)
	buf.WriteString(`}`)
	return nil
}

// writeJSONUnitHeader writes the canonical {source, sql, ...} prefix used
// by every per-unit object. The closing brace and the kind-specific payload
// are appended by the caller.
func writeJSONUnitHeader(buf *bytes.Buffer, r *unitResult) error {
	sourceBytes, err := marshalJSON(r.Source)
	if err != nil {
		return err
	}
	sqlBytes, err := marshalJSON(r.SQL)
	if err != nil {
		return err
	}
	buf.WriteString(`{"source":`)
	buf.Write(sourceBytes)
	buf.WriteString(`,"sql":`)
	buf.Write(sqlBytes)
	return nil
}

// jsonErrorObject builds the per-unit error envelope used in the multi-input
// JSON shape. The buffered unitResult provides source, SQL, and the elapsed
// time captured by runUnitBuffered's error path. message is the
// already-formatted error wording (includes SQLSTATE / hint / detail for
// PgErrors).
//
// marshalJSON of a string never errors (encoding/json replaces invalid UTF-8
// with U+FFFD), so the inner errors are unreachable and we treat them as
// programming errors via panic.
func jsonErrorObject(r *unitResult, message string) []byte {
	var buf bytes.Buffer
	mustWriteJSONHeader(&buf, r)
	buf.WriteString(`,"kind":"error"`)
	fmt.Fprintf(&buf, `,"elapsed_ms":%d`, r.Elapsed.Milliseconds())
	buf.WriteString(`,"error":{"message":`)
	buf.Write(mustMarshalJSON(message))
	buf.WriteString(`}}`)
	return buf.Bytes()
}

// mustWriteJSONHeader is writeJSONUnitHeader with a panic instead of an
// error return. The only failure mode is an unreachable encoding/json error.
func mustWriteJSONHeader(buf *bytes.Buffer, r *unitResult) {
	if err := writeJSONUnitHeader(buf, r); err != nil {
		panic(fmt.Errorf("encoding json header: %w", err))
	}
}

// mustMarshalJSON is marshalJSON with a panic instead of an error return,
// for the same reason.
func mustMarshalJSON(v any) []byte {
	b, err := marshalJSON(v)
	if err != nil {
		panic(fmt.Errorf("encoding json value: %w", err))
	}
	return b
}
