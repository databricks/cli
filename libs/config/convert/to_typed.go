package convert

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/config"
)

func ToTyped(dst any, src config.Value) error {
	dstv := reflect.ValueOf(dst)

	// Dereference pointer if necessary
	for dstv.Kind() == reflect.Pointer {
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
	default:
		return fmt.Errorf("unsupported type: %s", dstv.Kind())
	}
}

func toTypedStruct(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindMap:
		info := getStructInfo(dst.Type())
		for k, v := range src.MustMap() {
			index, ok := info.Fields[k]
			if !ok {
				// unknown field
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

			err := ToTyped(f.Addr().Interface(), v)
			if err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedMap(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindMap:
		m := src.MustMap()

		// Always overwrite.
		dst.Set(reflect.MakeMapWithSize(dst.Type(), len(m)))
		for k, v := range m {
			kv := reflect.ValueOf(k)
			vv := reflect.New(dst.Type().Elem())
			err := ToTyped(vv.Interface(), v)
			if err != nil {
				return err
			}
			dst.SetMapIndex(kv, vv.Elem())
		}
		return nil
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedSlice(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindSequence:
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
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedString(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindString:
		dst.SetString(src.MustString())
		return nil
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedBool(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindBool:
		dst.SetBool(src.MustBool())
		return nil
	case config.KindString:
		// See https://github.com/go-yaml/yaml/blob/f6f7691b1fdeb513f56608cd2c32c51f8194bf51/decode.go#L684-L693.
		switch src.MustString() {
		case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
			dst.SetBool(true)
			return nil
		case "n", "N", "no", "No", "NO", "off", "Off", "OFF":
			dst.SetBool(false)
			return nil
		}
		return fmt.Errorf("todo: work with variables")
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedInt(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindInt:
		dst.SetInt(src.MustInt())
		return nil
	case config.KindString:
		return fmt.Errorf("todo: work with variables")
	default:
		return fmt.Errorf("todo")
	}
}

func toTypedFloat(dst reflect.Value, src config.Value) error {
	switch src.Kind() {
	case config.KindFloat:
		dst.SetFloat(src.MustFloat())
		return nil
	case config.KindString:
		return fmt.Errorf("todo: work with variables")
	default:
		return fmt.Errorf("todo")
	}
}
