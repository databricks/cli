package direct

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestDynPathToStructPath(t *testing.T) {
	tests := []struct {
		path     dyn.Path
		expected string
	}{
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Key("bar")),
			expected: "foo.bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar")),
			expected: "foo[1].bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("configuration"), dyn.Key("europris.swipe.egress_streaming_schema")),
			expected: "configuration['europris.swipe.egress_streaming_schema']",
		},
		{
			path:     dyn.NewPath(dyn.Key("tags"), dyn.Key("it's.here")),
			expected: "tags['it''s.here']",
		},
	}

	for _, tc := range tests {
		node := dynPathToStructPath(tc.path)
		assert.Equal(t, tc.expected, node.String())
	}
}
