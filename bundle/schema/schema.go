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
	Properities           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperties,omitempty"`
}

type Property struct {
	Type                  JsType               `json:"type"`
	Items                 *Item                `json:"items,omitempty"`
	Properities           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperties,omitempty"`
}

// TODO: panic for now, add support for adding schemas to $defs in case of cycles

type Item struct {
	Type JsType `json:"type"`
	Properities  map[string]*Property `json:"properties,omitempty"`
}

func NewSchema(golangType reflect.Type) (*Schema, error) {
	traceSet := map[reflect.Type]struct{}{}
	traceSlice := []reflect.Type{}
	rootProp, err := toProperity(golangType, traceSet, traceSlice)
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

func javascriptType(golangType reflect.Type) (JsType, error) {
	switch golangType.Kind() {
	case reflect.Bool:
		return Boolean, nil
	case reflect.String:
		return String, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:

		return Number, nil
	case reflect.Struct:
		return Object, nil
	// TODO: add support for pattern properities to account for maps
	case reflect.Map:
		if golangType.Key().Kind() != reflect.String {
			return Invalid, fmt.Errorf("only strings map keys are valid. key type: %v", golangType.Key().Kind())
		}
		return Object, nil
	case reflect.Array, reflect.Slice:
		return Array, nil
	default:
		return Invalid, fmt.Errorf("unhandled golang type: %s", golangType)
	}
}

// TODO: add a simple test for this
func errWithTrace(prefix string, trace []reflect.Type) error {
	traceString := ""
	for _, golangType := range trace {
		traceString += " -> " + golangType.Name()
	}
	return fmt.Errorf("[ERROR] " + prefix + " type traveral trace: " + traceString)
}

// TODO: handle case of self referential pointers in structs
// TODO: add handling of embedded types

// TODO: add tests for the error cases, forcefully triggering them

// TODO: add doc string explaining numHistoryOccurances
func toProperity(golangType reflect.Type, traceSet map[reflect.Type]struct{}, traceSlice []reflect.Type) (*Property, error) {
	traceSlice = append(traceSlice, golangType)

	// *Struct and Struct generate identical json schemas
	if golangType.Kind() == reflect.Pointer {
		return toProperity(golangType.Elem(), traceSet, traceSlice)
	}

	rootJavascriptType, err := javascriptType(golangType)

	if err != nil {
		return nil, errWithTrace(err.Error(), traceSlice)
	}

	var items *Item
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemGolangType := golangType.Elem()
		elemJavascriptType, err := javascriptType(elemGolangType)
		if err != nil {
			return nil, errWithTrace(err.Error(), traceSlice)
		}

		// detect cycles. Fail if a cycle is detected
		// TODO: Add references here for cycles
		_, ok := traceSet[elemGolangType]
		if ok {
			fmt.Println("[DEBUG] traceSet: ", traceSet)
			return nil, errWithTrace("cycle detected", traceSlice)
		}
		// add current child field to history
		traceSet[elemGolangType] = struct{}{}
		elemProps, err := toProperity(elemGolangType, traceSet, traceSlice)
		if err != nil {
			return nil, errWithTrace(err.Error(), traceSlice)
		}

		items = &Item{
			// TODO: Add a test for slice of object
			Type: elemJavascriptType,
			Properities: elemProps.Properities,
		}
	}

	// var additionalProperties *Property
	// if golangType.Kind() == reflect.Map {
	// 	additionalProperties = 
	// }

	properities := map[string]*Property{}
	if golangType.Kind() == reflect.Struct {
		for i := 0; i < golangType.NumField(); i++ {
			child := golangType.Field(i)

			// compute child properties
			childJsonTag := child.Tag.Get("json")
			childName := strings.Split(childJsonTag, ",")[0]

			// skip non json annotated fields
			if childName == "" {
				continue
			}

			// detect cycles. Fail if a cycle is detected
			// TODO: Add references here for cycles
			_, ok := traceSet[child.Type]
			if ok {
				fmt.Println("[DEBUG] traceSet: ", traceSet)
				return nil, errWithTrace("cycle detected", traceSlice)
			}

			// add current child field to history
			traceSet[child.Type] = struct{}{}

			// recursively compute properties for this child field
			fieldProps, err := toProperity(child.Type, traceSet, traceSlice)

			if err != nil {
				return nil, errWithTrace(err.Error(), traceSlice)
			}

			// traversal complete, delete child from history
			delete(traceSet, child.Type)

			properities[childName] = fieldProps
		}
	}

	traceSlice = traceSlice[:len(traceSlice)-1]

	return &Property{
		Type:        rootJavascriptType,
		Items:       items,
		Properities: properities,
	}, nil
}
