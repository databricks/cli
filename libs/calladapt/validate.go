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
		for i := range ifaceType.NumMethod() {
			allowed[ifaceType.Method(i).Name] = struct{}{}
		}
	}

	for i := range rt.NumMethod() {
		m := rt.Method(i)
		if _, ok := allowed[m.Name]; !ok {
			return fmt.Errorf("unexpected method %s on %v; only methods from %v are allowed", m.Name, rt, ifaceTypes)
		}
	}
	return nil
}
