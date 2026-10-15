package postgrescmd

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONValue_PrimitiveTypes(t *testing.T) {
	assert.Equal(t, true, jsonValue(true))
	assert.Equal(t, "hello", jsonValue("hello"))
	assert.Equal(t, int64(42), jsonValue(int64(42)))
	assert.InDelta(t, 3.14, jsonValue(float64(3.14)), 1e-9)
}

func TestJSONValue_NULL(t *testing.T) {
	assert.Nil(t, jsonValue(nil))
}

func TestJSONValue_FloatSpecials(t *testing.T) {
	assert.Equal(t, "NaN", jsonValue(math.NaN()))
	assert.Equal(t, "Infinity", jsonValue(math.Inf(1)))
	assert.Equal(t, "-Infinity", jsonValue(math.Inf(-1)))
}

func TestJSONValue_LargeIntPreservedAsString(t *testing.T) {
	big := int64(1<<53 + 1)
	assert.Equal(t, "9007199254740993", jsonValue(big))

	negBig := -int64(1<<53 + 1)
	assert.Equal(t, "-9007199254740993", jsonValue(negBig))
}

func TestJSONValue_SafeIntPreservedAsNumber(t *testing.T) {
	safe := int64(1<<53 - 1)
	assert.Equal(t, safe, jsonValue(safe))
}

func TestJSONValue_TimestampToRFC3339(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	v := jsonValue(tm)
	assert.Equal(t, "2024-01-15T10:30:00Z", v)
}

func TestJSONValueWithOID_JSONBPassthrough(t *testing.T) {
	raw := []byte(`{"id":9007199254740993,"name":"alice"}`)
	v := jsonValueWithOID(raw, pgtype.JSONBOID)

	encoded, err := json.Marshal(v)
	assert.NoError(t, err)
	assert.JSONEq(t, string(raw), string(encoded))
}

func TestJSONValueWithOID_ByteaToBase64(t *testing.T) {
	v := jsonValueWithOID([]byte{0xde, 0xad, 0xbe, 0xef}, pgtype.ByteaOID)
	assert.Equal(t, "3q2+7w==", v)
}

func TestJSONValueWithOID_FallsBackToJSONValue(t *testing.T) {
	assert.Equal(t, int64(42), jsonValueWithOID(int64(42), pgtype.Int8OID))
	assert.Nil(t, jsonValueWithOID(nil, pgtype.TextOID))
}

func TestTextValue_NULL(t *testing.T) {
	assert.Equal(t, "NULL", textValue(nil))
}

func TestTextValue_Bool(t *testing.T) {
	assert.Equal(t, "t", textValue(true))
	assert.Equal(t, "f", textValue(false))
}

func TestTextValue_BytesAsHex(t *testing.T) {
	assert.Equal(t, `\xdeadbeef`, textValue([]byte{0xde, 0xad, 0xbe, 0xef}))
}

func TestTextValue_Time(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	assert.Equal(t, "2024-01-15T10:30:00Z", textValue(tm))
}

func TestTextValue_FloatSpecials(t *testing.T) {
	assert.Equal(t, "NaN", textValue(math.NaN()))
	assert.Equal(t, "Infinity", textValue(math.Inf(1)))
	assert.Equal(t, "-Infinity", textValue(math.Inf(-1)))
}

func TestTextValue_FiniteFloat(t *testing.T) {
	assert.Equal(t, "3.14", textValue(float64(3.14)))
	assert.Equal(t, "0", textValue(float64(0)))
}

func TestDecodeTextValue_DateTime(t *testing.T) {
	v, err := decodeTextValue(pgtype.DateOID, []byte("10000-01-02"))
	require.NoError(t, err)
	assert.Equal(t, time.Date(10000, 1, 2, 0, 0, 0, 0, time.UTC), v)

	v, err = decodeTextValue(pgtype.DateOID, []byte("0044-03-15 BC"))
	require.NoError(t, err)
	assert.Equal(t, time.Date(-43, 3, 15, 0, 0, 0, 0, time.UTC), v)

	v, err = decodeTextValue(pgtype.TimestampOID, []byte("2024-01-02 03:04:05.123456"))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 2, 3, 4, 5, 123456000, time.UTC), v)

	v, err = decodeTextValue(pgtype.TimestamptzOID, []byte("2024-06-01 10:00:00.5+02"))
	require.NoError(t, err)
	assert.True(t, time.Date(2024, 6, 1, 8, 0, 0, 500000000, time.UTC).Equal(v.(time.Time)))

	v, err = decodeTextValue(pgtype.DateOID, []byte("infinity"))
	require.NoError(t, err)
	assert.Equal(t, "infinity", v)
}

func TestDecodeTextValue_Bytea(t *testing.T) {
	v, err := decodeTextValue(pgtype.ByteaOID, []byte(`\xdeadbeef`))
	require.NoError(t, err)
	assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, v)

	_, err = decodeTextValue(pgtype.ByteaOID, []byte(`\336`))
	assert.Error(t, err)
}

func TestJSONValueWithOID_InvalidUTF8(t *testing.T) {
	v := jsonValueWithOID("{\"x\":\"\xe9\"}", pgtype.JSONOID)
	encoded, err := json.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `{"x":"�"}`, string(encoded))
}
