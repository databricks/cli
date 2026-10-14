package cmdio

import "reflect"

// templateData returns a copy of v in which nil pointers to scalars point to
// zero values. API types represent optional fields as pointers, and
// text/template prints a nil one as "<nil>"; templates were written for value
// fields that print as "" or 0, so this keeps table output unchanged.
func templateData(v any) any {
	if v == nil {
		return nil
	}
	return fillNilScalars(reflect.ValueOf(v)).Interface()
}

func isScalar(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func fillNilScalars(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			if isScalar(v.Type().Elem().Kind()) {
				return reflect.New(v.Type().Elem())
			}
			return v
		}
		if v.Elem().Kind() != reflect.Struct {
			return v
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(fillNilScalars(v.Elem()))
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := range v.NumField() {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			out.Field(i).Set(fillNilScalars(v.Field(i)))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		switch v.Type().Elem().Kind() {
		case reflect.Struct, reflect.Pointer, reflect.Interface:
		default:
			return v
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(fillNilScalars(v.Index(i)))
		}
		return out
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		filled := fillNilScalars(v.Elem())
		out := reflect.New(v.Type()).Elem()
		out.Set(filled)
		return out
	}
	return v
}
