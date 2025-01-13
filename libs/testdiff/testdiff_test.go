package testdiff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	assert.Equal(t, "", UnifiedDiff("a", "b", "", ""))
	assert.Equal(t, "", UnifiedDiff("a", "b", "abc", "abc"))
	assert.Equal(t, "--- a\n+++ b\n@@ -1 +1,2 @@\n abc\n+123\n", UnifiedDiff("a", "b", "abc\n", "abc\n123\n"))
}

func TestMatchesPrefixes(t *testing.T) {
	assert.False(t, matchesPrefixes([]string{}, ""))
	assert.False(t, matchesPrefixes([]string{"/hello", "/hello/world"}, ""))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/hello"))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/a/b/c"))
}
