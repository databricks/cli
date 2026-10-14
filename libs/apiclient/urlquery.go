package apiclient

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// URLQuery encodes the fields of a request struct that carry `url` tags into
// query parameters, the way the previous SDK encoded GET/DELETE requests:
// `url:"-"` and untagged fields are skipped, `omitempty` drops zero values and
// nested structs use dotted keys. Pass the result as Do's request argument for
// GET, DELETE and HEAD.
func URLQuery(v any) url.Values {
	q := url.Values{}
	encodeStruct(q, "", reflect.ValueOf(v))
	return q
}

// queryEncoder is implemented by types with a custom query encoding, such as
// field masks.
type queryEncoder interface {
	EncodeValues(key string, v *url.Values) error
}

func encodeStruct(q url.Values, prefix string, v reflect.Value) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup("url")
		if !ok || tag == "-" || !field.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			name = field.Name
		}
		if prefix != "" {
			name = prefix + "." + name
		}
		fv := v.Field(i)
		if strings.Contains(opts, "omitempty") && fv.IsZero() {
			continue
		}
		encodeValue(q, name, fv)
	}
}

func encodeValue(q url.Values, key string, v reflect.Value) {
	if v.CanAddr() {
		if enc, ok := v.Addr().Interface().(queryEncoder); ok {
			_ = enc.EncodeValues(key, &q)
			return
		}
	}
	if enc, ok := v.Interface().(queryEncoder); ok {
		_ = enc.EncodeValues(key, &q)
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			encodeValue(q, key, v.Elem())
		}
	case reflect.Struct:
		encodeStruct(q, key, v)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			encodeValue(q, key, v.Index(i))
		}
	default:
		q.Add(key, fmt.Sprint(v.Interface()))
	}
}
