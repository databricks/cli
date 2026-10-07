package dynvar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPureReferenceToStructPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "simple reference",
			input:    "${resources.jobs.foo.id}",
			expected: "resources.jobs.foo.id",
			ok:       true,
		},
		{
			name:     "simple reference",
			input:    "${resources.jobs.foo.tasks[1].env.key}",
			expected: "resources.jobs.foo.tasks[1].env.key",
			ok:       true,
		},
		{
			name:  "complex nested reference",
			input: "${var.resources.jobs['my_job'].tasks[0]}",
			// we use regex from dyn module which only support integers inside brackets:
			// expected: "resources.jobs['my_job'].tasks[0]",
		},
		{
			name:  "not a pure reference",
			input: "prefix_${var.field}",
		},
		{
			name:  "not a variable reference",
			input: "plain_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathNode, ok := PureReferenceToStructPath(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.NotNil(t, pathNode)
				assert.Equal(t, tt.expected, pathNode.String())
			} else {
				assert.Nil(t, pathNode)
			}
		})
	}
}
