// Package cmdgen holds runtime helpers for the generated commands in
// cmd/workspace and cmd/account (see internal/cligen).
package cmdgen

import (
	"context"
	"fmt"
	"iter"
	"reflect"
)

// MapSeq converts the elements of an sdk-go list iterator, typically into the
// JSON mirror types of libs/sdktypes so that rendering matches the API's JSON.
func MapSeq[S, T any](seq iter.Seq2[*S, error], conv func(*S) (*T, error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for s, err := range seq {
			if err != nil {
				yield(nil, err)
				return
			}
			t, err := conv(s)
			if !yield(t, err) || err != nil {
				return
			}
		}
	}
}

// NamedIdMap lists all items of seq and maps the value at namePath to the value
// at idPath (Go field names, following pointers), for interactive drop-downs.
func NamedIdMap[T any](ctx context.Context, seq iter.Seq2[*T, error], namePath, idPath []string) (map[string]string, error) {
	out := map[string]string{}
	for item, err := range seq {
		if err != nil {
			return nil, err
		}
		name, ok1 := fieldString(reflect.ValueOf(item), namePath)
		id, ok2 := fieldString(reflect.ValueOf(item), idPath)
		if !ok1 || !ok2 {
			continue
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("duplicate name: %s", name)
		}
		out[name] = id
	}
	return out, nil
}

func fieldString(v reflect.Value, path []string) (string, bool) {
	for _, p := range path {
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return "", false
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return "", false
		}
		v = v.FieldByName(p)
		if !v.IsValid() {
			return "", false
		}
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	return fmt.Sprint(v.Interface()), true
}
