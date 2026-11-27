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
		case op.operation == OperationReplace:
			require.True(t, ok)
			result[index] = op.value
		case ok:
			result = slices.Insert(result, index, op.value)
		default:
			result = append(result, op.value)
		}
	}
	return result
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

func TestIndexTrackerAddReusesFreedSlot(t *testing.T) {
	list := []string{"keep", "a", "b"}
	got := applyTracked(t, list, []trackedOp{
		{operation: OperationRemove, index: 1},
		{operation: OperationRemove, index: 2},
		{operation: OperationAdd, index: -1, value: "x"},
	})
	assert.Equal(t, []string{"keep", "x"}, got)
}

func TestIndexTrackerAddsReuseFreedSlotsInOrder(t *testing.T) {
	list := []string{"a", "b", "c"}
	got := applyTracked(t, list, []trackedOp{
		{operation: OperationRemove, index: 0},
		{operation: OperationRemove, index: 1},
		{operation: OperationAdd, index: -1, value: "n1"},
		{operation: OperationAdd, index: -1, value: "n2"},
	})
	assert.Equal(t, []string{"n1", "n2", "c"}, got)
}

func TestIndexTrackerReplaceAfterAddIntoFreedSlot(t *testing.T) {
	// The replace must reach z, which the new element m pushed forward, not m.
	list := []string{"a", "z"}
	got := applyTracked(t, list, []trackedOp{
		{operation: OperationRemove, index: 0},
		{operation: OperationAdd, index: -1, value: "m"},
		{operation: OperationReplace, index: 1, value: "z2"},
	})
	assert.Equal(t, []string{"m", "z2"}, got)
}
