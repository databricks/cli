// Package sdkconv converts between sdk-go model types and their JSON mirrors
// in libs/sdktypes.
//
// The generated <Type>ToSDK / <Type>FromSDK functions only exist in the
// directions sdk-go itself needs (requests are only encoded, responses only
// decoded). This package converts any mirror/model pair in either direction
// by matching fields by Go name; oneof members, which the mirrors flatten
// into their parent, are matched through the wrapper types registered in
// oneofs.generated.go.
package sdkconv

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// FromSDK converts an sdk-go value (or pointer to one) into a new mirror of
// type M. A nil pointer yields nil.
func FromSDK[M any](src any) (*M, error) {
	return convertTo[M](src)
}

// ToSDK converts a mirror value (or pointer to one) into a new sdk-go value
// of type S. A nil pointer yields nil.
func ToSDK[S any](src any) (*S, error) {
	return convertTo[S](src)
}

func convertTo[T any](src any) (*T, error) {
	sv := reflect.ValueOf(src)
	if !sv.IsValid() || (sv.Kind() == reflect.Pointer && sv.IsNil()) {
		return nil, nil
	}
	var dst T
	if err := convert(reflect.ValueOf(&dst).Elem(), sv); err != nil {
		return nil, err
	}
	return &dst, nil
}

// Convert sets *dst to the converted value of src. dst must be a non-nil pointer.
func Convert(dst, src any) error {
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Pointer || dv.IsNil() {
		return fmt.Errorf("sdkconv: dst must be a non-nil pointer, got %T", dst)
	}
	return convert(dv.Elem(), reflect.ValueOf(src))
}

var wrapperTypes = sync.OnceValue(func() []reflect.Type {
	result := make([]reflect.Type, len(oneofWrappers))
	for i, w := range oneofWrappers {
		result[i] = reflect.TypeOf(w)
	}
	return result
})

var wrappersByInterface sync.Map // reflect.Type -> []reflect.Type

// implementations returns the wrapper pointer types implementing a oneof interface.
func implementations(iface reflect.Type) []reflect.Type {
	if v, ok := wrappersByInterface.Load(iface); ok {
		return v.([]reflect.Type)
	}
	var result []reflect.Type
	for _, w := range wrapperTypes() {
		if w.Implements(iface) {
			result = append(result, w)
		}
	}
	wrappersByInterface.Store(iface, result)
	return result
}

func convert(dst, src reflect.Value) error {
	if !src.IsValid() {
		return nil
	}
	if src.Kind() == reflect.Pointer {
		if src.IsNil() {
			return nil
		}
		if dst.Kind() == reflect.Pointer {
			if dst.Type() == src.Type() {
				dst.Set(src)
				return nil
			}
			elem := reflect.New(dst.Type().Elem())
			if err := convert(elem.Elem(), src.Elem()); err != nil {
				return err
			}
			dst.Set(elem)
			return nil
		}
		return convert(dst, src.Elem())
	}
	if dst.Kind() == reflect.Pointer {
		elem := reflect.New(dst.Type().Elem())
		if err := convert(elem.Elem(), src); err != nil {
			return err
		}
		dst.Set(elem)
		return nil
	}

	if src.Type() == dst.Type() {
		dst.Set(src)
		return nil
	}

	switch {
	case isBasic(src.Kind()) && isBasic(dst.Kind()) && src.Type().ConvertibleTo(dst.Type()):
		dst.Set(src.Convert(dst.Type()))
		return nil
	case src.Kind() == reflect.Struct && dst.Kind() == reflect.Struct:
		return convertStruct(dst, src)
	case src.Kind() == reflect.Slice && dst.Kind() == reflect.Slice:
		if src.IsNil() {
			return nil
		}
		out := reflect.MakeSlice(dst.Type(), src.Len(), src.Len())
		for i := range src.Len() {
			if err := convert(out.Index(i), src.Index(i)); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		dst.Set(out)
		return nil
	case src.Kind() == reflect.Map && dst.Kind() == reflect.Map:
		if src.IsNil() {
			return nil
		}
		out := reflect.MakeMapWithSize(dst.Type(), src.Len())
		iter := src.MapRange()
		for iter.Next() {
			k := reflect.New(dst.Type().Key()).Elem()
			if err := convert(k, iter.Key()); err != nil {
				return err
			}
			v := reflect.New(dst.Type().Elem()).Elem()
			if err := convert(v, iter.Value()); err != nil {
				return fmt.Errorf("[%v]: %w", iter.Key(), err)
			}
			out.SetMapIndex(k, v)
		}
		dst.Set(out)
		return nil
	}

	// Types with a custom JSON encoding, e.g. types.FieldMask (a string on the
	// wire) or json.RawMessage, convert through their JSON form.
	data, err := json.Marshal(src.Interface())
	if err != nil {
		return fmt.Errorf("cannot convert %s to %s: %w", src.Type(), dst.Type(), err)
	}
	if err := json.Unmarshal(data, dst.Addr().Interface()); err != nil {
		return fmt.Errorf("cannot convert %s to %s: %w", src.Type(), dst.Type(), err)
	}
	return nil
}

func isBasic(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func convertStruct(dst, src reflect.Value) error {
	dt, st := dst.Type(), src.Type()
	consumed := map[string]bool{}

	// Mirror -> sdk-go: fill each oneof interface field of dst from whichever
	// flattened member of src is set.
	for i := range dt.NumField() {
		f := dt.Field(i)
		if !f.IsExported() || f.Type.Kind() != reflect.Interface {
			continue
		}
		for _, w := range implementations(f.Type) {
			member := w.Elem().Field(0).Name
			sv := src.FieldByName(member)
			consumed[member] = true
			if !sv.IsValid() || sv.IsZero() {
				continue
			}
			if !dst.Field(i).IsNil() {
				return fmt.Errorf("%s.%s: multiple oneof members set", dt.Name(), f.Name)
			}
			wv := reflect.New(w.Elem())
			if err := convert(wv.Elem().Field(0), sv); err != nil {
				return fmt.Errorf("%s: %w", member, err)
			}
			dst.Field(i).Set(wv)
		}
	}

	for i := range st.NumField() {
		f := st.Field(i)
		if !f.IsExported() {
			continue
		}
		sv := src.Field(i)

		// sdk-go -> mirror: a oneof interface holds *Wrapper{Member: value};
		// the mirror has Member as a field of the parent.
		if f.Type.Kind() == reflect.Interface {
			if sv.IsNil() || sv.Elem().IsNil() {
				continue
			}
			w := sv.Elem().Elem()
			for j := range w.NumField() {
				name := w.Type().Field(j).Name
				df := dst.FieldByName(name)
				if !df.IsValid() {
					return fmt.Errorf("%s has no field %s for oneof %s.%s", dt.Name(), name, st.Name(), f.Name)
				}
				if err := convert(df, w.Field(j)); err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
			}
			continue
		}

		if consumed[f.Name] {
			continue
		}
		df := dst.FieldByName(f.Name)
		if !df.IsValid() {
			if sv.IsZero() {
				continue
			}
			return fmt.Errorf("%s has no field %s (from %s)", dt.Name(), f.Name, st.Name())
		}
		if err := convert(df, sv); err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
	}
	return nil
}
