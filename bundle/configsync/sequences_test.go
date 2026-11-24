package configsync

import (
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackedOp struct {
	operation OperationType
	// index is the element's position in the original list; -1 appends a new element.
	index int
	value string
}

// applyTracked places each operation with an indexTracker and applies it to list the
// way the YAML patcher would, so the result shows which elements each index reached.
func applyTracked(t *testing.T, list []string, ops []trackedOp) []string {
	tracker := newIndexTracker()
	parent := structpath.NewPatternStringKey(nil, "tasks")
	result := slices.Clone(list)
	for _, op := range ops {
		path := structpath.NewPatternBracketStar(parent)
		if op.index >= 0 {
			path = structpath.NewPatternIndex(parent, op.index)
		}
		placed := tracker.place("", path, op.operation)
		index, ok := placed.Index()
		switch {
		case op.operation == OperationRemove:
			require.True(t, ok)
			result = slices.Delete(result, index, index+1)
		case ok:
			result = slices.Insert(result, index, op.value)
		default:
			result = append(result, op.value)
		}
	}
	return result
}

func TestIndexTrackerRemovalsOutOfOrder(t *testing.T) {
	// Removals arrive in key order (a, b, c, d) while the list has d before c.
	list := []string{"keep", "a", "b", "d", "c"}
	got := applyTracked(t, list, []trackedOp{
		{operation: OperationRemove, index: 1},
		{operation: OperationRemove, index: 2},
		{operation: OperationRemove, index: 4},
		{operation: OperationRemove, index: 3},
	})
	assert.Equal(t, []string{"keep"}, got)
}

func TestIndexTrackerRemovalsDescending(t *testing.T) {
	list := []string{"a", "b", "c", "keep"}
	got := applyTracked(t, list, []trackedOp{
		{operation: OperationRemove, index: 2},
		{operation: OperationRemove, index: 1},
		{operation: OperationRemove, index: 0},
	})
	assert.Equal(t, []string{"keep"}, got)
}
