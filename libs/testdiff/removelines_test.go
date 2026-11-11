package testdiff

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveLines(t *testing.T) {
	kept, removed := RemoveLines("a\nNoise: 1\nb\nNoise: 2\n", regexp.MustCompile(`^Noise: `))
	assert.Equal(t, "a\nb\n", kept)
	assert.Equal(t, []string{"Noise: 1", "Noise: 2"}, removed)
}

func TestRemoveLinesNoMatch(t *testing.T) {
	kept, removed := RemoveLines("a\nb", regexp.MustCompile(`^Noise: `))
	assert.Equal(t, "a\nb", kept)
	assert.Empty(t, removed)
}
