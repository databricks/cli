package dag

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Lessable[T comparable] interface {
	comparable
	Less(T) bool
}

type adjEdge[N Lessable[N]] struct {
	to    N
	label string
}

type Graph[N Lessable[N]] struct {
	adj map[N][]adjEdge[N]
}

func NewGraph[N Lessable[N]]() *Graph[N] {
	return &Graph[N]{adj: make(map[N][]adjEdge[N])}
}

func (g *Graph[N]) AddNode(n N) {
	if _, ok := g.adj[n]; !ok {
		g.adj[n] = nil
	}
}

func (g *Graph[N]) AddDirectedEdge(from, to N, label string) error {
	if from == to {
		return fmt.Errorf("self-loop %v", from)
	}
	g.AddNode(from)
	g.AddNode(to)
	g.adj[from] = append(g.adj[from], adjEdge[N]{to: to, label: label})
	return nil
}

type CycleError[N Lessable[N]] struct {
	Nodes []N
	Edges []string
}

func (e *CycleError[N]) Error() string {
	if len(e.Nodes) == 0 {
		return "cycle detected"
	}
	var b strings.Builder
	b.WriteString("cycle detected: ")
	fmt.Fprint(&b, e.Nodes[0])
	for i := 1; i < len(e.Nodes); i++ {
		b.WriteString(" refers to ")
		fmt.Fprint(&b, e.Nodes[i])
		b.WriteString(" via ")
		b.WriteString(e.Edges[i-1])
	}
	b.WriteString(" which refers to ")
	fmt.Fprint(&b, e.Nodes[0])
	b.WriteString(" via ")
	b.WriteString(e.Edges[len(e.Edges)-1])
	b.WriteString(".")
	return b.String()
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
	// 1. sort every adjacency list once
	for k, outs := range g.adj {
		sort.Slice(outs, func(i, j int) bool { return outs[i].to.Less(outs[j].to) })
		g.adj[k] = outs
	}

	// 2. sorted list of roots
	roots := make([]N, 0, len(g.adj))
	for v := range g.adj {
		roots = append(roots, v)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Less(roots[j]) })

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

func (g *Graph[N]) Run(pool int, runUnit func(N)) error {
	if err := g.DetectCycle(); err != nil {
		return err
	}
	if pool <= 0 || pool > len(g.adj) {
		pool = len(g.adj)
	}

	in := g.indegrees()
	ready := make(chan N, len(in))
	done := make(chan N, len(in))

	var wg sync.WaitGroup
	wg.Add(pool)
	for i := 0; i < pool; i++ {
		go func() {
			defer wg.Done()
			for n := range ready {
				runUnit(n)
				done <- n
			}
		}()
	}

	// stable initial-ready order
	keys := make([]N, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Less(keys[j]) })
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
