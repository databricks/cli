package template

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/libs/jsonschema"
)

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

func toString(v any, T jsonschema.Type) (string, error) {
	switch T {
	case jsonschema.BooleanType:
		boolVal, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("expected bool, got: %#v", v)
		}
		return strconv.FormatBool(boolVal), nil
	case jsonschema.StringType:
		strVal, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("expected string, got: %#v", v)
		}
		return strVal, nil
	case jsonschema.NumberType:
		floatVal, ok := v.(float64)
		if !ok {
			return "", fmt.Errorf("expected float, got: %#v", v)
		}
		return strconv.FormatFloat(floatVal, 'f', -1, 64), nil
	case jsonschema.IntegerType:
		intVal, err := toInteger(v)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(intVal, 10), nil
	case jsonschema.ArrayType, jsonschema.ObjectType:
		return "", fmt.Errorf("cannot format object of type %s as a string. Value of object: %#v", T, v)
	default:
		if T == "int" {
			return "", fmt.Errorf(`unknown json schema type %q. Please use "integer" instead`, T)
		}
		return "", fmt.Errorf("unknown json schema type: %q", T)
	}
}

func fromString(s string, T jsonschema.Type) (any, error) {
	if T == jsonschema.StringType {
		return s, nil
	}

	// Variables to store value and error from parsing
	var v any
	var err error

	switch T {
	case jsonschema.BooleanType:
		v, err = strconv.ParseBool(s)
	case jsonschema.NumberType:
		v, err = strconv.ParseFloat(s, 32)
	case jsonschema.IntegerType:
		v, err = strconv.ParseInt(s, 10, 64)
	case jsonschema.ArrayType, jsonschema.ObjectType:
		return "", fmt.Errorf("cannot parse string as object of type %s. Value of string: %q", T, s)
	default:
		if T == "int" {
			return "", fmt.Errorf(`unknown json schema type %q. Please use "integer" instead`, T)
		}
		return "", fmt.Errorf("unknown json schema type: %q", T)
	}

	// Return more readable error incase of a syntax error
	if errors.Is(err, strconv.ErrSyntax) {
		return nil, fmt.Errorf("could not parse %q as a %s: %w", s, T, err)
	}
	return v, err
}
