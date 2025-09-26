package structaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	CustomString string
	CustomInt    int
)

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
		errorMsg string
	}{
		// Basic scalar types
		{
			name:     "string",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "int",
			value:    42,
			expected: "42",
		},
		{
			name:     "float64",
			value:    3.14,
			expected: "3.14",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},

		// Custom types (type aliases)
		{
			name:     "custom string",
			value:    CustomString("world"),
			expected: "world",
		},
		{
			name:     "custom int",
			value:    CustomInt(100),
			expected: "100",
		},

		// Pointers
		{
			name:     "string pointer",
			value:    stringPtr("test"),
			expected: "test",
		},
		{
			name:     "int pointer",
			value:    intPtr(123),
			expected: "123",
		},
		{
			name:     "float64 pointer",
			value:    float64Ptr(2.5),
			expected: "2.5",
		},
		{
			name:     "bool pointer",
			value:    boolPtr(true),
			expected: "true",
		},
		{
			name:     "nil string pointer",
			value:    (*string)(nil),
			expected: "",
		},
		{
			name:     "nil int pointer",
			value:    (*int)(nil),
			expected: "",
		},
		{
			name:     "nil float64 pointer",
			value:    (*float64)(nil),
			expected: "",
		},
		{
			name:     "nil bool pointer",
			value:    (*bool)(nil),
			expected: "",
		},
		{
			name:  "nil",
			value: nil,
		},

		// Unsupported types - should return errors
		{
			name:     "slice",
			value:    []int{1, 2, 3},
			errorMsg: "unsupported type for string conversion: []int",
		},
		{
			name:     "map",
			value:    map[string]int{"a": 1},
			errorMsg: "unsupported type for string conversion: map[string]int",
		},
		{
			name:     "struct",
			value:    struct{ X int }{X: 1},
			errorMsg: "unsupported type for string conversion: struct { X int }",
		},
		{
			name:     "pointer to struct",
			value:    &struct{ X int }{X: 1},
			errorMsg: "unsupported type for string conversion: *struct { X int }",
		},
		{
			name:     "channel",
			value:    make(chan int),
			errorMsg: "unsupported type for string conversion: chan int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToString(tt.value)
			if tt.errorMsg != "" {
				require.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Equal(t, "", result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}
