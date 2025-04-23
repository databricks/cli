package convert

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type fromTypedOptions int

const (
	// If this flag is set, zero values in the typed representation are resolved to
	// the equivalent zero value in the dynamic representation.
	// If it is not set, zero values resolve to [dyn.NilValue].
	//
	// This flag exists to reconcile default values in Go being zero values with values
	// being intentionally set to their zero value. We capture zero values in the dynamic
	// configuration if they are 1) behind a pointer, 2) a map value, 3) a slice element,
	// in the typed configuration.
	includeZeroValues fromTypedOptions = 1 << iota
)

// FromTyped converts changes made in the typed structure w.r.t. the configuration value
// back to the configuration value, retaining existing location information where possible.
//
// It uses the reference value both for location information and to determine if the typed
// value was changed or not. For example, if a struct-by-value field is nil in the reference
// it will be zero-valued in the typed configuration. If it remains zero-valued, this
// this function will still emit a nil value in the dynamic representation.
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
			return dyn.NilValue.WithLocations(ref.Locations()), nil
		}
		srcv = srcv.Elem()

		// If a pointer to a type points to a zero value, we should include
		// that zero value in the dynamic representation.
		// This is because by default a pointer is nil in Go, and it not being nil
		// indicates its value was intentionally set to zero.
		if !slices.Contains(options, includeZeroValues) {
			options = append(options, includeZeroValues)
		}
	}

	var v dyn.Value
	var err error
	switch srcv.Kind() {
	case reflect.Struct:
		v, err = fromTypedStruct(srcv, ref, options...)
	case reflect.Map:
		v, err = fromTypedMap(srcv, ref)
	case reflect.Slice:
		v, err = fromTypedSlice(srcv, ref)
	case reflect.String:
		v, err = fromTypedString(srcv, ref, options...)
	case reflect.Bool:
		v, err = fromTypedBool(srcv, ref, options...)
	case reflect.Int, reflect.Int32, reflect.Int64:
		v, err = fromTypedInt(srcv, ref, options...)
	case reflect.Float32, reflect.Float64:
		v, err = fromTypedFloat(srcv, ref, options...)
	case reflect.Invalid:
		// If the value is untyped and not set (e.g. any type with nil value), we return nil.
		v, err = dyn.NilValue, nil
	default:
		return dyn.InvalidValue, fmt.Errorf("unsupported type: %s", srcv.Kind())
	}

	// Ensure the location metadata is retained.
	if err != nil {
		return dyn.InvalidValue, err
	}
	return v.WithLocations(ref.Locations()), err
}

func fromTypedStruct(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	case dyn.KindMap, dyn.KindNil:
	default:
		return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	refm, _ := ref.AsMap()
	out := dyn.NewMapping()
	info := getStructInfo(src.Type())
	for k, v := range info.FieldValues(src) {
		pair, ok := refm.GetPairByString(k)
		refloc := pair.Key.Locations()
		refv := pair.Value

		// Use nil reference if there is no reference for this key
		if !ok {
			refloc = nil
			refv = dyn.NilValue
		}

		var options []fromTypedOptions
		if v.Kind() == reflect.Interface {
			options = append(options, includeZeroValues)
		}

		// Convert the field taking into account the reference value (may be equal to config.NilValue).
		nv, err := fromTyped(v.Interface(), refv, options...)
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Either if the key was set in the reference or the field is not zero-valued, we include it.
		if ok || nv.Kind() != dyn.KindNil {
			out.SetLoc(k, refloc, nv)
		}
	}

	// Return the new mapping if:
	// 1. The mapping has entries (i.e. the struct was not empty).
	// 2. The reference is a map (i.e. the struct was and still is empty).
	// 3. The "includeZeroValues" option is set (i.e. the struct is a non-nil pointer).
	if out.Len() > 0 || ref.Kind() == dyn.KindMap || slices.Contains(options, includeZeroValues) {
		return dyn.V(out), nil
	}

	// Otherwise, return nil.
	return dyn.NilValue, nil
}

func fromTypedMap(src reflect.Value, ref dyn.Value) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	case dyn.KindMap, dyn.KindNil:
	default:
		return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	// Return nil if the map is nil.
	if src.IsNil() {
		return dyn.NilValue, nil
	}

	refm, _ := ref.AsMap()
	out := dyn.NewMapping()
	iter := src.MapRange()
	for iter.Next() {
		k := iter.Key().String()
		v := iter.Value()
		pair, ok := refm.GetPairByString(k)
		refloc := pair.Key.Locations()
		refv := pair.Value

		// Use nil reference if there is no reference for this key
		if !ok {
			refloc = nil
			refv = dyn.NilValue
		}

		// Convert entry taking into account the reference value (may be equal to dyn.NilValue).
		nv, err := fromTyped(v.Interface(), refv, includeZeroValues)
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Every entry is represented, even if it is a nil.
		// Otherwise, a map with zero-valued structs would yield a nil as well.
		out.SetLoc(k, refloc, nv)
	}

	return dyn.V(out), nil
}

func fromTypedSlice(src reflect.Value, ref dyn.Value) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	case dyn.KindSequence, dyn.KindNil:
	default:
		return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
	}

	// Return nil if the slice is nil.
	if src.IsNil() {
		return dyn.NilValue, nil
	}

	out := make([]dyn.Value, src.Len())
	for i := range src.Len() {
		v := src.Index(i)
		refv := ref.Index(i)

		// Use nil reference if there is no reference for this index.
		if refv.Kind() == dyn.KindInvalid {
			refv = dyn.NilValue
		}

		// Convert entry taking into account the reference value (may be equal to dyn.NilValue).
		nv, err := fromTyped(v.Interface(), refv, includeZeroValues)
		if err != nil {
			return dyn.InvalidValue, err
		}

		out[i] = nv
	}

	return dyn.V(out), nil
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

	return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
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
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	}

	return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
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
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	}

	return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
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
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	}

	return dyn.InvalidValue, fmt.Errorf("unhandled type: %s", ref.Kind())
}
