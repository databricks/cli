package cmdio

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalJSONWithoutColorMatchesMarshalIndent(t *testing.T) {
	tests := []any{
		map[string]any{"name": "alice", "n": 1, "ok": true, "tags": []any{"x", "y"}, "v": nil},
		[]any{1, 2.5, -3, 1e10, "s", true, false, nil},
		map[string]any{"nested": map[string]any{"a": []any{1, 2, 3}}},
		"plain string",
		42,
	}
	for _, v := range tests {
		want, err := json.MarshalIndent(v, "", "  ")
		require.NoError(t, err)
		got, err := marshalJSON(v, false)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got))
	}
}

func TestColorizeJSONTokens(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "string value",
			in:   `"hello"`,
			want: ansiGreen + `"hello"` + ansiReset,
		},
		{
			name: "string with embedded colon",
			in:   `"a:b"`,
			want: ansiGreen + `"a:b"` + ansiReset,
		},
		{
			name: "string with escapes",
			in:   `"a\"b\\c"`,
			want: ansiGreen + `"a\"b\\c"` + ansiReset,
		},
		{
			name: "true",
			in:   `true`,
			want: ansiBoldGreen + `true` + ansiReset,
		},
		{
			name: "false",
			in:   `false`,
			want: ansiRed + `false` + ansiReset,
		},
		{
			name: "null",
			in:   `null`,
			want: ansiMagenta + `null` + ansiReset,
		},
		{
			name: "negative number",
			in:   `-12`,
			want: ansiCyan + `-12` + ansiReset,
		},
		{
			name: "decimal",
			in:   `3.14`,
			want: ansiCyan + `3.14` + ansiReset,
		},
		{
			name: "exponent",
			in:   `1.5e+10`,
			want: ansiCyan + `1.5e+10` + ansiReset,
		},
		{
			name: "object key bold blue, value green",
			in:   `{"k": "v"}`,
			want: `{` + ansiBoldBlue + `"k"` + ansiReset + `: ` + ansiGreen + `"v"` + ansiReset + `}`,
		},
		{
			name: "punctuation passes through",
			in:   "{\n  \"a\": [1, 2]\n}",
			want: "{\n  " + ansiBoldBlue + `"a"` + ansiReset + ": [" + ansiCyan + `1` + ansiReset + ", " + ansiCyan + `2` + ansiReset + "]\n}",
		},
		{
			name: "string value containing literal-looking content",
			in:   `"true"`,
			want: ansiGreen + `"true"` + ansiReset,
		},
		{
			name: "string value containing number-looking content",
			in:   `"-1.5e+10"`,
			want: ansiGreen + `"-1.5e+10"` + ansiReset,
		},
		{
			name: "string value containing JSON-in-a-string",
			in:   `"{\"k\": 1}"`,
			want: ansiGreen + `"{\"k\": 1}"` + ansiReset,
		},
		{
			name: "string value with unicode escape",
			in:   `"café"`,
			want: ansiGreen + `"café"` + ansiReset,
		},
		{
			name: "string value containing only an escaped backslash",
			in:   `"\\"`,
			want: ansiGreen + `"\\"` + ansiReset,
		},
		{
			name: "string value containing only an escaped quote",
			in:   `"\""`,
			want: ansiGreen + `"\""` + ansiReset,
		},
		{
			name: "empty string value",
			in:   `""`,
			want: ansiGreen + `""` + ansiReset,
		},
		{
			name: "empty object",
			in:   `{}`,
			want: `{}`,
		},
		{
			name: "empty array",
			in:   `[]`,
			want: `[]`,
		},
		{
			name: "single zero",
			in:   `0`,
			want: ansiCyan + `0` + ansiReset,
		},
		{
			name: "packed mixed array",
			in:   `[true,false,null,1,"s"]`,
			want: `[` + ansiBoldGreen + `true` + ansiReset + `,` + ansiRed + `false` + ansiReset + `,` + ansiMagenta + `null` + ansiReset + `,` + ansiCyan + `1` + ansiReset + `,` + ansiGreen + `"s"` + ansiReset + `]`,
		},
		{
			name: "string containing colon is a value, not a key",
			in:   `["a:b", "c"]`,
			want: `[` + ansiGreen + `"a:b"` + ansiReset + `, ` + ansiGreen + `"c"` + ansiReset + `]`,
		},
		{
			name: "whitespace before closing brace",
			in:   "{\"k\": \"v\"\n}",
			want: "{" + ansiBoldBlue + `"k"` + ansiReset + ": " + ansiGreen + `"v"` + ansiReset + "\n}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorizeJSON([]byte(tt.in))
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestColorizeJSONNested(t *testing.T) {
	v := map[string]any{
		"name": "alice",
		"age":  30,
		"ok":   true,
		"v":    nil,
		"kids": []any{"bob", "carol"},
	}
	b, err := marshalJSON(v, true)
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, ansiGreen+`"alice"`+ansiReset)
	assert.Contains(t, s, ansiCyan+`30`+ansiReset)
	assert.Contains(t, s, ansiBoldGreen+`true`+ansiReset)
	assert.Contains(t, s, ansiMagenta+`null`+ansiReset)
	assert.Contains(t, s, ansiGreen+`"bob"`+ansiReset)
	assert.Contains(t, s, ansiBoldBlue+`"name"`+ansiReset)
	assert.Contains(t, s, ansiBoldBlue+`"age"`+ansiReset)
	assert.NotContains(t, s, ansiGreen+`"name"`+ansiReset)
	assert.NotContains(t, s, ansiGreen+`"age"`+ansiReset)
}

func TestColorizeJSONRoundTrip(t *testing.T) {
	inputs := []any{
		nil,
		true,
		false,
		0,
		-1,
		3.14,
		"",
		"plain",
		`with "quotes" and \ backslash`,
		"with\ttab\nand\nnewline",
		"café",
		map[string]any{},
		[]any{},
		[]any{1, 2, 3},
		map[string]any{"a": 1, "b": "two", "c": nil, "d": true, "e": false},
		map[string]any{"k:v": "a:b", "true": "false", "null": "123"},
		map[string]any{"nested": map[string]any{"x": []any{nil, true, "s", -2.5e10}}},
	}
	for _, v := range inputs {
		want, err := json.MarshalIndent(v, "", "  ")
		require.NoError(t, err)
		got := colorizeJSON(want)
		assert.Equal(t, string(want), stripANSI(string(got)))
	}
}
