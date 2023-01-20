package schema

import (
	"container/list"
	"fmt"
	"reflect"
)

type tracker struct {
	// Types encountered in current path during the recursive traversal. Used to
	// check for cycles
	seenTypes map[reflect.Type]struct{}

	// List of field names encountered in current path during the recursive traversal.
	// Used to hydrate errors with path to the exact node where error occured.
	//
	// The field names here are the first tag in the json tags of struct field.
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
	if ok {
		fmt.Println("[DEBUG] traceSet for cycle: ", t.seenTypes)
	}
	return ok
}

func (t *tracker) push(nodeType reflect.Type, jsonName string) {
	t.seenTypes[nodeType] = struct{}{}
	t.debugTrace.PushBack(jsonName)
}

func (t *tracker) pop(nodeType reflect.Type) {
	back := t.debugTrace.Back()
	t.debugTrace.Remove(back)
	delete(t.seenTypes, nodeType)
}
