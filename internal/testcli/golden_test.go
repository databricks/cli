package testcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	input := []string{"a", "bc", "cd"}
	stableSortReverseLength(input)
	assert.Equal(t, []string{"bc", "cd", "a"}, input)
}

func TestMatchesPrefixes(t *testing.T) {
	assert.False(t, matchesPrefixes([]string{}, ""))
	assert.False(t, matchesPrefixes([]string{"/hello", "/hello/world"}, ""))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/hello"))
	assert.True(t, matchesPrefixes([]string{"/hello", "/a/b"}, "/a/b/c"))
}
