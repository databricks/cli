package postgrescmd

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// safeIntegerBound is the largest absolute integer value that can be
// represented exactly in IEEE 754 double precision. Beyond this, encoding an
// int64 as a JSON number silently loses precision in JavaScript-style
// consumers. We render those as JSON strings to preserve the original digits.
const safeIntegerBound = 1<<53 - 1

// textValue renders a Go value (as decoded by pgx) to its canonical Postgres
// text representation. Used by --output text and --output csv.
//
// NULL renders as the literal "NULL" so it lines up with the column rather
// than appearing as an empty cell. CSV converts that back to an empty field
// at write time (matches `psql --csv`).
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
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case fmt.Stringer:
		return x.String()
	}

	return fmt.Sprintf("%v", v)
}

// jsonValue renders a Go value (as decoded by pgx) to a JSON-encodable
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
	case int8, int16, int32, int, uint8, uint16, uint32:
		return x
	case int64:
		if x > safeIntegerBound || x < -safeIntegerBound {
			return strconv.FormatInt(x, 10)
		}
		return x
	case uint64:
		if x > safeIntegerBound {
			return strconv.FormatUint(x, 10)
		}
		return x
	case float32:
		return jsonFloat(float64(x))
	case float64:
		return jsonFloat(x)
	case []byte:
		// Postgres jsonb / json arrive as []byte holding raw JSON. Anything
		// else we'd like to base64-encode. We can't tell them apart from the
		// Go type alone; the sink calls jsonValueWithOID for oid-aware
		// disambiguation. This bare path is the conservative fallback and
		// treats unknown bytes as base64 (lossless and correct for bytea).
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
// places this matters today are JSON/JSONB and bytea: both arrive from pgx as
// []byte but want different JSON shapes (raw JSON passthrough vs base64).
func jsonValueWithOID(v any, oid uint32) any {
	if v == nil {
		return nil
	}

	switch oid {
	case pgtype.JSONOID, pgtype.JSONBOID:
		// pgx returns json/jsonb as already-decoded Go values when no codec
		// is registered; with the default codec, they decode to map/slice/etc.
		// In QueryExecModeExec text-mode, pgx returns the raw JSON bytes as
		// string (since the wire is text). We accept both shapes.
		switch x := v.(type) {
		case []byte:
			return json.RawMessage(x)
		case string:
			return json.RawMessage(x)
		}
	case pgtype.ByteaOID:
		if b, ok := v.([]byte); ok {
			return base64.StdEncoding.EncodeToString(b)
		}
	}

	return jsonValue(v)
}
