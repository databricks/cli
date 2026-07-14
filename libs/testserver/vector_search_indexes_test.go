package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSchemaJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "user-facing type stored as Spark type name",
			in:   `{"id":"integer","vector":"array<float>"}`,
			want: `{"id":"int","vector":"array<float>"}`,
		},
		{
			name: "all integer-family names",
			in:   `{"a":"long","b":"short","c":"byte"}`,
			want: `{"a":"bigint","b":"smallint","c":"tinyint"}`,
		},
		{
			name: "array element type is mapped",
			in:   `{"tags":"array<integer>"}`,
			want: `{"tags":"array<int>"}`,
		},
		{
			name: "matching spellings pass through and keys are sorted",
			in:   `{"y":"float","x":"string","z":"int"}`,
			want: `{"x":"string","y":"float","z":"int"}`,
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
		{
			name: "non-object input is returned unchanged",
			in:   "not json",
			want: "not json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeSchemaJSON(tt.in))
		})
	}
}
