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
		// return fromTypedFloat(srcv, dst)
	}

	return config.NilValue, fmt.Errorf("unsupported type: %s", srcv.Kind())
}

func fromTypedStruct(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindMap, config.KindNil:
		// Nothing to do.
	default:
		panic("type error")
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

	// what are my options
	// totyped / fromtyped at every mutator boundary
	// pro's -- minimal changes to existing mutators
	// con's -- doesn't hold for all mutators, so we need different interface ANYWAY
	// (e.g. get/set config.Value instances)
	// cons -- lossy (cannot do all to/from conversions, lose location, lose variables)

	// explicit mutator interface
	// pro's -- very clear what's happening
	// cons -- all code + tests need to be changed
	//

	// need an incremental approach
	// thus, we run totyped + fromtyped at mutator boundary
	// can eventually move this into the mutators themselves?
	// can treat the typed structure as read-only, perhaps?
	// can generate wrapper type that exposes a Get + GetValue at every node

}

func fromTypedMap(src reflect.Value, ref config.Value) (config.Value, error) {
	switch ref.Kind() {
	case config.KindMap, config.KindNil:
		// Nothing to do.
	default:
		panic("type error")
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
	switch ref.Kind() {
	case config.KindSequence, config.KindNil:
		// Nothing to do.
	default:
		panic("type error")
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
	if src.IsZero() {
		return config.NilValue, nil
	}

	switch ref.Kind() {
	case config.KindString:
		value := src.String()
		if value == ref.MustString() {
			return ref, nil
		}

		return config.V(value), nil
	case config.KindNil:
		return config.V(src.String()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedBool(src reflect.Value, ref config.Value) (config.Value, error) {
	// Note: this means it's not possible to flip a boolean to false on a typed
	// structure and see it reflected in the dynamic configuration.
	// This case is not handled as is, so we punt on it until the mutaotrs
	// modify the dynamic configuration directly.
	if src.IsZero() {
		return config.NilValue, nil
	}

	switch ref.Kind() {
	case config.KindBool:
		value := src.Bool()
		if value == ref.MustBool() {
			return ref, nil
		}
		return config.V(value), nil
	case config.KindNil:
		return config.V(src.Bool()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}

func fromTypedInt(src reflect.Value, ref config.Value) (config.Value, error) {
	if src.IsZero() {
		return config.NilValue, nil
	}

	switch ref.Kind() {
	case config.KindInt:
		value := src.Int()
		if value == ref.MustInt() {
			return ref, nil
		}
		return config.V(value), nil
	case config.KindNil:
		return config.V(src.Bool()), nil
	}

	return config.Value{}, fmt.Errorf("unhandled type: %s", ref.Kind())
}
