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

func errWithTrace(prefix string, trace []reflect.Type) error {
	traceString := ""
	for _, golangType := range trace {
		traceString += " -> " + golangType.Name()
	}
	return fmt.Errorf("[ERROR] " + prefix + " type traveral trace: " + traceString)
}

// TODO: handle case of self referential pointers in structs

// TODO: add doc string explaining numHistoryOccurances
func toProperity(golangType reflect.Type, traceSet map[reflect.Type]struct{}, traceSlice []reflect.Type) (*Property, error) {
	traceSlice = append(traceSlice, golangType)

	// *Struct and Struct generate identical json schemas
	if golangType.Kind() == reflect.Pointer {
		return toProperity(golangType.Elem(), traceSet, traceSlice)
	}

	rootJavascriptType, err := javascriptType(golangType)

	// TODO: recursive debugging can be a pain. Make sure the error localtion
	// floats up
	if err != nil {
		return nil, err
	}

	var items *Item
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemJsType, err := javascriptType(golangType.Elem())
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

	// TODO: for reflect.Map case for prop computation
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

			// TODO: make sure this error floats up with context
			if err != nil {
				return nil, err
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
