package convert

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func Normalize(dst any, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	return normalizeType(reflect.TypeOf(dst), src)
}

func normalizeType(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		return normalizeStruct(typ, src)
	case reflect.Map:
		return normalizeMap(typ, src)
	case reflect.Slice:
		return normalizeSlice(typ, src)
	case reflect.String:
		return normalizeString(typ, src)
	case reflect.Bool:
		return normalizeBool(typ, src)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return normalizeInt(typ, src)
	case reflect.Float32, reflect.Float64:
		return normalizeFloat(typ, src)
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

func normalizeStruct(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
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
			v, err := normalizeType(typ.FieldByIndex(index).Type, v)
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

func normalizeMap(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindMap:
		out := make(map[string]dyn.Value)
		for k, v := range src.MustMap() {
			// Normalize the value according to the map element type.
			v, err := normalizeType(typ.Elem(), v)
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

func normalizeSlice(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case dyn.KindSequence:
		out := make([]dyn.Value, 0, len(src.MustSequence()))
		for _, v := range src.MustSequence() {
			// Normalize the value according to the slice element type.
			v, err := normalizeType(typ.Elem(), v)
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

func normalizeString(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
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

func normalizeBool(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
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

func normalizeInt(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
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

func normalizeFloat(typ reflect.Type, src dyn.Value) (dyn.Value, diag.Diagnostics) {
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
