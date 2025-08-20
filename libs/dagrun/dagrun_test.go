package dagrun

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type edge struct{ from, to, name string }

type stringWrapper struct {
	Value string
}

func (s stringWrapper) String() string {
	return s.Value
}

func TestRun_VariousGraphsAndPools(t *testing.T) {
	pools := []int{1, 2, 3, 4}

	tests := []struct {
		name       string
		nodes      []string
		seen       []string
		seenSorted []string
		edges      []edge
		cycle      string
	}{
		// disconnected graphs
		{
			name: "empty graph",
		},
		{
			name:  "one node",
			nodes: []string{"A"},
			seen:  []string{"A"},
		},
		{
			name:       "two nodes",
			nodes:      []string{"A", "B"},
			seenSorted: []string{"A", "B"},
		},
		{
			name:       "three nodes",
			nodes:      []string{"A", "B", "C"},
			seenSorted: []string{"A", "B", "C"},
		},
		{
			name: "simple DAG",
			edges: []edge{
				{"A", "B", "A->B"},
				{"B", "C", "B->C"},
			},
			seen: []string{"A", "B", "C"},
		},
		{
			name: "one-node cycle",
			edges: []edge{
				{"A", "A", "${A.id}"},
			},
			cycle: "cycle detected: A refers to itself via ${A.id}",
		},
		{
			name: "two-node cycle",
			edges: []edge{
				{"A", "B", "${A.id}"},
				{"B", "A", "${B.id}"},
			},
			cycle: "cycle detected: A refers to B via ${A.id} which refers to A via ${B.id}",
		},
		{
			name: "three-node cycle",
			edges: []edge{
				{"X", "Y", "e1"},
				{"Y", "Z", "e2"},
				{"Z", "X", "e3"},
			},
			cycle: "cycle detected: X refers to Y via e1 Y refers to Z via e2 which refers to X via e3",
		},
	}

	for _, tc := range tests {
		for _, p := range pools {
			t.Run(tc.name+fmt.Sprintf(" pool=%d", p), func(t *testing.T) {
				g := NewGraph[stringWrapper]()
				for _, n := range tc.nodes {
					g.AddNode(stringWrapper{n})
				}
				for _, e := range tc.edges {
					g.AddDirectedEdge(stringWrapper{e.from}, stringWrapper{e.to}, e.name)
				}

				err := g.DetectCycle()
				if tc.cycle != "" {
					require.Error(t, err, "expected cycle, got none")
					require.Equal(t, tc.cycle, err.Error())
					innerCalled := 0
					require.Panics(t, func() {
						g.Run(p, func(n stringWrapper) {
							innerCalled += 1
						})
					})
					require.Zero(t, innerCalled)
					return
				}
				require.NoError(t, err)

				var mu sync.Mutex
				var seen []string
				g.Run(p, func(n stringWrapper) {
					mu.Lock()
					seen = append(seen, n.Value)
					mu.Unlock()
				})

				if tc.seen != nil {
					assert.Equal(t, tc.seen, seen)
				} else if tc.seenSorted != nil {
					sort.Strings(seen)
					assert.Equal(t, tc.seenSorted, seen)
				} else {
					assert.Empty(t, seen)
				}
			})
		}
	}
}
