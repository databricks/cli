package tests

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A field's values form a complete digraph, so one Eulerian circuit covers every ordered
// pair exactly once and no second chain is ever needed. This pins that down: full
// coverage, no repeats, and every step starting where the last one ended.
func TestTransitionsCoverEveryPairInOneChain(t *testing.T) {
	for n := 2; n <= 6; n++ {
		values := make([]dyn.Value, 0, n-1)
		for i := 1; i < n; i++ {
			values = append(values, dyn.V(i))
		}
		f := field{path: "some.field", values: values} //exhaustruct:ignore

		got := f.transitions()

		// n-1 explicit values plus the implicit absent.
		want := n * (n - 1)
		require.Len(t, got, want, "n=%d", n)

		seen := map[string]bool{}
		for i, tr := range got {
			pair := tr.label()
			assert.False(t, seen[pair], "pair %s repeated at %d (n=%d)", pair, i, n)
			seen[pair] = true
			assert.NotEqual(t, valueLabel(tr.from), valueLabel(tr.to), "self-loop at %d", i)
			if i > 0 {
				assert.Equal(t, valueLabel(got[i-1].to), valueLabel(tr.from),
					"step %d does not start where step %d ended (n=%d)", i, i-1, n)
			}
		}
		assert.Len(t, seen, want, "n=%d", n)
	}
}

// A required field has no absent value, so its chain is one vertex smaller.
func TestTransitionsSkipAbsentForRequiredField(t *testing.T) {
	f := field{path: "name", values: []dyn.Value{dyn.V("a"), dyn.V("b")}, required: true} //exhaustruct:ignore

	got := f.transitions()

	require.Len(t, got, 2)
	for _, tr := range got {
		assert.NotEqual(t, "absent", valueLabel(tr.from))
		assert.NotEqual(t, "absent", valueLabel(tr.to))
	}
}
