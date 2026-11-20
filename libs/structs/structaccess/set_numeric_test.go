package structaccess_test

import (
	"math"
	"testing"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type numericTarget struct {
	I    int      `json:"i,omitempty"`
	I8   int8     `json:"i8,omitempty"`
	I32  int32    `json:"i32,omitempty"`
	I64  int64    `json:"i64,omitempty"`
	U    uint     `json:"u,omitempty"`
	U8   uint8    `json:"u8,omitempty"`
	U64  uint64   `json:"u64,omitempty"`
	F32  float32  `json:"f32,omitempty"`
	F64  float64  `json:"f64,omitempty"`
	PI   *int     `json:"pi,omitempty"`
	PI32 *int32   `json:"pi32,omitempty"`
	PU   *uint    `json:"pu,omitempty"`
	PF32 *float32 `json:"pf32,omitempty"`
	PF64 *float64 `json:"pf64,omitempty"`
}

func TestSetNumericConversion(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value any
		// Exactly one of expected and err is set; expected has the type of the target field.
		expected any
		err      string
	}{
		// int <- float
		{
			name:     "int from integral float",
			path:     "i",
			value:    2.0,
			expected: 2,
		},
		{
			name:     "int from negative integral float",
			path:     "i",
			value:    -3.0,
			expected: -3,
		},
		{
			name:  "int from fractional float",
			path:  "i",
			value: 1.9,
			err:   "cannot set 1.9 to int: precision loss",
		},
		{
			name:  "int from NaN",
			path:  "i",
			value: math.NaN(),
			err:   "cannot set NaN to int: precision loss",
		},
		{
			name:  "int from +Inf",
			path:  "i",
			value: math.Inf(1),
			err:   "value +Inf overflows int",
		},
		{
			name:  "int8 from out of range float",
			path:  "i8",
			value: 200.0,
			err:   "value 200 overflows int8",
		},
		{
			name:  "int64 from float 2^63",
			path:  "i64",
			value: math.Exp2(63),
			err:   "overflows int64",
		},
		{
			name:     "int64 from float -2^63",
			path:     "i64",
			value:    -math.Exp2(63),
			expected: int64(math.MinInt64),
		},
		{
			name:  "uint from negative float",
			path:  "u",
			value: -1.0,
			err:   "value -1 overflows uint",
		},
		{
			name:  "uint8 from out of range float",
			path:  "u8",
			value: 256.0,
			err:   "value 256 overflows uint8",
		},
		{
			name:  "uint64 from float 2^64",
			path:  "u64",
			value: math.Exp2(64),
			err:   "overflows uint64",
		},
		{
			name:     "uint from float32 integral",
			path:     "u",
			value:    float32(7),
			expected: uint(7),
		},

		// int <- int/uint
		{
			name:     "int32 from fitting int64",
			path:     "i32",
			value:    int64(5),
			expected: int32(5),
		},
		{
			name:  "int32 from overflowing int64",
			path:  "i32",
			value: int64(1 << 40),
			err:   "value 1099511627776 overflows int32",
		},
		{
			name:  "int8 from negative overflowing int",
			path:  "i8",
			value: -129,
			err:   "value -129 overflows int8",
		},
		{
			name:  "int64 from uint64 above MaxInt64",
			path:  "i64",
			value: uint64(math.MaxUint64),
			err:   "value 18446744073709551615 overflows int64",
		},
		{
			name:     "int64 from uint64 MaxInt64",
			path:     "i64",
			value:    uint64(math.MaxInt64),
			expected: int64(math.MaxInt64),
		},
		{
			name:  "int8 from overflowing uint",
			path:  "i8",
			value: uint(128),
			err:   "value 128 overflows int8",
		},

		// uint <- int/uint
		{
			name:  "uint from negative int",
			path:  "u",
			value: -1,
			err:   "value -1 overflows uint",
		},
		{
			name:  "uint8 from overflowing int",
			path:  "u8",
			value: 256,
			err:   "value 256 overflows uint8",
		},
		{
			name:     "uint8 from fitting int",
			path:     "u8",
			value:    255,
			expected: uint8(255),
		},
		{
			name:  "uint8 from overflowing uint64",
			path:  "u8",
			value: uint64(300),
			err:   "value 300 overflows uint8",
		},
		{
			name:     "uint64 from large uint64",
			path:     "u64",
			value:    uint64(math.MaxUint64),
			expected: uint64(math.MaxUint64),
		},

		// float <- int/uint
		{
			name:     "float64 from small int",
			path:     "f64",
			value:    1 << 53,
			expected: float64(1 << 53),
		},
		{
			name:     "float64 from int64 above 2^53 rounds",
			path:     "f64",
			value:    int64(1<<53 + 1),
			expected: float64(1 << 53),
		},
		{
			name:     "float64 from MaxInt64 rounds",
			path:     "f64",
			value:    int64(math.MaxInt64),
			expected: float64(math.MaxInt64),
		},
		{
			name:     "float64 from uint64 above 2^53 rounds",
			path:     "f64",
			value:    uint64(1<<53 + 1),
			expected: float64(1 << 53),
		},
		{
			name:     "float64 from MaxUint64 rounds",
			path:     "f64",
			value:    uint64(math.MaxUint64),
			expected: float64(math.MaxUint64),
		},
		{
			name:     "float32 from int above 2^24 rounds",
			path:     "f32",
			value:    1<<24 + 1,
			expected: float32(1 << 24),
		},
		{
			name:     "float32 from int 2^24",
			path:     "f32",
			value:    1 << 24,
			expected: float32(1 << 24),
		},

		// float <- float
		{
			name:     "float32 from float64",
			path:     "f32",
			value:    1.5,
			expected: float32(1.5),
		},
		{
			name:     "float32 from rounding float64",
			path:     "f32",
			value:    1.1,
			expected: float32(1.1),
		},
		{
			name:     "float32 from overflowing float64",
			path:     "f32",
			value:    1e300,
			expected: float32(math.Inf(1)),
		},
		{
			name:     "float64 from float32",
			path:     "f64",
			value:    float32(0.5),
			expected: 0.5,
		},

		// pointer targets
		{
			name:     "ptr int from integral float",
			path:     "pi",
			value:    2.0,
			expected: 2,
		},
		{
			name:  "ptr int from fractional float",
			path:  "pi",
			value: 1.9,
			err:   "cannot set 1.9 to int: precision loss",
		},
		{
			name:  "ptr int32 from overflowing int64",
			path:  "pi32",
			value: int64(1 << 40),
			err:   "value 1099511627776 overflows int32",
		},
		{
			name:  "ptr uint from negative int",
			path:  "pu",
			value: -1,
			err:   "value -1 overflows uint",
		},
		{
			name:     "ptr float32 from overflowing float64",
			path:     "pf32",
			value:    1e300,
			expected: float32(math.Inf(1)),
		},
		{
			name:     "ptr float64 from large int64 rounds",
			path:     "pf64",
			value:    int64(1<<53 + 1),
			expected: float64(1 << 53),
		},
		{
			name:     "ptr float64 from small int",
			path:     "pf64",
			value:    3,
			expected: 3.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := &numericTarget{}
			err := structaccess.SetByString(v, tc.path, tc.value)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				assert.Equal(t, &numericTarget{}, v, "failed set must leave the struct untouched")
				return
			}
			require.NoError(t, err)
			got, err := structaccess.GetByString(v, tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
