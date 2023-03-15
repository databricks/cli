package schema

import (
	"container/list"
	"fmt"
)

type tracker struct {
	// Nodes encountered in current path during the recursive traversal. Used to
	// check for cycles
	seenNodes map[interface{}]struct{}

	// List of node names encountered in order in current path during the recursive traversal.
	// Used to hydrate errors with path to the exact node where error occured.
	//
	// NOTE: node and node names can be the same
	listOfNodes *list.List
}

func newTracker() *tracker {
	return &tracker{
		seenNodes:   map[interface{}]struct{}{},
		listOfNodes: list.New(),
	}
}

func (t *tracker) errWithTrace(prefix string, initTrace string) error {
	traceString := initTrace
	curr := t.listOfNodes.Front()
	for curr != nil {
		if curr.Value.(string) != "" {
			traceString += " -> " + curr.Value.(string)
		}
		curr = curr.Next()
	}
	return fmt.Errorf(prefix + ". traversal trace: " + traceString)
}

func (t *tracker) hasCycle(node interface{}) bool {
	_, ok := t.seenNodes[node]
	return ok
}

func (t *tracker) push(node interface{}, name string) {
	t.seenNodes[node] = struct{}{}
	t.listOfNodes.PushBack(name)
}

func (t *tracker) pop(nodeType interface{}) {
	back := t.listOfNodes.Back()
	t.listOfNodes.Remove(back)
	delete(t.seenNodes, nodeType)
}
