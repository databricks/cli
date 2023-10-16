package config

import "time"

type Kind int

const (
	// Invalid is the zero value of Kind.
	KindInvalid Kind = iota
	KindMap
	KindSequence
	KindNil
	KindString
	KindBool
	KindInt
	KindFloat
	KindTime
)

func kindOf(v any) Kind {
	switch v.(type) {
	case map[string]Value:
		return KindMap
	case []Value:
		return KindSequence
	case nil:
		return KindNil
	case string:
		return KindString
	case bool:
		return KindBool
	case int, int32, int64:
		return KindInt
	case float32, float64:
		return KindFloat
	case time.Time:
		return KindTime
	default:
		panic("not handled")
	}
}
