package dag

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type adjEdge struct {
	to    string
	label string
}

type Graph struct {
	adj map[string][]adjEdge
}

func NewGraph() *Graph { return &Graph{adj: make(map[string][]adjEdge)} }

func (g *Graph) AddNode(name string) {
	if _, ok := g.adj[name]; !ok {
		g.adj[name] = nil
	}
}

// AddDirectedEdge inserts from → to with label.
func (g *Graph) AddDirectedEdge(from, to, label string) error {
	if from == to {
		return fmt.Errorf("self-loop %q", from)
	}
	g.AddNode(to)
	g.adj[from] = append(g.adj[from], adjEdge{to: to, label: label})
	return nil
}

type CycleError struct {
	Nodes []string
	Edges []string
}

func (e *CycleError) Error() string {
	if len(e.Nodes) == 0 {
		return "cycle detected"
	}
	var b strings.Builder
	b.WriteString("cycle detected: ")
	b.WriteString(e.Nodes[0])
	for i := 1; i < len(e.Nodes); i++ {
		b.WriteString(" refers to ")
		b.WriteString(e.Nodes[i])
		b.WriteString(" via ")
		b.WriteString(e.Edges[i-1])
	}
	b.WriteString(" which refers to ")
	b.WriteString(e.Nodes[0])
	b.WriteString(" via ")
	b.WriteString(e.Edges[len(e.Edges)-1])
	b.WriteString(".")
	return b.String()
}

func (g *Graph) indegrees() map[string]int {
	in := make(map[string]int, len(g.adj))
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

/* non-recursive DFS cycle check */

func (g *Graph) DetectCycle() error {
	for _, outs := range g.adj {
		sort.Slice(outs, func(i, j int) bool { return outs[i].to < outs[j].to })
	}
	roots := make([]string, 0, len(g.adj))
	for v := range g.adj {
		roots = append(roots, v)
	}
	sort.Strings(roots)

	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := make(map[string]int, len(g.adj))

	type frame struct {
		node  string
		inLbl string // edge label via which we entered this node
		next  int    // next neighbour index to explore
	}
	var st stack[frame]

	for _, r := range roots {
		if color[r] != white {
			continue
		}
		color[r] = grey
		st.push(frame{node: r})

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
				case grey: // back-edge → cycle
					closeLbl := edge.label
					var nodes, edges []string
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

/* Run with fixed worker pool */

func (g *Graph) Run(pool int, runUnit func(string)) error {
	if err := g.DetectCycle(); err != nil {
		return err
	}

	if pool > len(g.adj) {
		pool = len(g.adj)
	}

	in := g.indegrees()
	ready := make(chan string, len(in))
	done := make(chan string, len(in))

	var wg sync.WaitGroup
	wg.Add(pool)
	for range pool {
		go func() {
			defer wg.Done()
			for n := range ready {
				runUnit(n)
				done <- n
			}
		}()
	}

	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, n := range keys {
		if in[n] == 0 {
			ready <- n
		}
	}

	for remaining := len(in); remaining > 0; {
		n := <-done
		remaining--
		for _, e := range g.adj[n] {
			if in[e.to]--; in[e.to] == 0 {
				ready <- e.to
			}
		}
	}
	close(ready)
	wg.Wait()
	return nil
}
