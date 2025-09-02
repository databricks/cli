package structtag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleTagMethods(t *testing.T) {
	tests := []struct {
		tag        string
		isReadOnly bool
		isInternal bool
	}{
		// only one annotation.
		{tag: "readonly", isReadOnly: true},
		{tag: "internal", isInternal: true},

		// multiple annotations.
		{tag: "readonly,internal", isReadOnly: true, isInternal: true},

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
		})
	}
}
