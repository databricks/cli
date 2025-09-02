package calladapt

import (
	"fmt"
	"reflect"
)

// EnsureNoExtraMethods ensures receiver's exported methods are a subset of ifaceType methods.
func EnsureNoExtraMethods(receiver any, ifaceType reflect.Type) error {
	rt := reflect.TypeOf(receiver)
	allowed := make(map[string]struct{}, ifaceType.NumMethod())
	for i := range ifaceType.NumMethod() {
		allowed[ifaceType.Method(i).Name] = struct{}{}
	}
	for i := range rt.NumMethod() {
		m := rt.Method(i)
		if _, ok := allowed[m.Name]; !ok {
			return fmt.Errorf("unexpected exported method %s on %v; only methods from %v are allowed", m.Name, rt, ifaceType)
		}
	}
	return nil
}
