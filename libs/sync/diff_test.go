package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffGroupedMkdir(t *testing.T) {
	d := diff{
		mkdir: []string{
			"foo",
			"foo/bar",
			"foo/bar/baz1",
			"foo/bar/baz2",
			"foo1",
			"a/b",
			"a/b/c/d/e/f",
		},
	}

	// Expect only leaf directories to be included.
	out := d.groupedMkdir()
	assert.ElementsMatch(t, []string{
		"foo/bar/baz1",
		"foo/bar/baz2",
		"foo1",
		"a/b/c/d/e/f",
	}, out[0])
}

func TestDiffGroupedRmdir(t *testing.T) {
	d := diff{
		rmdir: []string{
			"a/b/c/d/e/f",
			"a/b/c/d/e",
			"a/b/c/d",
			"a/b/c",
			"a/b/e/f/g/h",
			"a/b/e/f/g",
			"a/b/e/f",
			"a/b/e",
			"a/b",
		},
	}

	out := d.groupedRmdir()
	assert.Len(t, out, 5)
	assert.ElementsMatch(t, []string{"a/b/c/d/e/f", "a/b/e/f/g/h"}, out[0])
	assert.ElementsMatch(t, []string{"a/b/c/d/e", "a/b/e/f/g"}, out[1])
	assert.ElementsMatch(t, []string{"a/b/c/d", "a/b/e/f"}, out[2])
	assert.ElementsMatch(t, []string{"a/b/c", "a/b/e"}, out[3])
	assert.ElementsMatch(t, []string{"a/b"}, out[4])
}

func TestDiffGroupedRmdirWithLeafsOnly(t *testing.T) {
	d := diff{
		rmdir: []string{
			"foo/bar/baz1",
			"foo/bar1",
			"foo/bar/baz2",
			"foo/bar2",
			"foo1",
			"foo2",
		},
	}

	// Expect all directories to be included.
	out := d.groupedRmdir()
	assert.Len(t, out, 1)
	assert.ElementsMatch(t, d.rmdir, out[0])
}
