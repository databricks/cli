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
		name            string
		nodes           []string
		seen            []string
		seenSorted      []string
		edges           []edge
		stops           map[string]bool // node -> false to indicate failure
		pools           []int           // optional override of pools to run
		cycle           string
		failedFrom      map[string]string   // node -> expected failedFrom
		failedFromOneOf map[string][]string // node -> any of these failedFrom values acceptable
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
		{
			name:  "downstream runs with failed dependency",
			edges: []edge{{"A", "B", "A->B"}, {"B", "C", "B->C"}},
			seen:  []string{"A", "B", "C"},
			stops: map[string]bool{"B": false},
			failedFrom: map[string]string{
				"C": "B",
			},
		},
		{
			name:       "multiple failures propagate to same node (any one reported)",
			edges:      []edge{{"A", "D", "A->D"}, {"B", "D", "B->D"}},
			seenSorted: []string{"A", "B", "D"},
			stops:      map[string]bool{"A": false, "B": false},
			pools:      []int{1},
			failedFromOneOf: map[string][]string{
				"D": {"A", "B"},
			},
		},
	}

	for _, tc := range tests {
		poolsToRun := pools
		if len(tc.pools) > 0 {
			poolsToRun = tc.pools
		}
		for _, p := range poolsToRun {
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
						g.Run(p, func(n stringWrapper, failed *stringWrapper) bool {
							innerCalled += 1
							return true
						})
					})
					require.Zero(t, innerCalled)
					return
				}
				require.NoError(t, err)

				var mu sync.Mutex
				var seen []string
				failedFrom := map[string]*string{}
				g.Run(p, func(n stringWrapper, failed *stringWrapper) bool {
					mu.Lock()
					seen = append(seen, n.Value)
					if failed != nil {
						v := failed.Value
						failedFrom[n.Value] = &v
					} else {
						failedFrom[n.Value] = nil
					}
					mu.Unlock()
					if stop, exists := tc.stops[n.Value]; exists {
						return stop
					}
					return true // success by default
				})

				if tc.seen != nil {
					assert.Equal(t, tc.seen, seen)
				} else if tc.seenSorted != nil {
					sort.Strings(seen)
					assert.Equal(t, tc.seenSorted, seen)
				} else {
					assert.Empty(t, seen)
				}

				for node, want := range tc.failedFrom {
					gotPtr := failedFrom[node]
					if assert.NotNil(t, gotPtr, "expected failedFrom for %s", node) {
						assert.Equal(t, want, *gotPtr)
					}
				}
				for node, oneOf := range tc.failedFromOneOf {
					gotPtr := failedFrom[node]
					if assert.NotNil(t, gotPtr, "expected failedFrom for %s", node) {
						found := false
						for _, candidate := range oneOf {
							if *gotPtr == candidate {
								found = true
								break
							}
						}
						assert.True(t, found, "failedFrom for %s not in %v, got %v", node, oneOf, *gotPtr)
					}
				}
			})
		}
	}
}
