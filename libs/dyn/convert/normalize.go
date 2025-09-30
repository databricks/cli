package convert

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
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

	return n.normalizeType(reflect.TypeOf(dst), src, []reflect.Type{}, dyn.EmptyPath)
}

func (n normalizeOptions) normalizeType(typ reflect.Type, src dyn.Value, seen []reflect.Type, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		return n.normalizeStruct(typ, src, append(seen, typ), path)
	case reflect.Map:
		return n.normalizeMap(typ, src, append(seen, typ), path)
	case reflect.Slice:
		return n.normalizeSlice(typ, src, append(seen, typ), path)
	case reflect.String:
		return n.normalizeString(typ, src, path)
	case reflect.Bool:
		return n.normalizeBool(typ, src, path)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return n.normalizeInt(typ, src, path)
	case reflect.Float32, reflect.Float64:
		return n.normalizeFloat(typ, src, path)
	case reflect.Interface:
		return n.normalizeInterface(typ, src, path)
	}

	return dyn.InvalidValue, diag.Errorf("unsupported type: %s", typ.Kind())
}

func nullWarning(expected dyn.Kind, src dyn.Value, path dyn.Path) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("expected a %s value, found null", expected),
		Locations: []dyn.Location{src.Location()},
		Paths:     []dyn.Path{path},
	}
}

func typeMismatch(expected dyn.Kind, src dyn.Value, path dyn.Path) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("expected %s, found %s", expected, src.Kind()),
		Locations: []dyn.Location{src.Location()},
		Paths:     []dyn.Path{path},
	}
}

func (n normalizeOptions) normalizeStruct(typ reflect.Type, src dyn.Value, seen []reflect.Type, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindMap:
		out := dyn.NewMapping()
		info := getStructInfo(typ)
		for _, pair := range src.MustMap().Pairs() {
			pk := pair.Key
			pv := pair.Value

			index, ok := info.Fields[pk.MustString()]
			if !ok {
				if !pv.IsAnchor() {
					diags = diags.Append(diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "unknown field: " + pk.MustString(),
						// Show all locations the unknown field is defined at.
						Locations: pk.Locations(),
						Paths:     []dyn.Path{path},
					})
				}
				continue
			}

			// Normalize the value according to the field type.
			nv, err := n.normalizeType(typ.FieldByIndex(index).Type, pv, seen, path.Append(dyn.Key(pk.MustString())))
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !nv.IsValid() {
					continue
				}
			}

			out.SetLoc(pk.MustString(), pk.Locations(), nv)
		}

		// Return the normalized value if missing fields are not included.
		if !n.includeMissingFields {
			return dyn.NewValue(out, src.Locations()), diags
		}

		// Populate missing fields with their zero values.
		for k, index := range info.Fields {
			if _, ok := out.GetByString(k); ok {
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
				v, _ = n.normalizeType(ftyp, dyn.V(map[string]dyn.Value{}), seen, path.Append(dyn.Key(k)))
			case reflect.Slice:
				v, _ = n.normalizeType(ftyp, dyn.V([]dyn.Value{}), seen, path.Append(dyn.Key(k)))
			case reflect.String:
				v, _ = n.normalizeType(ftyp, dyn.V(""), seen, path.Append(dyn.Key(k)))
			case reflect.Bool:
				v, _ = n.normalizeType(ftyp, dyn.V(false), seen, path.Append(dyn.Key(k)))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				v, _ = n.normalizeType(ftyp, dyn.V(int64(0)), seen, path.Append(dyn.Key(k)))
			case reflect.Float32, reflect.Float64:
				v, _ = n.normalizeType(ftyp, dyn.V(float64(0)), seen, path.Append(dyn.Key(k)))
			default:
				// Skip fields for which we do not have a natural [dyn.Value] equivalent.
				// For example, we don't handle reflect.Complex* and reflect.Uint* types.
				continue
			}
			if v.IsValid() {
				out.SetLoc(k, nil, v)
			}
		}

		return dyn.NewValue(out, src.Locations()), diags
	case dyn.KindNil:
		return src, diags

	case dyn.KindString:
		// Return verbatim if it's a pure variable reference.
		if dynvar.IsPureVariableReference(src.MustString()) {
			return src, nil
		}
	}

	// Cannot interpret as a struct.
	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindMap, src, path))
}

func (n normalizeOptions) normalizeMap(typ reflect.Type, src dyn.Value, seen []reflect.Type, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindMap:
		out := dyn.NewMapping()
		for _, pair := range src.MustMap().Pairs() {
			pk := pair.Key
			pv := pair.Value

			// Normalize the value according to the map element type.
			nv, err := n.normalizeType(typ.Elem(), pv, seen, path.Append(dyn.Key(pk.MustString())))
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !nv.IsValid() {
					continue
				}
			}

			out.SetLoc(pk.MustString(), pk.Locations(), nv)
		}

		return dyn.NewValue(out, src.Locations()), diags
	case dyn.KindNil:
		return src, diags

	case dyn.KindString:
		// Return verbatim if it's a pure variable reference.
		if dynvar.IsPureVariableReference(src.MustString()) {
			return src, nil
		}
	}

	// Cannot interpret as a map.
	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindMap, src, path))
}

