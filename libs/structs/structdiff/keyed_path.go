package structdiff

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// KeyedPath replaces slice indices with the selectors used by GetStructDiff.
// Indices without a matching key function and existing selectors are preserved.
func KeyedPath(root any, path *structpath.PathNode, sliceKeys map[string]KeyFunc) (*structpath.PathNode, error) {
	diff := &diffContext{sliceKeys: sliceKeys}
	var result *structpath.PathNode
	for _, node := range path.AsSlice() {
		index, isIndex := node.Index()
		switch {
		case isIndex:
			keyFunc := diff.findKeyFunc(result)
			if keyFunc == nil {
				result = structpath.NewIndex(result, index)
				continue
			}
			element, err := structaccess.Get(root, node)
			if err != nil {
				return nil, err
			}
			caller, err := newKeyFuncCaller(keyFunc)
			if err != nil {
				return nil, err
			}
			if element == nil || !reflect.TypeOf(element).AssignableTo(caller.argType) {
				return nil, fmt.Errorf("KeyFunc expects %v at %s, got %T", caller.argType, node, element)
			}
			key, value := caller.call(element)
			result = structpath.NewKeyValue(result, key, value)
		default:
			if key, value, ok := node.KeyValue(); ok {
				result = structpath.NewKeyValue(result, key, value)
			} else if key, ok := node.StringKey(); ok {
				result = structpath.NewStringKey(result, key)
			} else {
				return nil, fmt.Errorf("unsupported path node: %s", node)
			}
		}
	}
	return result, nil
}
