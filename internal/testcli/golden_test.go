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
