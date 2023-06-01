package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffGroupDeletesByNestingLevel(t *testing.T) {
	d := diff{
		delete: []string{
			"foo/bar/baz1",
			"foo/bar1",
			"foo/bar/baz2",
			"foo/bar2",
			"foo1",
			"foo2",
		},
	}

	expected := [][]string{
		{"foo/bar/baz1", "foo/bar/baz2"},
		{"foo/bar1", "foo/bar2"},
		{"foo1", "foo2"},
	}

	assert.Equal(t, expected, d.GroupDeletesByNestingLevel())
}