func (n normalizeOptions) normalizeSlice(typ reflect.Type, src dyn.Value, seen []reflect.Type, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindSequence:
		out := make([]dyn.Value, 0, len(src.MustSequence()))
		for _, v := range src.MustSequence() {
			// Normalize the value according to the slice element type.
			v, err := n.normalizeType(typ.Elem(), v, seen, path.Append(dyn.Index(len(out))))
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if !v.IsValid() {
					continue
				}
			}

			out = append(out, v)
		}

		return dyn.NewValue(out, src.Locations()), diags
	case dyn.KindNil:
		return src, diags

	case dyn.KindString:
		// Return verbatim if it's a pure variable reference.
		if dynvar.IsPureVariableReference(src.MustString()) {
			return src, nil
		}
	}

	// Cannot interpret as a slice.
	return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindSequence, src, path))
}

func (n normalizeOptions) normalizeString(typ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
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
	case dyn.KindTime:
		out = src.MustTime().String()
	case dyn.KindNil:
		// Return a warning if the field is present but has a null value.
		return dyn.InvalidValue, diags.Append(nullWarning(dyn.KindString, src, path))
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindString, src, path))
	}

	return dyn.NewValue(out, src.Locations()), diags
}

func (n normalizeOptions) normalizeBool(typ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
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
			// Return verbatim if it's a pure variable reference.
			if dynvar.IsPureVariableReference(src.MustString()) {
				return src, nil
			}

			// Cannot interpret as a boolean.
			return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindBool, src, path))
		}
	case dyn.KindNil:
		// Return a warning if the field is present but has a null value.
		return dyn.InvalidValue, diags.Append(nullWarning(dyn.KindBool, src, path))
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindBool, src, path))
	}

	return dyn.NewValue(out, src.Locations()), diags
}

func (n normalizeOptions) normalizeInt(typ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out int64

	switch src.Kind() {
	case dyn.KindInt:
		out = src.MustInt()
	case dyn.KindFloat:
		out = int64(src.MustFloat())
		if src.MustFloat() != float64(out) {
			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf(`cannot accurately represent "%g" as integer due to precision loss`, src.MustFloat()),
				Locations: []dyn.Location{src.Location()},
				Paths:     []dyn.Path{path},
			})
		}
	case dyn.KindString:
		var err error
		out, err = strconv.ParseInt(src.MustString(), 10, 64)
		if err != nil {
			// Return verbatim if it's a pure variable reference.
			if dynvar.IsPureVariableReference(src.MustString()) {
				return src, nil
			}

			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf("cannot parse %q as an integer", src.MustString()),
				Locations: []dyn.Location{src.Location()},
				Paths:     []dyn.Path{path},
			})
		}
	case dyn.KindNil:
		// Return a warning if the field is present but has a null value.
		return dyn.InvalidValue, diags.Append(nullWarning(dyn.KindInt, src, path))
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindInt, src, path))
	}

	return dyn.NewValue(out, src.Locations()), diags
}

func (n normalizeOptions) normalizeFloat(typ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out float64

	switch src.Kind() {
	case dyn.KindFloat:
		out = src.MustFloat()
	case dyn.KindInt:
		out = float64(src.MustInt())
		if src.MustInt() != int64(out) {
			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf(`cannot accurately represent "%d" as floating point number due to precision loss`, src.MustInt()),
				Locations: []dyn.Location{src.Location()},
				Paths:     []dyn.Path{path},
			})
		}
	case dyn.KindString:
		var err error
		out, err = strconv.ParseFloat(src.MustString(), 64)
		if err != nil {
			// Return verbatim if it's a pure variable reference.
			if dynvar.IsPureVariableReference(src.MustString()) {
				return src, nil
			}

			return dyn.InvalidValue, diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf("cannot parse %q as a floating point number", src.MustString()),
				Locations: []dyn.Location{src.Location()},
				Paths:     []dyn.Path{path},
			})
		}
	case dyn.KindNil:
		// Return a warning if the field is present but has a null value.
		return dyn.InvalidValue, diags.Append(nullWarning(dyn.KindFloat, src, path))
	default:
		return dyn.InvalidValue, diags.Append(typeMismatch(dyn.KindFloat, src, path))
	}

	return dyn.NewValue(out, src.Locations()), diags
}

func (n normalizeOptions) normalizeInterface(_ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	// Deal with every [dyn.Kind] here to ensure completeness.
	switch src.Kind() {
	case dyn.KindMap:
		// Fall through
	case dyn.KindSequence:
		// Fall through
	case dyn.KindString:
		// Fall through
	case dyn.KindBool:
		// Fall through
	case dyn.KindInt:
		// Fall through
	case dyn.KindFloat:
		// Fall through
	case dyn.KindTime:
		// Conversion of a time value to an interface{}.
		// The [dyn.Value.AsAny] equivalent for this kind is the [time.Time] struct.
		// If we convert to a typed representation and back again, we cannot distinguish
		// a [time.Time] struct from any other struct.
		//
		// Therefore, we normalize the time value to a string.
		return dyn.NewValue(src.MustTime().String(), src.Locations()), nil
	case dyn.KindNil:
		// Fall through
	default:
		return dyn.InvalidValue, diag.Errorf("unsupported kind: %s", src.Kind())
	}

	return src, nil
}
