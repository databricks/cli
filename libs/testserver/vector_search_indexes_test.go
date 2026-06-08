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
			name: "alias folded to canonical type",
			in:   `{"id":"int","vector":"array<float>"}`,
			want: `{"id":"integer","vector":"array<float>"}`,
		},
		{
			name: "all integer-family aliases",
			in:   `{"a":"bigint","b":"smallint","c":"tinyint"}`,
			want: `{"a":"long","b":"short","c":"byte"}`,
		},
		{
			name: "array element type is normalized",
			in:   `{"tags":"array<int>"}`,
			want: `{"tags":"array<integer>"}`,
		},
		{
			name: "canonical types pass through and keys are sorted",
			in:   `{"y":"float","x":"string","z":"integer"}`,
			want: `{"x":"string","y":"float","z":"integer"}`,
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
