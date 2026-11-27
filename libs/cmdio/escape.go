package cmdio

import (
	"io"
	"reflect"
	"text/template"

	"github.com/databricks/cli/libs/textutil"
)

// executeTemplate executes t on data after escaping terminal control characters
// in the strings data holds. Data shown through a template mostly comes from an
// API response, and text/template (unlike the JSON encoder) writes strings as
// they are. The data is escaped rather than the rendered output, so the colors
// the template itself adds are kept.
func executeTemplate(t *template.Template, w io.Writer, data any) error {
	return t.Execute(w, escapeControlInData(data))
}

// escapeControlInData returns data with [textutil.EscapeControl] applied to
// every string reachable through exported struct fields, pointers, slices,
// arrays, maps and interfaces. Strings a template gets another way (unexported
// fields, method results, template functions such as b64_decode) are not
// covered. data itself is never modified: the parts that hold an escaped string
// are copied, and data is returned as is when there is nothing to escape, which
// is the common case.
func escapeControlInData(data any) any {
	if data == nil {
		return nil
	}
	escaped, changed := escapeControlInValue(reflect.ValueOf(data))
	if !changed {
		return data
	}
	return escaped.Interface()
}

// escapeControlInValue returns v, or a copy of v when a string in it had to be
// escaped, and whether it made that copy.
func escapeControlInValue(v reflect.Value) (reflect.Value, bool) {
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		escaped := textutil.EscapeControl(s)
		// Escaping only ever makes a string longer.
		if len(escaped) == len(s) {
			return v, false
		}
		return reflect.ValueOf(escaped).Convert(v.Type()), true

	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return v, false
		}
		elem, changed := escapeControlInValue(v.Elem())
		if !changed {
			return v, false
		}
		if v.Kind() == reflect.Pointer {
			p := reflect.New(v.Type().Elem())
			p.Elem().Set(elem)
			return p.Convert(v.Type()), true
		}
		iface := reflect.New(v.Type()).Elem()
		iface.Set(elem)
		return iface, true

	case reflect.Struct:
		var copied reflect.Value
		for i := range v.NumField() {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			field, changed := escapeControlInValue(v.Field(i))
			if !changed {
				continue
			}
			if !copied.IsValid() {
				copied = reflect.New(v.Type()).Elem()
				copied.Set(v)
			}
			copied.Field(i).Set(field)
		}
		return copiedOr(copied, v)

	case reflect.Slice, reflect.Array:
		if !canHoldString(v.Type().Elem().Kind()) {
			return v, false
		}
		var copied reflect.Value
		for i := range v.Len() {
			elem, changed := escapeControlInValue(v.Index(i))
			if !changed {
				continue
			}
			if !copied.IsValid() {
				copied = reflect.New(v.Type()).Elem()
				if v.Kind() == reflect.Slice {
					copied.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
					reflect.Copy(copied, v)
				} else {
					copied.Set(v)
				}
			}
			copied.Index(i).Set(elem)
		}
		return copiedOr(copied, v)

	case reflect.Map:
		var copied reflect.Value
		for iter := v.MapRange(); iter.Next(); {
			key, keyChanged := escapeControlInValue(iter.Key())
			elem, elemChanged := escapeControlInValue(iter.Value())
			if !keyChanged && !elemChanged {
				continue
			}
			if !copied.IsValid() {
				copied = reflect.MakeMapWithSize(v.Type(), v.Len())
				for all := v.MapRange(); all.Next(); {
					copied.SetMapIndex(all.Key(), all.Value())
				}
			}
			if keyChanged {
				copied.SetMapIndex(iter.Key(), reflect.Value{})
			}
			copied.SetMapIndex(key, elem)
		}
		return copiedOr(copied, v)

	default:
		return v, false
	}
}

func copiedOr(copied, original reflect.Value) (reflect.Value, bool) {
	if copied.IsValid() {
		return copied, true
	}
	return original, false
}

// canHoldString reports whether a value of this kind can contain a string, so
// that byte and number slices are not walked element by element.
func canHoldString(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Pointer, reflect.Interface, reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}
