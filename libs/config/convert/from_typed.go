package convert

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/config"
)

// FromTyped converts changes made in the typed structure w.r.t. the configuration value
// back to the configuration value, retaining existing location information where possible.
func FromTyped(src any, ref config.Value) (config.Value, error) {
	srcv := reflect.ValueOf(src)

	// Dereference pointer if necessary
	for srcv.Kind() == reflect.Pointer {
		if srcv.IsNil() {
			return config.NilValue, nil
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
		return fromTypedString(srcv, ref)
	case reflect.Bool:
		return fromTypedBool(srcv, ref)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return fromTypedInt(srcv, ref)
	case reflect.Float32, reflect.Float64:
		return fromTypedFloat(srcv, ref)
	}

	return config.NilValue, fmt.Errorf("unsupported type: %s", srcv.Kind())
}

func fromTypedStruct(src reflect.Value, ref config.Value) (config.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case config.KindMap, config.KindNil:
	default:
		return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	out := make(map[string]config.Value)
	info := getStructInfo(src.Type())
	for k, v := range info.FieldValues(src) {
		// Convert the field taking into account the reference value (may be equal to config.NilValue).
		nv, err := FromTyped(v.Interface(), ref.Get(k))
		if err != nil {
			return config.Value{}, err
		}

		if nv != config.NilValue {
			out[k] = nv
		}
	}

	// If the struct was equal to its zero value, emit a nil.
	if len(out) == 0 {
		return config.NilValue, nil
	}

	return config.NewValue(out, ref.Location()), nil
}

func fromTypedMap(src reflect.Value, ref config.Value) (config.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case config.KindMap, config.KindNil:
	default:
		return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	out := make(map[string]config.Value)
	iter := src.MapRange()
	for iter.Next() {
		k := iter.Key().String()
		v := iter.Value()

		// Convert entry taking into account the reference value (may be equal to config.NilValue).
		nv, err := FromTyped(v.Interface(), ref.Get(k))
		if err != nil {
			return config.Value{}, err
		}

		// Every entry is represented, even if it is a nil.
		// Otherwise, a map with zero-valued structs would yield a nil as well.
		out[k] = nv
	}

	// If the map has no entries, emit a nil.
	if len(out) == 0 {
		return config.NilValue, nil
	}

	return config.NewValue(out, ref.Location()), nil
}

func fromTypedSlice(src reflect.Value, ref config.Value) (config.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case config.KindSequence, config.KindNil:
	default:
		return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	out := make([]config.Value, src.Len())
	for i := 0; i < src.Len(); i++ {
		v := src.Index(i)

		// Convert entry taking into account the reference value (may be equal to config.NilValue).
		nv, err := FromTyped(v.Interface(), ref.Index(i))
		if err != nil {
			return config.Value{}, err
		}

		out[i] = nv
	}

	// If the slice has no entries, emit a nil.
	if len(out) == 0 {
		return config.NilValue, nil
	}

	return config.NewValue(out, ref.Location()), nil
}

func fromTypedString(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindString:
		value := src.String()
		if value == ref.MustString() {
			return ref, nil
		}

		return config.V(value), nil
	case config.KindNil:
		// This field is not set in the reference, so we only include it if it has a non-zero value.
		// Otherwise, we would always include all zero valued fields.
		if src.IsZero() {
			return config.NilValue, nil
		}
		return config.V(src.String()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedBool(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindBool:
		value := src.Bool()
		if value == ref.MustBool() {
			return ref, nil
		}
		return config.V(value), nil
	case config.KindNil:
		// This field is not set in the reference, so we only include it if it has a non-zero value.
		// Otherwise, we would always include all zero valued fields.
		if src.IsZero() {
			return config.NilValue, nil
		}
		return config.V(src.Bool()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedInt(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindInt:
		value := src.Int()
		if value == ref.MustInt() {
			return ref, nil
		}
		return config.V(value), nil
	case config.KindNil:
		// This field is not set in the reference, so we only include it if it has a non-zero value.
		// Otherwise, we would always include all zero valued fields.
		if src.IsZero() {
			return config.NilValue, nil
		}
		return config.V(src.Int()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedFloat(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindFloat:
		value := src.Float()
		if value == ref.MustFloat() {
			return ref, nil
		}
		return config.V(value), nil
	case config.KindNil:
		// This field is not set in the reference, so we only include it if it has a non-zero value.
		// Otherwise, we would always include all zero valued fields.
		if src.IsZero() {
			return config.NilValue, nil
		}
		return config.V(src.Float()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}
