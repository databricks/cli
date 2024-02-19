package convert

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// NormalizeOption is the type for options that can be passed to Normalize.
type NormalizeOption int

const (
	// IncludeMissingFields causes the normalization to include fields that defined on the given
	// type but are missing in the source value. They are included with their zero values.
	IncludeMissingFields NormalizeOption = iota
)

type normalizeOptions struct {
	includeMissingFields bool
}

func Normalize(dst any, src dyn.Value, opts ...NormalizeOption) (dyn.Value, diag.Diagnostics) {
	var n normalizeOptions
	for _, opt := range opts {
		switch opt {
		case IncludeMissingFields:
			n.includeMissingFields = true
		}
	}

	return n.normalizeType(reflect.TypeOf(dst), src, []reflect.Type{})
}

func (n normalizeOptions) normalizeType(typ reflect.Type, src dyn.Value, seen []reflect.Type) (dyn.Value, diag.Diagnostics) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		return n.normalizeStruct(typ, src, append(seen, typ))
	case reflect.Map:
		return n.normalizeMap(typ, src, append(seen, typ))
	case reflect.Slice:
		return n.normalizeSlice(typ, src, append(seen, typ))
	case reflect.String:
		return n.normalizeString(typ, src)
	case reflect.Bool:
		return n.normalizeBool(typ, src)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return n.normalizeInt(typ, src)
	case reflect.Float32, reflect.Float64:
		return n.normalizeFloat(typ, src)
	}

	return dyn.InvalidValue, diag.Errorf("unsupported type: %s", typ.Kind())
}

func typeMismatch(expected dyn.Kind, src dyn.Value) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("expected %s, found %s", expected, src.Kind()),
		Location: src.Location(),
	}
}

func (n normalizeOptions) normalizeStruct(typ reflect.Type, src dyn.Value, seen []reflect.Type) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindMap:
		out := make(map[string]dyn.Value)
		info := getStructInfo(typ)
		for k, v := range src.MustMap() {
			index, ok := info.Fields[k]
			if !ok {
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("unknown field: %s", k),
					Location: src.Location(),
				})
				continue
			}

			// Normalize the value according to the field type.
			v, err := n.normalizeType(typ.FieldByIndex(index).Type, v, seen)
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !v.IsValid() {
					continue
				}
			}

			out[k] = v
		}

		// Return the normalized value if missing fields are not included.
		if !n.includeMissingFields {
			return dyn.NewValue(out, src.Location()), diags
		}

		// Populate missing fields with their zero values.
		for k, index := range info.Fields {
			if _, ok := out[k]; ok {
				continue
			}

			// Optionally dereference pointers to get the underlying field type.
			ftyp := typ.FieldByIndex(index).Type
			for ftyp.Kind() == reflect.Pointer {
				ftyp = ftyp.Elem()
			}

			// Skip field if we have already seen its type to avoid infinite recursion
			// when filling in the zero value of a recursive type.
			if slices.Contains(seen, ftyp) {
				continue
			}

			var v dyn.Value
			switch ftyp.Kind() {
			case reflect.Struct, reflect.Map:
				v, _ = n.normalizeType(ftyp, dyn.V(map[string]dyn.Value{}), seen)
			case reflect.Slice:
				v, _ = n.normalizeType(ftyp, dyn.V([]dyn.Value{}), seen)
			case reflect.String:
				v, _ = n.normalizeType(ftyp, dyn.V(""), seen)
			case reflect.Bool:
				v, _ = n.normalizeType(ftyp, dyn.V(false), seen)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				v, _ = n.normalizeType(ftyp, dyn.V(int64(0)), seen)
			case reflect.Float32, reflect.Float64:
				v, _ = n.normalizeType(ftyp, dyn.V(float64(0)), seen)
			default:
				// Skip fields for which we do not have a natural [dyn.Value] equivalent.
				// For example, we don't handle reflect.Complex* and reflect.Uint* types.
				continue
			}
			if v.IsValid() {
				out[k] = v
			}
		}

		return dyn.NewValue(out, src.Location()), diags
	case dyn.KindNil:
		return src, diags
	}

	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindMap, src))
}

func (n normalizeOptions) normalizeMap(typ reflect.Type, src dyn.Value, seen []reflect.Type) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindMap:
		out := make(map[string]dyn.Value)
		for k, v := range src.MustMap() {
			// Normalize the value according to the map element type.
			v, err := n.normalizeType(typ.Elem(), v, seen)
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !v.IsValid() {
					continue
				}
			}

			out[k] = v
		}

		return dyn.NewValue(out, src.Location()), diags
	case dyn.KindNil:
		return src, diags
	}

	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindMap, src))
}

func (n normalizeOptions) normalizeSlice(typ reflect.Type, src dyn.Value, seen []reflect.Type) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindSequence:
		out := make([]dyn.Value, 0, len(src.MustSequence()))
		for _, v := range src.MustSequence() {
			// Normalize the value according to the slice element type.
			v, err := n.normalizeType(typ.Elem(), v, seen)
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !v.IsValid() {
					continue
				}
			}

			out = append(out, v)
		}

		return dyn.NewValue(out, src.Location()), diags
	case dyn.KindNil:
		return src, diags
	}

	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindSequence, src))
}

func (n normalizeOptions) normalizeString(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out string

	switch src.Kind() {
	case dyn.KindString:
		out = src.MustString()
	case dyn.KindBool:
		out = strconv.FormatBool(src.MustBool())
	case dyn.KindInt:
		out = strconv.FormatInt(src.MustInt(), 10)
	case dyn.KindFloat:
		out = strconv.FormatFloat(src.MustFloat(), 'f', -1, 64)
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindString, src))
	}

	return dyn.NewValue(out, src.Location()), diags
}

func (n normalizeOptions) normalizeBool(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out bool

	switch src.Kind() {
	case dyn.KindBool:
		out = src.MustBool()
	case dyn.KindString:
		// See https://github.com/go-yaml/yaml/blob/f6f7691b1fdeb513f56608cd2c32c51f8194bf51/decode.go#L684-L693.
		switch src.MustString() {
		case "true", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
			out = true
		case "false", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
			out = false
		default:
			// Cannot interpret as a boolean.
			return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindBool, src))
		}
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindBool, src))
	}

	return dyn.NewValue(out, src.Location()), diags
}

func (n normalizeOptions) normalizeInt(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out int64

	switch src.Kind() {
	case dyn.KindInt:
		out = src.MustInt()
	case dyn.KindString:
		var err error
		out, err = strconv.ParseInt(src.MustString(), 10, 64)
		if err != nil {
			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("cannot parse %q as an integer", src.MustString()),
				Location: src.Location(),
			})
		}
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindInt, src))
	}

	return dyn.NewValue(out, src.Location()), diags
}

func (n normalizeOptions) normalizeFloat(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out float64

	switch src.Kind() {
	case dyn.KindFloat:
		out = src.MustFloat()
	case dyn.KindString:
		var err error
		out, err = strconv.ParseFloat(src.MustString(), 64)
		if err != nil {
			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("cannot parse %q as a floating point number", src.MustString()),
				Location: src.Location(),
			})
		}
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindFloat, src))
	}

	return dyn.NewValue(out, src.Location()), diags
}
