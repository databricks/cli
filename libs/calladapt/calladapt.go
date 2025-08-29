package calladapt

import (
	"fmt"
	"reflect"
)

// TypeOf returns reflect.Type for type parameter T, analogous to
// reflect.TypeOf((*T)(nil)).Elem().
func TypeOf[T any]() reflect.Type {
	var t *T
	return reflect.TypeOf(t).Elem()
}

// BoundCaller encapsulates a bound method and metadata about its signature.
// It can invoke the underlying function and returns all non-error outputs and
// the error (if the method returns one as the last return value).
type BoundCaller struct {
	Func             reflect.Value
	Receiver         reflect.Value
	InTypes          []reflect.Type
	OutTypes         []reflect.Type
	Name             string
	HasTrailingError bool
}

type CallAdaptError struct {
	Msg string
}

func (e CallAdaptError) Error() string {
	return e.Msg
}

func (c *BoundCaller) call(args ...any) ([]reflect.Value, error) {
	if len(args) != len(c.InTypes) {
		return nil, &CallAdaptError{Msg: fmt.Sprintf("%s: want %d args, got %d", c.Name, len(c.InTypes), len(args))}
	}
	in := make([]reflect.Value, 1+len(args))
	in[0] = c.Receiver
	for i, a := range args {
		it := c.InTypes[i]
		if a == nil {
			return nil, &CallAdaptError{Msg: fmt.Sprintf("%s: arg %d type mismatch: want %v, got nil", c.Name, i, it)}
		}
		v := reflect.ValueOf(a)
		vt := v.Type()
		// Allow passing a value that is assignable to the expected type. This
		// includes concrete types that implement an interface parameter type.
		if vt != it && !vt.AssignableTo(it) {
			return nil, &CallAdaptError{Msg: fmt.Sprintf("%s: arg %d type mismatch: want %v, got %v", c.Name, i, it, vt)}
		}
		in[i+1] = v
	}
	return c.Func.Call(in), nil
}

// Call returns all non-error outputs and an error if the last return value is error and non-nil.
func (c *BoundCaller) Call(args ...any) ([]any, error) {
	outs, err := c.call(args...)
	if err != nil {
		return nil, err
	}
	n := len(outs)
	result := make([]any, 0, n)
	if n == 0 {
		return result, nil
	}
	if c.HasTrailingError {
		lastIdx := n - 1
		last := outs[lastIdx]
		for i := range lastIdx {
			result = append(result, outs[i].Interface())
		}
		if last.IsNil() {
			return result, nil
		}
		return result, last.Interface().(error)
	}
	for i := range n {
		result = append(result, outs[i].Interface())
	}
	return result, nil
}

var (
	errType = TypeOf[error]()
	anyType = TypeOf[any]()
)

// PrepareCall creates a unified BoundCaller for the given method on receiver that matches the ifaceType method.
func PrepareCall(receiver any, ifaceType reflect.Type, methodName string) (*BoundCaller, error) {
	if receiver == nil {
		return nil, &CallAdaptError{Msg: "first argument must not be untyped nil"}
	}
	rt := reflect.TypeOf(receiver)
	if ifaceType == nil || ifaceType.Kind() != reflect.Interface {
		return nil, &CallAdaptError{Msg: "second argument must be an interface reflect.Type"}
	}
	im, ok := ifaceType.MethodByName(methodName)
	if !ok {
		return nil, &CallAdaptError{Msg: fmt.Sprintf("%v has no method %q", ifaceType, methodName)}
	}
	mt, ok := rt.MethodByName(methodName)
	if !ok {
		return nil, nil
	}

	// Check compatibility and build type arrays
	ifaceFT := im.Type
	concFT := mt.Type

	// Check parameter count and compatibility
	argN := concFT.NumIn() - 1
	if argN != ifaceFT.NumIn() {
		return nil, &CallAdaptError{Msg: fmt.Sprintf("%v.%s: param count mismatch: iface %d, concrete %d (incl. recv)",
			ifaceType, methodName, ifaceFT.NumIn(), concFT.NumIn())}
	}
	inTypes := make([]reflect.Type, argN)
	for i := range argN {
		inTypes[i] = concFT.In(i + 1)
		it := ifaceFT.In(i)
		// Interface side may use `any` as a wildcard
		if it != anyType && inTypes[i] != it {
			return nil, &CallAdaptError{Msg: fmt.Sprintf("%v.%s: param %d mismatch: iface %v, concrete %v",
				ifaceType, methodName, i, it, inTypes[i])}
		}
	}

	// Check return count and compatibility
	outN := concFT.NumOut()
	if outN != ifaceFT.NumOut() {
		return nil, &CallAdaptError{Msg: fmt.Sprintf("%v.%s: return count mismatch: iface %d, concrete %d",
			ifaceType, methodName, ifaceFT.NumOut(), outN)}
	}
	outTypes := make([]reflect.Type, outN)
	for i := range outN {
		outTypes[i] = concFT.Out(i)
		it := ifaceFT.Out(i)
		// Interface side may use `any` as a wildcard
		if it != anyType && outTypes[i] != it {
			return nil, &CallAdaptError{Msg: fmt.Sprintf("%v.%s: result %d mismatch: iface %v, concrete %v",
				ifaceType, methodName, i, it, outTypes[i])}
		}
	}

	hasErr := outN > 0 && outTypes[outN-1] == errType
	bc := &BoundCaller{
		Func:             mt.Func,
		Receiver:         reflect.ValueOf(receiver),
		InTypes:          inTypes,
		OutTypes:         outTypes,
		Name:             methodName,
		HasTrailingError: hasErr,
	}
	return bc, nil
}
