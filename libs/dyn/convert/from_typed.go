package convert

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/dyn"
)

type fromTypedOptions int

const (
	// Use the zero value instead of setting zero values to nil. This is useful
	// for types where the zero values and nil are semantically different. That is
	// strings, bools, ints, floats.
	//
	// Note: this is not needed for structs because dyn.NilValue is converted back
	// to a zero value when using the convert.ToTyped function.
	//
	// Values in maps and slices should be set to zero values, and not nil in the
	// dynamic representation.
	includeZeroValues fromTypedOptions = 1 << iota
)

// FromTyped converts changes made in the typed structure w.r.t. the configuration value
// back to the configuration value, retaining existing location information where possible.
func FromTyped(src any, ref dyn.Value) (dyn.Value, error) {
	return fromTyped(src, ref)
}

// Private implementation of FromTyped that allows for additional options not exposed
// in the public API.
func fromTyped(src any, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	srcv := reflect.ValueOf(src)

	// Dereference pointer if necessary
	for srcv.Kind() == reflect.Pointer {
		if srcv.IsNil() {
			return dyn.NilValue, nil
		}
		srcv = srcv.Elem()
	}

	switch srcv.Kind() {
	case reflect.Struct:
		return fromTypedStruct(srcv, ref)
	case reflect.Map:
		return fromTypedMap(srcv, ref)
	case reflect.Slice:
		return fromTypedSlice(srcv, ref)
	case reflect.String:
		return fromTypedString(srcv, ref, options...)
	case reflect.Bool:
		return fromTypedBool(srcv, ref, options...)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return fromTypedInt(srcv, ref, options...)
	case reflect.Float32, reflect.Float64:
		return fromTypedFloat(srcv, ref, options...)
	}

	return dyn.NilValue, fmt.Errorf("unsupported type: %s", srcv.Kind())
}

func fromTypedStruct(src reflect.Value, ref dyn.Value) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindMap, dyn.KindNil:
	default:
		return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	out := make(map[string]dyn.Value)
	info := getStructInfo(src.Type())
	for k, v := range info.FieldValues(src) {
		// Convert the field taking into account the reference value (may be equal to config.NilValue).
		nv, err := fromTyped(v.Interface(), ref.Get(k))
		if err != nil {
			return dyn.Value{}, err
		}

		if nv != dyn.NilValue {
			out[k] = nv
		}
	}

	return dyn.NewValue(out, ref.Location()), nil
}

func fromTypedMap(src reflect.Value, ref dyn.Value) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindMap, dyn.KindNil:
	default:
		return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	// Return nil if the map is nil.
	if src.IsNil() {
		return dyn.NilValue, nil
	}

	out := make(map[string]dyn.Value)
	iter := src.MapRange()
	for iter.Next() {
		k := iter.Key().String()
		v := iter.Value()

		// Convert entry taking into account the reference value (may be equal to dyn.NilValue).
		nv, err := fromTyped(v.Interface(), ref.Get(k), includeZeroValues)
		if err != nil {
			return dyn.Value{}, err
		}

		// Every entry is represented, even if it is a nil.
		// Otherwise, a map with zero-valued structs would yield a nil as well.
		out[k] = nv
	}

	return dyn.NewValue(out, ref.Location()), nil
}

func fromTypedSlice(src reflect.Value, ref dyn.Value) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindSequence, dyn.KindNil:
	default:
		return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	// Return nil if the slice is nil.
	if src.IsNil() {
		return dyn.NilValue, nil
	}

	out := make([]dyn.Value, src.Len())
	for i := 0; i < src.Len(); i++ {
		v := src.Index(i)

		// Convert entry taking into account the reference value (may be equal to dyn.NilValue).
		nv, err := fromTyped(v.Interface(), ref.Index(i), includeZeroValues)
		if err != nil {
			return dyn.Value{}, err
		}

		out[i] = nv
	}

	return dyn.NewValue(out, ref.Location()), nil
}

func fromTypedString(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	switch ref.Kind() {
	case dyn.KindString:
		value := src.String()
		if value == ref.MustString() {
			return ref, nil
		}

		return dyn.V(value), nil
	case dyn.KindNil:
		// This field is not set in the reference. We set it to nil if it's zero
		// valued in the typed representation and the includeZeroValues option is not set.
		if src.IsZero() && !slices.Contains(options, includeZeroValues) {
			return dyn.NilValue, nil
		}
		return dyn.V(src.String()), nil
	}

	return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedBool(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	switch ref.Kind() {
	case dyn.KindBool:
		value := src.Bool()
		if value == ref.MustBool() {
			return ref, nil
		}
		return dyn.V(value), nil
	case dyn.KindNil:
		// This field is not set in the reference. We set it to nil if it's zero
		// valued in the typed representation and the includeZeroValues option is not set.
		if src.IsZero() && !slices.Contains(options, includeZeroValues) {
			return dyn.NilValue, nil
		}
		return dyn.V(src.Bool()), nil
	}

	return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedInt(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	switch ref.Kind() {
	case dyn.KindInt:
		value := src.Int()
		if value == ref.MustInt() {
			return ref, nil
		}
		return dyn.V(value), nil
	case dyn.KindNil:
		// This field is not set in the reference. We set it to nil if it's zero
		// valued in the typed representation and the includeZeroValues option is not set.
		if src.IsZero() && !slices.Contains(options, includeZeroValues) {
			return dyn.NilValue, nil
		}
		return dyn.V(src.Int()), nil
	}

	return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedFloat(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	switch ref.Kind() {
	case dyn.KindFloat:
		value := src.Float()
		if value == ref.MustFloat() {
			return ref, nil
		}
		return dyn.V(value), nil
	case dyn.KindNil:
		// This field is not set in the reference. We set it to nil if it's zero
		// valued in the typed representation and the includeZeroValues option is not set.
		if src.IsZero() && !slices.Contains(options, includeZeroValues) {
			return dyn.NilValue, nil
		}
		return dyn.V(src.Float()), nil
	}

	return dyn.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}
