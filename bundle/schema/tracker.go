package schema

import (
	"container/list"
	"fmt"
	"reflect"
)

type tracker struct {
	// Types encountered in path of reaching the current type. Used to deletect
	// cycles
	seenTypes map[reflect.Type]struct{}

	// List of names from json tag encountered while reaching current type. This
	// is logged on any error so we know on which type an error occured
	debugTrace *list.List
}

func newTracker() *tracker {
	return &tracker{
		seenTypes:  map[reflect.Type]struct{}{},
		debugTrace: list.New(),
	}
}

func (t *tracker) errWithTrace(prefix string) error {
	traceString := "root"
	curr := t.debugTrace.Front()
	for curr != nil {
		if curr.Value.(string) != "" {
			traceString += " -> " + curr.Value.(string)
		}
		curr = curr.Next()
	}
	return fmt.Errorf("[ERROR] " + prefix + ". traversal trace: " + traceString)
}

func (t *tracker) hasCycle(golangType reflect.Type) bool {
	_, ok := t.seenTypes[golangType]
	if !ok {
		fmt.Println("[DEBUG] traceSet for cycle: ", t.seenTypes)
	}
	return ok
}

func (t *tracker) step(nodeType reflect.Type, jsonName string) {
	t.seenTypes[nodeType] = struct{}{}
	t.debugTrace.PushBack(jsonName)
}

func (t *tracker) undoStep(nodeType reflect.Type) {
	back := t.debugTrace.Back()
	t.debugTrace.Remove(back)
	delete(t.seenTypes, nodeType)
}
