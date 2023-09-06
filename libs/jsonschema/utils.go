package jsonschema

import "fmt"

// TODO: this code is copied from libs/template/utils.go. Remove it from
// the template package once validation is moved to the json schema package

// function to check whether a float value represents an integer
func isIntegerValue(v float64) bool {
	return v == float64(int64(v))
}

func toInteger(v any) (int64, error) {
	switch typedVal := v.(type) {
	// cast float to int
	case float32:
		if !isIntegerValue(float64(typedVal)) {
			return 0, fmt.Errorf("expected integer value, got: %v", v)
		}
		return int64(typedVal), nil
	case float64:
		if !isIntegerValue(typedVal) {
			return 0, fmt.Errorf("expected integer value, got: %v", v)
		}
		return int64(typedVal), nil

	// pass through common integer cases
	case int:
		return int64(typedVal), nil
	case int32:
		return int64(typedVal), nil
	case int64:
		return typedVal, nil

	default:
		return 0, fmt.Errorf("cannot convert %#v to an integer", v)
	}
}
