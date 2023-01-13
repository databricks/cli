package schema

import (
	"fmt"
	"reflect"
	"strings"
)

const MaxHistoryOccurances = 3

// TODO: should omit empty denote non required fields in the json schema?
type Schema struct {
	Type                  JsType               `json:"type"`
	Properities           map[string]*Property `json:"properities,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperities,omitempty"`
}

type Property struct {
	// TODO: Add a enum for json types
	Type                  JsType               `json:"type"`
	Items                 *Item                `json:"item,omitempty"`
	Properities           map[string]*Property `json:"properities,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperities,omitempty"`
}

type Item struct {
	Type JsType `json:"type"`
}

func NewSchema(goType reflect.Type) (*Schema, error) {
	rootProp, err := properity(goType)
	if err != nil {
		return nil, err
	}
	return &Schema{
		Type:                  rootProp.Type,
		Properities:           rootProp.Properities,
		AdditionalProperities: rootProp.AdditionalProperities,
	}, nil
}

type JsType string

const (
	Invalid JsType = "invalid"
	Boolean        = "boolean"
	String         = "string"
	Number         = "number"
	Object         = "object"
	Array          = "array"
)

func jsType(goType reflect.Type) (JsType, error) {
	switch goType.Kind() {
	case reflect.Bool:
		return Boolean, nil
	case reflect.String:
		return String, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return Number, nil
	case reflect.Struct:
		return Object, nil
	// TODO: add support for pattern properities to account for maps
	case reflect.Map:
		if goType.Key().Kind() != reflect.String {
			return Invalid, fmt.Errorf("only strings map keys are valid. key type: ", goType.Key().Kind())
		}
		return Object, nil
	case reflect.Array, reflect.Slice:
		return Array, nil
	default:
		return Invalid, fmt.Errorf("unhandled golang type: %s", goType)
	}
}

// TODO: handle case of self referential pointers in structs

// TODO: add doc string explaining numHistoryOccurances
func properity(goType reflect.Type, numHistoryOccurances map[string]int) (*Property, error) {
	// *Struct and Struct generate identical json schemas
	if goType.Kind() == reflect.Pointer {
		return properity(goType.Elem())
	}

	rootJsType, err := jsType(goType)

	// TODO: recursive debugging can be a pain. Make sure the error localtion
	// floats up
	if err != nil {
		return nil, err
	}

	var items *Item
	if goType.Kind() == reflect.Array || goType.Kind() == reflect.Slice {
		elemJsType, err := jsType(goType.Elem())
		if err != nil {
			// TODO: float up error in case of deep recursion
			return nil, err
		}
		items = &Item{
			// TODO: what if there is an array of objects?
			Type: elemJsType,
		}
	}

	properities := map[string]*Property{}
	var additionalProperities *Property

	// TODO: for reflect.Map case for prop computation

	if goType.Kind() == reflect.Struct {
		for i := 0; i < goType.NumField(); i++ {
			field := goType.Field(i)

			// compute child properties
			fieldJsonTag := field.Tag.Get("json")
			fieldName := strings.Split(fieldJsonTag, ",")[0]

			// stopgap infinite recursion
			numHistoryOccurances[fieldName] += 1
			if numHistoryOccurances[fieldName] > MaxHistoryOccurances {
				return nil
			}
			fieldProps, err := properity(field.Type)
			numHistoryOccurances[fieldName] -= 1

			// TODO: make sure this error floats up with context
			if err != nil {
				return nil, err
			}

			if fieldJsonTag != "" {
				properities[fieldName] = fieldProps
			} else if additionalProperities == nil {
				// TODO: add error disallowing self referenincing without json tags
				additionalProperities = fieldProps
			} else {
				// TODO: float error up with context
				return nil, fmt.Errorf("only one non json annotated field allowed")
			}
		}
	}

	return &Property{
		Type:                  rootJsType,
		Items:                 items,
		Properities:           properities,
		AdditionalProperities: additionalProperities,
	}, nil

}
