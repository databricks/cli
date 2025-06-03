package structtag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleTagMethods(t *testing.T) {
	tests := []struct {
		tag          string
		isReadOnly   bool
		isInternal   bool
		isDeprecated bool
	}{
		// only one annotation.
		{"readonly", true, false, false},
		{"internal", false, true, false},
		{"deprecated", false, false, true},

		// multiple annotations.
		{"readonly,internal", true, true, false},
		{"readonly,deprecated", true, false, true},
		{"internal,deprecated", false, true, true},
		{"readonly,internal,deprecated", true, true, true},

		// unknown annotations are ignored.
		{"something", false, false, false},
		{"-", false, false, false},
		{"name,string", false, false, false},
		{"weird,whatever,readonly,foo", true, false, false},
	}

	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			tag := BundleTag(test.tag)

			assert.Equal(t, test.isReadOnly, tag.ReadOnly())
			assert.Equal(t, test.isInternal, tag.Internal())
			assert.Equal(t, test.isDeprecated, tag.Deprecated())
		})
	}
}
