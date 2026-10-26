package structdiff

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// KeyedPath replaces slice indices with the selectors used by GetStructDiff.
// Keyed elements must have nonempty, unique identities.
// Indices and selectors without a matching key function are preserved.
func KeyedPath(root any, path *structpath.PathNode, sliceKeys map[string]KeyFunc) (*structpath.PathNode, error) {
	diff := &diffContext{sliceKeys: sliceKeys}
	var result *structpath.PathNode
	for _, node := range path.AsSlice() {
		index, isIndex := node.Index()
		key, value, isSelector := node.KeyValue()
		switch {
		case isIndex || isSelector:
			keyFunc := diff.findKeyFunc(result)
			if keyFunc == nil {
				if isIndex {
					result = structpath.NewIndex(result, index)
				} else {
					result = structpath.NewKeyValue(result, key, value)
				}
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
			key, value = caller.call(element)
			if key == "" || value == "" {
				return nil, fmt.Errorf("empty identity at %s", node)
			}
			if err := validateUniqueIdentity(root, node.Parent(), caller, key, value); err != nil {
				return nil, err
			}
			result = structpath.NewKeyValue(result, key, value)
		default:
			if key, ok := node.StringKey(); ok {
				result = structpath.NewStringKey(result, key)
			} else {
				return nil, fmt.Errorf("unsupported path node: %s", node)
			}
		}
	}
	return result, nil
}

func validateUniqueIdentity(root any, path *structpath.PathNode, caller *keyFuncCaller, key, value string) error {
	parent, err := structaccess.Get(root, path)
	if err != nil {
		return err
	}
	sequence := reflect.ValueOf(parent)
	if sequence.Kind() != reflect.Slice && sequence.Kind() != reflect.Array {
		return fmt.Errorf("expected sequence at %s, got %T", path, parent)
	}
	if err := validateKeyFuncElementType(sequence, caller.argType); err != nil {
		return err
	}
	matches := 0
	for i := range sequence.Len() {
		k, v := caller.call(sequence.Index(i).Interface())
		if k == key && v == value {
			matches++
		}
	}
	if matches != 1 {
		return fmt.Errorf("ambiguous identity at %s: %s=%q matches %d elements", path, key, value, matches)
	}
	return nil
}
