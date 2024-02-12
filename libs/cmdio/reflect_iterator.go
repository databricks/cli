package cmdio

import (
	"context"
	"reflect"
)

type reflectIterator struct {
	hasNext reflect.Value
	next    reflect.Value
}

func newReflectIterator(v any) (reflectIterator, bool) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	_, hasHasNext := rt.MethodByName("HasNext")
	_, hasNext := rt.MethodByName("Next")
	if hasNext && hasHasNext {
		return reflectIterator{
			hasNext: rv.MethodByName("HasNext"),
			next:    rv.MethodByName("Next"),
		}, true
	}
	return reflectIterator{}, false
}

func (r reflectIterator) HasNext(ctx context.Context) bool {
	res := r.hasNext.Call([]reflect.Value{reflect.ValueOf(ctx)})
	return res[0].Bool()
}

func (r reflectIterator) Next(ctx context.Context) (any, error) {
	res := r.next.Call([]reflect.Value{reflect.ValueOf(ctx)})
	item := res[0].Interface()
	if res[1].IsNil() {
		return item, nil
	}
	return item, res[1].Interface().(error)
}
