package jsonschema

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// This error indicates an failure to parse a string as a particular JSON schema type.
type parseStringError struct {
	// Expected JSON schema type for the value
	ExpectedType Type

	// The string value that failed to parse
	Value string
}

func (e parseStringError) Error() string {
	return fmt.Sprintf("%q is not a %s", e.Value, e.ExpectedType)
}

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

func toString(v any, T Type) (string, error) {
	switch T {
	case BooleanType:
		boolVal, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("expected bool, got: %#v", v)
		}
		return strconv.FormatBool(boolVal), nil
	case StringType:
		strVal, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("expected string, got: %#v", v)
		}
		return strVal, nil
	case NumberType:
		floatVal, ok := v.(float64)
		if !ok {
			return "", fmt.Errorf("expected float, got: %#v", v)
		}
		return strconv.FormatFloat(floatVal, 'f', -1, 64), nil
	case IntegerType:
		intVal, err := toInteger(v)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(intVal, 10), nil
	case ArrayType, ObjectType:
		return "", fmt.Errorf("cannot format object of type %s as a string. Value of object: %#v", T, v)
	default:
		return "", fmt.Errorf("unknown json schema type: %q", T)
	}
}

func toStringSlice(arr []any, T Type) ([]string, error) {
	var res []string
	for _, v := range arr {
		s, err := toString(v, T)
		if err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

func fromString(s string, T Type) (any, error) {
	if T == StringType {
		return s, nil
	}

	// Variables to store value and error from parsing
	var v any
	var err error

	switch T {
	case BooleanType:
		v, err = strconv.ParseBool(s)
	case NumberType:
		v, err = strconv.ParseFloat(s, 32)
	case IntegerType:
		v, err = strconv.ParseInt(s, 10, 64)
	case ArrayType, ObjectType:
		return "", fmt.Errorf("cannot parse string as object of type %s. Value of string: %q", T, s)
	default:
		return "", fmt.Errorf("unknown json schema type: %q", T)
	}

	// Return more readable error incase of a syntax error
	if errors.Is(err, strconv.ErrSyntax) {
		return nil, parseStringError{
			ExpectedType: T,
			Value:        s,
		}
	}
	return v, err
}

// Error indicates a value entered by the user failed to match the pattern specified
// in the template schema.
type patternMatchError struct {
	// The name of the property that failed to match the pattern
	PropertyName string

	// The value of the property that failed to match the pattern
	PropertyValue any

	// The regex pattern that the property value failed to match
	Pattern string

	// Failure message to display to the user, if specified in the template
	// schema
	FailureMessage string
}

func (e patternMatchError) Error() string {
	// If custom user error message is defined, return error with the custom message
	msg := e.FailureMessage
	if msg == "" {
		msg = "Expected to match regex pattern: " + e.Pattern
	}
	return fmt.Sprintf("invalid value for %s: %q. %s", e.PropertyName, e.PropertyValue, msg)
}

func validatePatternMatch(name string, value any, propertySchema *Schema) error {
	if propertySchema.Pattern == "" {
		// Return early if no pattern is specified
		return nil
	}

	// Expect type of value to be a string
	stringValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid value for %s: %v. Expected a value of type string", name, value)
	}

	match, err := regexp.MatchString(propertySchema.Pattern, stringValue)
	if err != nil {
		return err
	}
	if match {
		// successful match
		return nil
	}

	return patternMatchError{
		PropertyName:   name,
		PropertyValue:  value,
		Pattern:        propertySchema.Pattern,
		FailureMessage: propertySchema.PatternMatchFailureMessage,
	}
}
