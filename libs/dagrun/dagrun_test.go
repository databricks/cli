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
		name        string
		nodes       []string
		seen        []string
		seen_sorted []string
		edges       []edge
		cycle       bool
		msg         string
	}{
		// disconnected graphs
		{
			name:  "one node",
			nodes: []string{"A"},
			seen:  []string{"A"},
		},
		{
			name:        "two nodes",
			nodes:       []string{"A", "B"},
			seen_sorted: []string{"A", "B"},
		},
		{
			name:        "three nodes",
			nodes:       []string{"A", "B", "C"},
			seen_sorted: []string{"A", "B", "C"},
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
			name: "two-node cycle",
			edges: []edge{
				{"A", "B", "${A.id}"},
				{"B", "A", "${B.id}"},
			},
			cycle: true,
			msg:   "cycle detected: A refers to B via ${A.id} which refers to A via ${B.id}.",
		},
		{
			name: "three-node cycle",
			edges: []edge{
				{"X", "Y", "e1"},
				{"Y", "Z", "e2"},
				{"Z", "X", "e3"},
			},
			cycle: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		for _, p := range pools {
			t.Run(tc.name+fmt.Sprintf(" pool=%d", p), func(t *testing.T) {
				g := NewGraph[stringWrapper]()
				for _, n := range tc.nodes {
					g.AddNode(stringWrapper{n})
				}
				for _, e := range tc.edges {
					_ = g.AddDirectedEdge(stringWrapper{e.from}, stringWrapper{e.to}, e.name)
				}

				var mu sync.Mutex
				var seen []string
				err := g.Run(p, func(n stringWrapper) {
					mu.Lock()
					seen = append(seen, n.Value)
					mu.Unlock()
				})

				if tc.cycle {
					if err == nil {
						t.Fatalf("expected cycle, got none")
					}
					if tc.msg != "" && err.Error() != tc.msg {
						t.Fatalf("wrong msg:\n got %q\nwant %q", err, tc.msg)
					}
				} else {
					require.NoError(t, err)
				}

				if tc.seen != nil {
					assert.Equal(t, tc.seen, seen)
				} else if tc.seen_sorted != nil {
					sort.Strings(seen)
					assert.Equal(t, tc.seen_sorted, seen)
				} else {
					assert.Empty(t, seen)
				}
			})
		}
	}
}
