package structvar

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/structpath"
)

// LookupError is returned by [View.LookupWithDefaults] when the path cannot be followed.
type LookupError struct {
	msg string
}

func (e *LookupError) Error() string {
	return e.msg
}

// KeyNotFoundError is returned by [View.LookupWithDefaults] for a missing key.
type KeyNotFoundError struct {
	Path        *structpath.PathNode
	Suggestions []string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key not found at %q%s", e.Path, DidYouMean(e.Suggestions))
}

func indexedKind(c pathComponent) string {
	if c.isKey {
		return "map"
	}
	return "sequence"
}

// LookupWithDefaults returns the value at path. Fields that are declared in the type
// but not set resolve to their zero value: e.g. ${bundle.git.origin_url} resolves to an
// empty string if a bundle isn't located in a Git repository (yet).
func (x View) LookupWithDefaults(path *structpath.PathNode) (View, error) {
	var p *structpath.PathNode
	for _, c := range components(path) {
		p = c.child(p)
		kind := x.Kind()
		if c.isKey {
			switch kind {
			case KindMap:
			case KindNil:
				return View{}, &LookupError{fmt.Sprintf("expected a %s to index %q, found nil", indexedKind(c), p)}
			default:
				return View{}, &LookupError{fmt.Sprintf("expected a map to index %q, found %s", p, kind)}
			}
			child := x.Get(c.key)
			if !child.IsValid() {
				zero, ok := zeroField(x, c.key)
				if !ok {
					return View{}, &KeyNotFoundError{Path: p, Suggestions: SuggestKeys(lookupKeys(x), c.key)}
				}
				child = zero
			}
			x = child
			continue
		}

		switch kind {
		case KindSequence:
		case KindNil:
			return View{}, &LookupError{fmt.Sprintf("expected a %s to index %q, found nil", indexedKind(c), p)}
		default:
			return View{}, &LookupError{fmt.Sprintf("expected a sequence to index %q, found %s", p, kind)}
		}
		child := x.Index(c.index)
		if !child.IsValid() {
			return View{}, &LookupError{fmt.Sprintf("index out of bounds at %q", p)}
		}
		x = child
	}
	return x, nil
}

// structType returns the struct type of the value, if it is a struct.
func structType(x View) (reflect.Type, bool) {
	t := x.Reflect().Type()
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		if t.Kind() == reflect.Interface {
			v := x.Reflect()
			for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
				if v.IsNil() {
					return nil, false
				}
				v = v.Elem()
			}
			t = v.Type()
			continue
		}
		t = t.Elem()
	}
	return t, t.Kind() == reflect.Struct && !IsSDKNativeType(t)
}

// fillableFieldType returns the type of the struct field with JSON name k, if a field of
// that type is given a zero value when it is not set (see convert.IncludeMissingFields).
func fillableFieldType(t reflect.Type, k string) (reflect.Type, bool) {
	info := GetStructInfo(t)
	index, ok := info.Fields[k]
	if !ok {
		return nil, false
	}
	ft := t.FieldByIndex(index).Type
	for ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	switch ft.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return ft, true
	default:
		return nil, false
	}
}

// zeroField returns the zero value of the struct field k of x that is not set.
func zeroField(x View, k string) (View, bool) {
	t, ok := structType(x)
	if !ok {
		return View{}, false
	}
	ft, ok := fillableFieldType(t, k)
	if !ok {
		return View{}, false
	}
	zero := reflect.New(ft)
	switch ft.Kind() {
	case reflect.Map:
		zero.Elem().Set(reflect.MakeMap(ft))
	case reflect.Slice:
		zero.Elem().Set(reflect.MakeSlice(ft, 0, 0))
	default:
	}
	return NewView(zero.Interface(), nil, nil), true
}

// lookupKeys returns the keys a lookup in x can find, for "did you mean" suggestions.
func lookupKeys(x View) []string {
	var keys []string
	for k := range x.MapItems() {
		keys = append(keys, k)
	}
	if t, ok := structType(x); ok {
		for _, k := range GetStructInfo(t).FieldNames {
			if _, ok := fillableFieldType(t, k); ok && !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	return keys
}
