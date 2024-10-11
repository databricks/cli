package convert

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// Populate a destination typed value from a source dynamic value.
//
// At any point while walking the destination type tree using
// reflection, if this function sees an exported field with type dyn.Value it
// will populate that field with the appropriate source dynamic value.
// see PR: https://github.com/databricks/cli/pull/1010
func ToTyped(dst any, src dyn.Value) error {
	dstv := reflect.ValueOf(dst)

	// Dereference pointer if necessary
	for dstv.Kind() == reflect.Pointer {
		// If the source value is nil and the destination is a settable pointer,
		// set the destination to nil. Also see `end_to_end_test.go`.
		if dstv.CanSet() && src.Kind() == dyn.KindNil {
			dstv.SetZero()
			return nil
		}
		if dstv.IsNil() {
			dstv.Set(reflect.New(dstv.Type().Elem()))
		}
		dstv = dstv.Elem()
	}

	// Verify that vv is settable.
	if !dstv.CanSet() {
		panic("cannot set destination value")
	}

	switch dstv.Kind() {
	case reflect.Struct:
		return toTypedStruct(dstv, src)
	case reflect.Map:
		return toTypedMap(dstv, src)
	case reflect.Slice:
		return toTypedSlice(dstv, src)
	case reflect.String:
		return toTypedString(dstv, src)
	case reflect.Bool:
		return toTypedBool(dstv, src)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return toTypedInt(dstv, src)
	case reflect.Float32, reflect.Float64:
		return toTypedFloat(dstv, src)
	case reflect.Interface:
		return toTypedInterface(dstv, src)
	}

	return fmt.Errorf("unsupported type: %s", dstv.Kind())
}

func toTypedStruct(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindMap:
		// Zero the destination struct such that fields
		// that aren't present in [src] are cleared.
		dst.SetZero()

		info := getStructInfo(dst.Type())
		for _, pair := range src.MustMap().Pairs() {
			pk := pair.Key
			pv := pair.Value

			index, ok := info.Fields[pk.MustString()]
			if !ok {
				// Ignore unknown fields.
				// A warning will be printed later. See PR #904.
				continue
			}

			// Create intermediate structs embedded as pointer types.
			// Code inspired by [reflect.FieldByIndex] implementation.
			f := dst
			for i, x := range index {
				if i > 0 {
					if f.Kind() == reflect.Pointer {
						if f.IsNil() {
							f.Set(reflect.New(f.Type().Elem()))
						}
						f = f.Elem()
					}
				}
				f = f.Field(x)
			}

			err := ToTyped(f.Addr().Interface(), pv)
			if err != nil {
				return err
			}
		}

		// Populate field(s) for [dyn.Value], if any.
		if info.ValueField != nil {
			vv := dst.FieldByIndex(info.ValueField)
			vv.Set(reflect.ValueOf(src))
		}

		return nil
	case dyn.KindNil:
		dst.SetZero()
		return nil
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a map, found a %s", src.Kind()),
	}
}

func toTypedMap(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindMap:
		m := src.MustMap()

		// Always overwrite.
		dst.Set(reflect.MakeMapWithSize(dst.Type(), m.Len()))
		for _, pair := range m.Pairs() {
			pk := pair.Key
			pv := pair.Value
			kv := reflect.ValueOf(pk.MustString())
			kt := dst.Type().Key()
			vv := reflect.New(dst.Type().Elem())
			err := ToTyped(vv.Interface(), pv)
			if err != nil {
				return err
			}
			dst.SetMapIndex(kv.Convert(kt), vv.Elem())
		}
		return nil
	case dyn.KindNil:
		dst.SetZero()
		return nil
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a map, found a %s", src.Kind()),
	}
}

func toTypedSlice(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindSequence:
		seq := src.MustSequence()

		// Always overwrite.
		dst.Set(reflect.MakeSlice(dst.Type(), len(seq), len(seq)))
		for i := range seq {
			err := ToTyped(dst.Index(i).Addr().Interface(), seq[i])
			if err != nil {
				return err
			}
		}
		return nil
	case dyn.KindNil:
		dst.SetZero()
		return nil
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a sequence, found a %s", src.Kind()),
	}
}

func toTypedString(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindString:
		dst.SetString(src.MustString())
		return nil
	case dyn.KindBool:
		dst.SetString(strconv.FormatBool(src.MustBool()))
		return nil
	case dyn.KindInt:
		dst.SetString(strconv.FormatInt(src.MustInt(), 10))
		return nil
	case dyn.KindFloat:
		dst.SetString(strconv.FormatFloat(src.MustFloat(), 'f', -1, 64))
		return nil
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a string, found a %s", src.Kind()),
	}
}

func toTypedBool(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindBool:
		dst.SetBool(src.MustBool())
		return nil
	case dyn.KindString:
		// See https://github.com/go-yaml/yaml/blob/f6f7691b1fdeb513f56608cd2c32c51f8194bf51/decode.go#L684-L693.
		switch src.MustString() {
		case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON", "true":
			dst.SetBool(true)
			return nil
		case "n", "N", "no", "No", "NO", "off", "Off", "OFF", "false":
			dst.SetBool(false)
			return nil
		}
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a boolean, found a %s", src.Kind()),
	}
}

func toTypedInt(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindInt:
		dst.SetInt(src.MustInt())
		return nil
	case dyn.KindFloat:
		v := src.MustFloat()
		if v == float64(int64(v)) {
			// If the destination is smaller than int64, but the value to set is bigger
			// then destination overflows and is set to -1
			dst.SetInt(int64(src.MustFloat()))
			return nil
		}

		return TypeError{
			value: src,
			msg:   fmt.Sprintf("expected an int, found a %s", src.Kind()),
		}
	case dyn.KindString:
		if i64, err := strconv.ParseInt(src.MustString(), 10, 64); err == nil {
			dst.SetInt(i64)
			return nil
		}
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected an int, found a %s", src.Kind()),
	}
}

func toTypedFloat(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindFloat:
		dst.SetFloat(src.MustFloat())
		return nil
	case dyn.KindString:
		if f64, err := strconv.ParseFloat(src.MustString(), 64); err == nil {
			dst.SetFloat(f64)
			return nil
		}
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a float, found a %s", src.Kind()),
	}
}

func toTypedInterface(dst reflect.Value, src dyn.Value) error {
	if src.Kind() == dyn.KindNil {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	dst.Set(reflect.ValueOf(src.AsAny()))
	return nil
}
