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
		{tag: "readonly", isReadOnly: true},
		{tag: "internal", isInternal: true},
		{tag: "deprecated", isDeprecated: true},

		// multiple annotations.
		{tag: "readonly,internal", isReadOnly: true, isInternal: true},
		{tag: "readonly,deprecated", isReadOnly: true, isDeprecated: true},
		{tag: "internal,deprecated", isInternal: true, isDeprecated: true},
		{tag: "readonly,internal,deprecated", isReadOnly: true, isInternal: true, isDeprecated: true},

		// unknown annotations are ignored.
		{tag: "something"},
		{tag: "-"},
		{tag: "name,string"},
		{tag: "weird,whatever,readonly,foo", isReadOnly: true},
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
