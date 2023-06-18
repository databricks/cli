package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func TestMakeDirSet(t *testing.T) {
	assert.ElementsMatch(t,
		[]string{
			"a",
			"a/b",
			"a/b/c",
			"a/b/d",
			"a/e",
			"b",
		},
		maps.Keys(
			MakeDirSet([]string{
				"./a/b/c/file1",
				"./a/b/c/file2",
				"./a/b/d/file",
				"./a/e/file",
				"b/file",
			}),
		),
	)
}

func TestDirSetRemove(t *testing.T) {
	a := MakeDirSet([]string{"./a/b/c/file1"})
	b := MakeDirSet([]string{"./a/b/d/file2"})
	assert.ElementsMatch(t, []string{"a/b/c"}, a.Remove(b).Slice())
	assert.ElementsMatch(t, []string{"a/b/d"}, b.Remove(a).Slice())
}
