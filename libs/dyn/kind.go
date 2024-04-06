package dyn

import (
	"fmt"
	"reflect"
	"time"
)

type Kind int

const (
	// Invalid is the zero value of Kind.
	KindInvalid Kind = iota
	KindMap
	KindSequence
	KindString
	KindBool
	KindInt
	KindFloat
	KindTime
	KindNil
)

func kindOf(v any) Kind {
	vc := reflect.ValueOf(v)

	for vc.Kind() == reflect.Pointer || vc.Kind() == reflect.Interface {
		if vc.IsNil() {
			return KindNil
		}
		vc = vc.Elem()
	}

	switch vc.Kind() {
	case reflect.Map:
		return KindMap
	case reflect.Slice:
		return KindSequence
	case reflect.String:
		return KindString
	case reflect.Bool:
		return KindBool
	case reflect.Int, reflect.Int32, reflect.Int64:
		return KindInt
	case reflect.Float32, reflect.Float64:
		return KindFloat
	case reflect.TypeOf(time.Time{}).Kind():
		return KindTime
	default:
		panic("not handled")
	}
}

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindMap:
		return "map"
	case KindSequence:
		return "sequence"
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindTime:
		return "time"
	case KindNil:
		return "nil"
	default:
		panic(fmt.Sprintf("invalid kind value: %d", k))
	}
}
