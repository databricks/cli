package golden

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	assert.Equal(t, "", UnifiedDiff("a", "b", "", ""))
	assert.Equal(t, "", UnifiedDiff("a", "b", "abc", "abc"))
	assert.Equal(t, "+123", UnifiedDiff("a", "b", "abc", "abc\123"))
}

func TestMatchesPrefixes(t *testing.T) {
	assert.False(t, matchesPrefixes([]string{}, ""))
	assert.False(t, matchesPrefixes([]string{"/hello", "/hello/world"}, ""))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/hello"))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/a/b/c"))
}
