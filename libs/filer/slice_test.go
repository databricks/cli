package filer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceWithout(t *testing.T) {
	assert.Equal(t, []int{}, sliceWithout([]int{}, 0))
	assert.Equal(t, []int{1, 2, 3}, sliceWithout([]int{1, 2, 3}, 4))
	assert.Equal(t, []int{2, 3}, sliceWithout([]int{1, 2, 3}, 1))
	assert.Equal(t, []int{1, 3}, sliceWithout([]int{1, 2, 3}, 2))
	assert.Equal(t, []int{1, 2}, sliceWithout([]int{1, 2, 3}, 3))
}

func TestSliceWithoutReturnsClone(t *testing.T) {
	ints := []int{1, 2, 3}
	assert.Equal(t, []int{2, 3}, sliceWithout(ints, 1))
	assert.Equal(t, []int{1, 2, 3}, ints)
}
