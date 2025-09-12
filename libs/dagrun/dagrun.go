package dagrun

import (
	"fmt"
	"strings"
	"sync"
)

type adjEdge struct {
	To    string
	Label string
}

type inboundEdge struct {
	From  string
	Label string
}

type Graph struct {
	Adj     map[string][]adjEdge
	Inbound map[string][]inboundEdge // maintains all inbound edges to the node
	Nodes   []string                 // maintains insertion order of added nodes
}

func NewGraph() *Graph {
	return &Graph{
		Adj:     make(map[string][]adjEdge),
		Inbound: make(map[string][]inboundEdge),
	}
}

func (g *Graph) Size() int { return len(g.Nodes) }

func (g *Graph) AddNode(n string) {
	if _, ok := g.Adj[n]; !ok {
		g.Adj[n] = nil
		g.Nodes = append(g.Nodes, n)
	}
}

func (g *Graph) HasNode(n string) bool { _, ok := g.Adj[n]; return ok }

// HasOutgoingEdges reports whether this node has at least one outgoing edge.
// In this graph, an outgoing edge from X->Y means Y references X.
func (g *Graph) HasOutgoingEdges(n string) bool { return len(g.Adj[n]) > 0 }

func (g *Graph) AddDirectedEdge(from, to, label string) {
	g.AddNode(from)
	g.AddNode(to)
	g.Adj[from] = append(g.Adj[from], adjEdge{To: to, Label: label})
	g.Inbound[to] = append(g.Inbound[to], inboundEdge{From: from, Label: label})
}

// OutgoingLabels returns labels of all outgoing edges from the given node
// in the order the edges were added. If the node has no outgoing edges or is
// unknown to the graph, an empty slice is returned.
func (g *Graph) OutgoingLabels(node string) []string {
	outs := g.Adj[node]
	if len(outs) == 0 {
		return []string{}
	}
	labels := make([]string, 0, len(outs))
	seen := make(map[string]struct{}, len(outs))
	for _, e := range outs {
		if _, ok := seen[e.Label]; ok {
			continue
		}
		seen[e.Label] = struct{}{}
		labels = append(labels, e.Label)
	}
	return labels
}

type CycleError struct {
	Nodes []string
	Edges []string
}

func (e *CycleError) Error() string {
	if len(e.Nodes) == 0 {
		return "cycle detected"
	}

	if len(e.Nodes) == 1 {
		return fmt.Sprintf("cycle detected: %v refers to itself via %s", e.Nodes[0], e.Edges[0])
	}

	// Build "to refers to from via edge" pieces for every edge except the closing one.
	var parts []string
	for i := 1; i < len(e.Nodes); i++ {
		parts = append(parts, fmt.Sprintf("%v refers to %v via %s", e.Nodes[i], e.Nodes[i-1], e.Edges[i-1]))
	}

	return fmt.Sprintf(
		"cycle detected: %s which refers to %v via %s",
		strings.Join(parts, " "),
		e.Nodes[len(e.Nodes)-1],
		e.Edges[len(e.Edges)-1],
	)
}

func (g *Graph) indegrees() map[string]int {
	in := make(map[string]int, len(g.Adj))
	for v := range g.Adj {
		in[v] = 0
	}
	for _, outs := range g.Adj {
		for _, e := range outs {
			in[e.To]++
		}
	}
	return in
}

func (g *Graph) DetectCycle() error {
	// Build list of roots in insertion order
	roots := g.Nodes

	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := make(map[string]int, len(g.Adj))

	type frame struct {
		node  string
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
			outs := g.Adj[f.node]

			if f.next < len(outs) {
				edge := outs[f.next]
				st.peek().next++
				switch color[edge.To] {
				case white:
					color[edge.To] = grey
					st.push(frame{node: edge.To, inLbl: edge.Label})
				case grey:
					closeLbl := edge.Label
					var nodes []string
					var edges []string
					for i := st.len() - 1; i >= 0; i-- {
						nodes = append(nodes, st.data[i].node)
						if lbl := st.data[i].inLbl; lbl != "" {
							edges = append(edges, lbl)
						}
						if st.data[i].node == edge.To {
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
					return &CycleError{Nodes: nodes, Edges: edges}
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
func (g *Graph) Run(pool int, runUnit func(node string, failedDependency *string) bool) {
	if pool <= 0 || pool > len(g.Adj) {
		pool = len(g.Adj)
	}

	in := g.indegrees()

	// Prepare initial ready nodes in stable insertion order
	var initial []string
	for _, n := range g.Nodes {
		if in[n] == 0 {
			initial = append(initial, n)
		}
	}

	// If there are nodes but no entry points, the run cannot start
	if len(in) > 0 && len(initial) == 0 {
		panic("dagrun: no entry points")
	}

	// For each node, remember a failed direct dependency (any one) if present.
	failedCause := make(map[string]*string, len(in))

	ready := make(chan task, len(in))
	done := make(chan doneResult, len(in))

	var wg sync.WaitGroup
	wg.Add(pool)
	for range pool {
		go runWorkerLoop(&wg, ready, done, runUnit)
	}

	for _, n := range initial {
		ready <- task{n: n, failedFrom: nil}
	}

	for remaining := len(in); remaining > 0; remaining-- {
		res := <-done

		if !res.success {
			// Determine the originating failure cause to propagate
			var cause *string
			if existing, ok := failedCause[res.n]; ok && existing != nil {
				cause = existing
			} else {
				parent := res.n
				cause = &parent
			}
			// Record a failed dependency cause for children, if not set yet
			for _, e := range g.Adj[res.n] {
				if _, exists := failedCause[e.To]; !exists {
					failedCause[e.To] = cause
				}
			}
		}

		// Decrement indegrees and enqueue children that become ready
		for _, e := range g.Adj[res.n] {
			if in[e.To]--; in[e.To] == 0 {
				ready <- task{n: e.To, failedFrom: failedCause[e.To]}
			}
		}
	}
	close(ready)
	wg.Wait()
}

type doneResult struct {
	n       string
	success bool
}

type task struct {
	n          string
	failedFrom *string
}

func runWorkerLoop(wg *sync.WaitGroup, ready <-chan task, done chan<- doneResult, runUnit func(string, *string) bool) {
	defer wg.Done()
	for t := range ready {
		success := runUnit(t.n, t.failedFrom)
		if t.failedFrom != nil {
			// Enforce failure status when a dependency has failed
			success = false
		}
		done <- doneResult{n: t.n, success: success}
	}
}
