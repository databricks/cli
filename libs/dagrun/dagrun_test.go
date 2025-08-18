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
		result     RunResult[stringWrapper]
		stops      map[string]bool // node -> false to indicate failure
		pools      []int           // optional override of pools to run
		cycle      string
		sortResult bool // if true sort result before comparing with expected
	}{
		// disconnected graphs
		{
			name: "empty graph",
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{},
			},
		},
		{
			name:  "one node",
			nodes: []string{"A"},
			seen:  []string{"A"},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{{"A"}},
			},
		},
		{
			name:       "two nodes",
			nodes:      []string{"A", "B"},
			seenSorted: []string{"A", "B"},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{{"A"}, {"B"}},
			},
			sortResult: true,
		},
		{
			name:       "three nodes",
			nodes:      []string{"A", "B", "C"},
			seenSorted: []string{"A", "B", "C"},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{{"A"}, {"B"}, {"C"}},
			},
			sortResult: true,
		},
		{
			name: "simple DAG",
			edges: []edge{
				{"A", "B", "A->B"},
				{"B", "C", "B->C"},
			},
			seen: []string{"A", "B", "C"},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{{"A"}, {"B"}, {"C"}},
			},
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
			name:  "skip downstream on failure",
			edges: []edge{{"A", "B", "A->B"}, {"B", "C", "B->C"}},
			seen:  []string{"A", "B"},
			stops: map[string]bool{"B": false},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{{"A"}},
				Failed:     []stringWrapper{{"B"}},
				NotRun:     []stringWrapper{{"C"}},
			},
		},
		{
			name:       "multiple failures propagate to same node",
			edges:      []edge{{"A", "D", "A->D"}, {"B", "D", "B->D"}},
			seenSorted: []string{"A", "B"},
			stops:      map[string]bool{"A": false, "B": false},
			pools:      []int{1},
			result: RunResult[stringWrapper]{
				Successful: []stringWrapper{},
				Failed:     []stringWrapper{{"A"}, {"B"}},
				NotRun:     []stringWrapper{{"D"}},
			},
			sortResult: true,
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
						g.Run(p, func(n stringWrapper) bool {
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
				result := g.Run(p, func(n stringWrapper) bool {
					mu.Lock()
					seen = append(seen, n.Value)
					mu.Unlock()
					if stop, exists := tc.stops[n.Value]; exists {
						return stop
					}
					return true // success by default
				})

				if tc.sortResult {
					sort.Slice(result.Successful, func(i, j int) bool {
						return result.Successful[i].Value < result.Successful[j].Value
					})
					sort.Slice(result.Failed, func(i, j int) bool {
						return result.Failed[i].Value < result.Failed[j].Value
					})
					sort.Slice(result.NotRun, func(i, j int) bool {
						return result.NotRun[i].Value < result.NotRun[j].Value
					})
				}

				assert.Equal(t, tc.result, result)

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
