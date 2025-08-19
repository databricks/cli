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

// Run executes the DAG with up to pool concurrent workers. The runUnit callback
// receives the node and an optional failed dependency node pointer. If failedDependency
// is non-nil, it indicates that at least one direct dependency of the node failed.
// The callback should return true on success or false on failure. Nodes are not
// skipped when dependencies fail; instead, they are executed with failedDependency set.
func (g *Graph[N]) Run(pool int, runUnit func(N, *N) bool) {
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

	// For each node, remember a failed direct dependency (any one) if present.
	failedCause := make(map[N]*N, len(in))

	ready := make(chan task[N], len(in))
	done := make(chan doneResult[N], len(in))

	var wg sync.WaitGroup
	wg.Add(pool)
	for range pool {
		go runWorkerLoop(&wg, ready, done, runUnit)
	}

	for _, n := range initial {
		ready <- task[N]{n: n, failedFrom: nil}
	}

	for remaining := len(in); remaining > 0; remaining-- {
		res := <-done

		if !res.success {
			// Determine the originating failure cause to propagate
			var cause *N
			if existing, ok := failedCause[res.n]; ok && existing != nil {
				cause = existing
			} else {
				parent := res.n
				cause = &parent
			}
			// Record a failed dependency cause for children, if not set yet
			for _, e := range g.adj[res.n] {
				if _, exists := failedCause[e.to]; !exists {
					failedCause[e.to] = cause
				}
			}
		}

		// Decrement indegrees and enqueue children that become ready
		for _, e := range g.adj[res.n] {
			if in[e.to]--; in[e.to] == 0 {
				ready <- task[N]{n: e.to, failedFrom: failedCause[e.to]}
			}
		}
	}
	close(ready)
	wg.Wait()
}

type doneResult[N StringerComparable] struct {
	n       N
	success bool
}

type task[T StringerComparable] struct {
	n          T
	failedFrom *T
}

func runWorkerLoop[N StringerComparable](wg *sync.WaitGroup, ready <-chan task[N], done chan<- doneResult[N], runUnit func(N, *N) bool) {
	defer wg.Done()
	for t := range ready {
		success := runUnit(t.n, t.failedFrom)
		if t.failedFrom != nil {
			// Enforce failure status when a dependency has failed
			success = false
		}
		done <- doneResult[N]{n: t.n, success: success}
	}
}
