package variable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariable_IsComplexValued(t *testing.T) {
	tests := []struct {
		name string
		v    *Variable
		want bool
	}{
		{
			name: "map",
			v: &Variable{
				Value: map[string]any{
					"foo": "bar",
				},
			},
			want: true,
		},
		{
			name: "slice",
			v: &Variable{
				Value: []any{1, 2, 3},
			},
			want: true,
		},
		{
			name: "struct",
			v: &Variable{
				Value: struct{ Foo string }{Foo: "bar"},
			},
			want: true,
		},
		{
			name: "non-complex valued",
			v: &Variable{
				Value: "foo",
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.v.IsComplexValued())
		})
	}
}
