package dyn

import (
	"fmt"
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
	switch v.(type) {
	case Mapping:
		return KindMap
	case []Value:
		return KindSequence
	case string:
		return KindString
	case bool:
		return KindBool
	case int, int32, int64:
		return KindInt
	case float32, float64:
		return KindFloat
	case Time:
		return KindTime
	case nil:
		return KindNil
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
