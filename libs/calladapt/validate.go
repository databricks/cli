package calladapt

import (
	"fmt"
	"reflect"
)

// EnsureNoExtraMethods ensures receiver's exported methods are a subset of the provided interface types' methods.
func EnsureNoExtraMethods(receiver any, ifaceTypes ...reflect.Type) error {
	rt := reflect.TypeOf(receiver)

	allowed := make(map[string]struct{})
	for _, ifaceType := range ifaceTypes {
		for method := range ifaceType.Methods() {
			allowed[method.Name] = struct{}{}
		}
	}

	for m := range rt.Methods() {
		if _, ok := allowed[m.Name]; !ok {
			return fmt.Errorf("unexpected method %s on %v; only methods from %v are allowed", m.Name, rt, ifaceTypes)
		}
	}
	return nil
}
