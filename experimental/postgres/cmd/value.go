package postgrescmd

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
)

// safeIntegerBound is the largest absolute integer value that can be
// represented exactly in IEEE 754 double precision. Beyond this, encoding an
// int64 as a JSON number silently loses precision in JavaScript-style
// consumers. We render those as JSON strings to preserve the original digits.
const safeIntegerBound = 1<<53 - 1

// decodeTextValue converts a text-format result column into the Go value the
// sinks render. Integers, floats, booleans, bytea and date/time types map to
// native Go types; everything else (numeric, uuid, interval, arrays, json,
// ...) stays as the server's canonical text, which is already the
// representation we want to print. Date/time values use pgtype's own text
// parsers, so they decode exactly as they did with pgx.Conn; infinity stays
// as text.
func decodeTextValue(oid uint32, raw []byte) (any, error) {
	if raw == nil {
		return nil, nil
	}
	if oid == pgtype.ByteaOID {
		hexPart, ok := bytes.CutPrefix(raw, []byte(`\x`))
		if !ok {
			return nil, fmt.Errorf("unsupported bytea text format %q", raw)
		}
		b := make([]byte, hex.DecodedLen(len(hexPart)))
		_, err := hex.Decode(b, hexPart)
		return b, err
	}
	s := string(raw)
	switch oid {
	case pgtype.Int2OID:
		v, err := strconv.ParseInt(s, 10, 16)
		return int16(v), err
	case pgtype.Int4OID:
		v, err := strconv.ParseInt(s, 10, 32)
		return int32(v), err
	case pgtype.Int8OID:
		return strconv.ParseInt(s, 10, 64)
	case pgtype.Float4OID:
		v, err := strconv.ParseFloat(s, 32)
		return float32(v), err
	case pgtype.Float8OID:
		return strconv.ParseFloat(s, 64)
	case pgtype.BoolOID:
		return s == "t", nil
	case pgtype.DateOID:
		var d pgtype.Date
		if err := d.Scan(s); err == nil && d.InfinityModifier == pgtype.Finite {
			return d.Time, nil
		}
	case pgtype.TimestampOID:
		var ts pgtype.Timestamp
		if err := ts.Scan(s); err == nil && ts.InfinityModifier == pgtype.Finite {
			return ts.Time, nil
		}
	case pgtype.TimestamptzOID:
		var ts pgtype.Timestamptz
		if err := ts.Scan(s); err == nil && ts.InfinityModifier == pgtype.Finite {
			return ts.Time, nil
		}
	}
	return s, nil
}

// textValue renders a Go value (as decoded by decodeTextValue) to its canonical Postgres
// text representation. Used by --output text and --output csv.
//
// NULL renders as the literal "NULL" so it lines up with the column rather
// than appearing as an empty cell. CSV converts that back to an empty field
// at write time (matches `psql --csv`).
//
// IEEE special floats use Postgres' canonical wording ("NaN" / "Infinity"
// / "-Infinity"), not Go's `fmt.Sprintf("%v")` defaults (which would emit
// "+Inf"/"-Inf"). Finite floats use Go's shortest-round-trip 'g' format,
// which may differ from psql in exponential vs fixed notation around the
// 'g' boundary (e.g. Go prints `1e+10`; psql prints `10000000000`). Full
// psql parity is not worth a custom formatter.
func textValue(v any) string {
	if v == nil {
		return "NULL"
	}

	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return `\x` + hex.EncodeToString(x)
	case bool:
		if x {
			return "t"
		}
		return "f"
	case float64:
		return floatTextForm(x)
	case float32:
		return floatTextForm(float64(x))
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case fmt.Stringer:
		return x.String()
	}

	return fmt.Sprintf("%v", v)
}

// floatTextForm formats a float using Postgres' canonical text wording for
// the IEEE specials and Go's shortest-round-trip 'g' format otherwise.
func floatTextForm(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// jsonValue renders a Go value (as decoded by decodeTextValue) to a JSON-encodable
// representation. Returns a value the standard json.Marshal can handle
// directly and the JSON shape we want; never returns Go values that would
// silently lose information (e.g. NaN, oversized integers).
//
// The mapping intentionally favours machine-friendly output:
//   - jsonb / json bytes round-trip as raw JSON (preserves bigint precision
//     inside JSON values, e.g. {"id": 9007199254740993}).
//   - bytea encodes as base64.
//   - timestamps render in RFC3339 with subsecond precision.
//   - Postgres NaN / +Inf / -Inf become JSON strings (JSON has no IEEE-special).
//   - Integers outside ±2^53 become JSON strings to preserve precision.
//   - Numerics, intervals, geometric types, and unknown types fall back to
//     the canonical Postgres text representation as a JSON string.
func jsonValue(v any) any {
	if v == nil {
		return nil
	}

	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x
	case int16, int32:
		return x
	case int64:
		// Postgres int8 decodes to Go int64. Outside the IEEE-754 safe
		// integer range we render as a string so JavaScript-style consumers
		// don't silently lose precision.
		if x > safeIntegerBound || x < -safeIntegerBound {
			return strconv.FormatInt(x, 10)
		}
		return x
	case float32:
		return jsonFloat(float64(x))
	case float64:
		return jsonFloat(x)
	case []byte:
		// bytea decodes to []byte; base64 keeps it lossless in JSON.
		return base64.StdEncoding.EncodeToString(x)
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case *big.Int:
		// numeric without scale; preserve as string to keep precision.
		return x.String()
	case fmt.Stringer:
		return x.String()
	}

	return fmt.Sprintf("%v", v)
}

// jsonFloat handles the IEEE-special cases that JSON cannot represent.
// Finite values pass through unchanged.
func jsonFloat(f float64) any {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return f
}

// jsonValueWithOID applies oid-aware overrides on top of jsonValue. The two
// places this matters today are JSON/JSONB and bytea: JSON arrives as its
// text and must pass through raw rather than as a JSON string; bytea arrives
// as []byte and is base64-encoded.
func jsonValueWithOID(v any, oid uint32) any {
	if v == nil {
		return nil
	}

	switch oid {
	case pgtype.JSONOID, pgtype.JSONBOID:
		switch x := v.(type) {
		case []byte:
			return rawJSON(x)
		case string:
			return rawJSON([]byte(x))
		}
	case pgtype.ByteaOID:
		if b, ok := v.([]byte); ok {
			return base64.StdEncoding.EncodeToString(b)
		}
	}

	return jsonValue(v)
}

// rawJSON passes JSON text through unchanged. Invalid UTF-8 (possible with a
// non-UTF8 client_encoding) would be copied into the output verbatim, so
// such values are decoded instead, which replaces it the way encoding/json
// does for strings.
func rawJSON(b []byte) any {
	if utf8.Valid(b) {
		return json.RawMessage(b)
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err == nil {
		return decoded
	}
	return string(b)
}
