package convert

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/diag"
)

func Normalize(dst any, src config.Value) (config.Value, diag.Diagnostics) {
	return normalizeType(reflect.TypeOf(dst), src)
}

func normalizeType(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
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

	return config.NilValue, diag.Errorf("unsupported type: %s", typ.Kind())
}

func typeMismatch(expected config.Kind, src config.Value) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("expected %s, found %s", expected, src.Kind()),
		Location: src.Location(),
	}
}

func normalizeStruct(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case config.KindMap:
		out := make(map[string]config.Value)
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
				if err.HasError() {
					continue
				}
			}

			out[k] = v
		}

		return config.NewValue(out, src.Location()), diags
	case config.KindNil:
		return src, diags
	}

	return config.NilValue, diags.Append(typeMismatch(config.KindMap, src))
}

func normalizeMap(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case config.KindMap:
		out := make(map[string]config.Value)
		for k, v := range src.MustMap() {
			// Normalize the value according to the map element type.
			v, err := normalizeType(typ.Elem(), v)
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if err.HasError() {
					continue
				}
			}

			out[k] = v
		}

		return config.NewValue(out, src.Location()), diags
	case config.KindNil:
		return src, diags
	}

	return config.NilValue, diags.Append(typeMismatch(config.KindMap, src))
}

func normalizeSlice(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch src.Kind() {
	case config.KindSequence:
		out := make([]config.Value, 0, len(src.MustSequence()))
		for _, v := range src.MustSequence() {
			// Normalize the value according to the slice element type.
			v, err := normalizeType(typ.Elem(), v)
			if err != nil {
				diags = diags.Extend(err)
				// Skip the element if it cannot be normalized.
				if err.HasError() {
					continue
				}
			}

			out = append(out, v)
		}

		return config.NewValue(out, src.Location()), diags
	case config.KindNil:
		return src, diags
	}

	return config.NilValue, diags.Append(typeMismatch(config.KindSequence, src))
}

func normalizeString(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out string

	switch src.Kind() {
	case config.KindString:
		out = src.MustString()
	case config.KindBool:
		out = strconv.FormatBool(src.MustBool())
	case config.KindInt:
		out = strconv.FormatInt(src.MustInt(), 10)
	case config.KindFloat:
		out = strconv.FormatFloat(src.MustFloat(), 'f', -1, 64)
	default:
		return config.NilValue, diags.Append(typeMismatch(config.KindString, src))
	}

	return config.NewValue(out, src.Location()), diags
}

func normalizeBool(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out bool

	switch src.Kind() {
	case config.KindBool:
		out = src.MustBool()
	case config.KindString:
		// See https://github.com/go-yaml/yaml/blob/f6f7691b1fdeb513f56608cd2c32c51f8194bf51/decode.go#L684-L693.
		switch src.MustString() {
		case "true", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
			out = true
		case "false", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
			out = false
		default:
			// Cannot interpret as a boolean.
			return config.NilValue, diags.Append(typeMismatch(config.KindBool, src))
		}
	default:
		return config.NilValue, diags.Append(typeMismatch(config.KindBool, src))
	}

	return config.NewValue(out, src.Location()), diags
}

func normalizeInt(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out int64

	switch src.Kind() {
	case config.KindInt:
		out = src.MustInt()
	case config.KindString:
		var err error
		out, err = strconv.ParseInt(src.MustString(), 10, 64)
		if err != nil {
			return config.NilValue, diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("cannot parse %q as an integer", src.MustString()),
				Location: src.Location(),
			})
		}
	default:
		return config.NilValue, diags.Append(typeMismatch(config.KindInt, src))
	}

	return config.NewValue(out, src.Location()), diags
}

func normalizeFloat(typ reflect.Type, src config.Value) (config.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out float64

	switch src.Kind() {
	case config.KindFloat:
		out = src.MustFloat()
	case config.KindString:
		var err error
		out, err = strconv.ParseFloat(src.MustString(), 64)
		if err != nil {
			return config.NilValue, diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("cannot parse %q as a floating point number", src.MustString()),
				Location: src.Location(),
			})
		}
	default:
		return config.NilValue, diags.Append(typeMismatch(config.KindFloat, src))
	}

	return config.NewValue(out, src.Location()), diags
}
