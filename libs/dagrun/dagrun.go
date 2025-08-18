package dagrun

import (
	"fmt"
	"strings"
	"sync"
)

type StringerComparable interface {
	fmt.Stringer
	comparable
}

type adjEdge[N StringerComparable] struct {
	to    N
	label string
}

type Graph[N StringerComparable] struct {
	adj   map[N][]adjEdge[N]
	nodes []N // maintains insertion order of added nodes
}

func NewGraph[N StringerComparable]() *Graph[N] {
	return &Graph[N]{adj: make(map[N][]adjEdge[N])}
}

func (g *Graph[N]) Size() int {
	return len(g.nodes)
}

func (g *Graph[N]) AddNode(n N) {
	if _, ok := g.adj[n]; !ok {
		g.adj[n] = nil
		g.nodes = append(g.nodes, n)
	}
}

func (g *Graph[N]) HasNode(n N) bool {
	_, ok := g.adj[n]
	return ok
}

func (g *Graph[N]) AddDirectedEdge(from, to N, label string) {
	g.AddNode(from)
	g.AddNode(to)
	g.adj[from] = append(g.adj[from], adjEdge[N]{to: to, label: label})
}

type CycleError[N comparable] struct {
	Nodes []N
	Edges []string
}

func (e *CycleError[N]) Error() string {
	if len(e.Nodes) == 0 {
		return "cycle detected"
	}

	if len(e.Nodes) == 1 {
		return fmt.Sprintf("cycle detected: %v refers to itself via %s", e.Nodes[0], e.Edges[0])
	}

	// Build "A refers to B via E1" pieces for every edge except the closing one.
	var parts []string
	for i := 1; i < len(e.Nodes); i++ {
		parts = append(parts, fmt.Sprintf("%v refers to %v via %s", e.Nodes[i-1], e.Nodes[i], e.Edges[i-1]))
	}

	return fmt.Sprintf(
		"cycle detected: %s which refers to %v via %s",
		strings.Join(parts, " "),
		e.Nodes[0],
		e.Edges[len(e.Edges)-1],
	)
}

func (g *Graph[N]) indegrees() map[N]int {
	in := make(map[N]int, len(g.adj))
	for v := range g.adj {
		in[v] = 0
	}
	for _, outs := range g.adj {
		for _, e := range outs {
			in[e.to]++
		}
	}
	return in
}

func (g *Graph[N]) DetectCycle() error {
	// Build list of roots in insertion order
	roots := g.nodes

	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := make(map[N]int, len(g.adj))

	type frame struct {
		node  N
		inLbl string
		next  int
	}
	var st stack[frame]

	for _, root := range roots {
		if color[root] != white {
			continue
		}
		color[root] = grey
		st.push(frame{node: root})

		for st.len() > 0 {
			f := st.peek()
			outs := g.adj[f.node]

			if f.next < len(outs) {
				edge := outs[f.next]
				st.peek().next++
				switch color[edge.to] {
				case white:
					color[edge.to] = grey
					st.push(frame{node: edge.to, inLbl: edge.label})
				case grey:
					closeLbl := edge.label
					var nodes []N
					var edges []string
					for i := st.len() - 1; i >= 0; i-- {
						nodes = append(nodes, st.data[i].node)
						if lbl := st.data[i].inLbl; lbl != "" {
							edges = append(edges, lbl)
						}
						if st.data[i].node == edge.to {
							break
						}
					}
					for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
						nodes[i], nodes[j] = nodes[j], nodes[i]
					}
					for i, j := 0, len(edges)-1; i < j; i, j = i+1, j-1 {
						edges[i], edges[j] = edges[j], edges[i]
					}
					edges = append(edges, closeLbl)
					return &CycleError[N]{Nodes: nodes, Edges: edges}
				}
			} else {
				color[f.node] = black
				st.pop()
			}
		}
	}
	return nil
}

// RunResult contains the categorized results of a DAG execution.
type RunResult[N StringerComparable] struct {
	// Successful contains nodes that executed successfully.
	Successful []N
	// Failed contains nodes that failed to execute.
	Failed []N
	// NotRun contains nodes that were not executed because their dependencies failed.
	NotRun []N
}

// Run executes the DAG with up to pool concurrent workers. The runUnit callback
// returns true for success (dependencies will be processed) or false for failure
// (dependencies will be skipped). The function returns a RunResult with nodes
// categorized by their execution outcome.
func (g *Graph[N]) Run(pool int, runUnit func(N) bool) RunResult[N] {
	if pool <= 0 || pool > len(g.adj) {
		pool = len(g.adj)
	}

	in := g.indegrees()

	// Prepare initial ready nodes in stable insertion order
	var initial []N
	for _, n := range g.nodes {
		if in[n] == 0 {
			initial = append(initial, n)
		}
	}

	// If there are nodes but no entry points, the run cannot start
	if len(in) > 0 && len(initial) == 0 {
		panic("dagrun: no entry points")
	}

	// Track execution outcomes
	successful := make([]N, 0, len(in)) // assume all succeed
	failed := make([]N, 0, len(in)/4)   // assume few failures
	notRun := make(map[N]bool, len(in))
	// finalized marks nodes whose outcome is already decided
	// so we decrement the remaining counter exactly once per node.
	finalized := make(map[N]bool, len(in))

	ready := make(chan N, len(in))
	done := make(chan doneResult[N], len(in))

	var wg sync.WaitGroup
	wg.Add(pool)
	for range pool {
		go runWorkerLoop[N](&wg, ready, done, runUnit)
	}

	for _, n := range initial {
		ready <- n
	}

	for remaining := len(in); remaining > 0; {
		res := <-done

		// Mark the current node as finalized first
		if !finalized[res.n] {
			finalized[res.n] = true
			remaining--
		}

		if !res.success {
			failed = append(failed, res.n)
			propagateCancelFrom[N](g, res.n, &remaining, finalized, notRun)
		} else {
			successful = append(successful, res.n)
		}

		for _, e := range g.adj[res.n] {
			if in[e.to]--; in[e.to] == 0 {
				if !notRun[e.to] {
					ready <- e.to
				}
			}
		}
	}
	close(ready)
	wg.Wait()

	// Build result slices in insertion order
	var result RunResult[N]
	result.Successful = successful
	result.Failed = failed
	result.NotRun = make([]N, 0, len(in)/4) // assume few not run
	for _, n := range g.nodes {
		if notRun[n] {
			result.NotRun = append(result.NotRun, n)
		}
	}
	return result
}

type doneResult[N StringerComparable] struct {
	n       N
	success bool
}

func runWorkerLoop[N StringerComparable](wg *sync.WaitGroup, ready <-chan N, done chan<- doneResult[N], runUnit func(N) bool) {
	defer wg.Done()
	for n := range ready {
		success := runUnit(n)
		done <- doneResult[N]{n: n, success: success}
	}
}

func propagateCancelFrom[N StringerComparable](g *Graph[N], src N, remaining *int, finalized, notRun map[N]bool) {
	var queue []N
	for _, e := range g.adj[src] {
		if !finalized[e.to] {
			notRun[e.to] = true
			finalized[e.to] = true
			*remaining--
			queue = append(queue, e.to)
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, e := range g.adj[n] {
			if !finalized[e.to] {
				notRun[e.to] = true
				finalized[e.to] = true
				*remaining--
				queue = append(queue, e.to)
			}
		}
	}
}
