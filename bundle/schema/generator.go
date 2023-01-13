package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// TODO: should omit empty denote non required fields in the json schema
type Schema struct {
	Type        JsType               `json:"type"`
	Properities map[string]*Property `json:"properities,omitempty"`
}

type Property struct {
	// TODO: Add a enum for json types
	Type        JsType               `json:"type"`
	Items       *Item                `json:"item,omitempty"`
	Properities map[string]*Property `json:"properities,omitempty"`
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
		Type:        rootProp.Type,
		Properities: rootProp.Properities,
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

// TODO: add support for pointers
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

// TODO: add support for lower case field name if json tag is missing
func properity(goType reflect.Type) (*Property, error) {
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
	if goType.Kind() == reflect.Struct {
		for i := 0; i < goType.NumField(); i++ {
			field := goType.Field(i)
			fieldProps, err := properity(field.Type)
			// TODO: make sure this error floats up with context
			if err != nil {
				return nil, err
			}
			fieldJsonTags := field.Tag.Get("json")
			// TODO: test if this split is needed
			fieldName := strings.Split(fieldJsonTags, ",")[0]

			properities[fieldName] = fieldProps
		}
	}

	return &Property{
		Type:        rootJsType,
		Items:       items,
		Properities: properities,
	}, nil

}
