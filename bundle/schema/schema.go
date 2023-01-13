package schema

import (
	"container/list"
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
	Type        JsType               `json:"type"`
	Properities map[string]*Property `json:"properties,omitempty"`
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

// TODO: add tests for errors being triggered

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

// checks and errors out for cycles
// wraps the error with context
func safeToProperty(golangType reflect.Type, traceSet map[reflect.Type]struct{}, traceSlice []reflect.Type) (*Property, error) {
	traceSlice = append(traceSlice, golangType)
	// detect cycles. Fail if a cycle is detected
	// TODO: Add references here for cycles
	// TODO: move this check somewhere nicer
	_, ok := traceSet[golangType]
	if ok {
		fmt.Println("[DEBUG] traceSet: ", traceSet)
		return nil, errWithTrace("cycle detected", traceSlice)
	}
	// add current child field to history
	traceSet[golangType] = struct{}{}
	props, err := toProperity(golangType, traceSet, traceSlice)
	if err != nil {
		return nil, errWithTrace(err.Error(), traceSlice)
	}
	delete(traceSet, golangType)
	traceSlice = traceSlice[:len(traceSlice)-1]
	return props, nil
}

func pop(q []reflect.StructField) reflect.StructField {
	elem := q[0]
	q = q[1:]
	return elem
}

func push(q []reflect.StructField, r reflect.StructField) {
	q = append(q, r)
}

// travels anonymous embedded fields in a bfs manner to give us a list of all
// member fields of a struct
// simple Tree based traversal will take place because embbedded fields cannot
// form a cycle
func addStructFields(fields []reflect.StructField, golangType reflect.Type) []reflect.StructField {
	bfsQueue := list.New()

	for i := golangType.NumField() - 1; i >= 0; i-- {
		bfsQueue.PushBack(golangType.Field(i))
	}
	for bfsQueue.Len() > 0 {
		front := bfsQueue.Front()
		field := front.Value.(reflect.StructField)
		bfsQueue.Remove(front)

		if !field.Anonymous {
			fields = append(fields, field)
			continue
		}

		// TODO: add test case for pointer too
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		for i := 0; i < fieldType.NumField(); i++ {
			bfsQueue.PushBack(fieldType.Field(i))
		}
	}
	return fields
}

// TODO: add doc string explaining numHistoryOccurances
func toProperity(golangType reflect.Type, traceSet map[reflect.Type]struct{}, traceSlice []reflect.Type) (*Property, error) {
	// *Struct and Struct generate identical json schemas
	if golangType.Kind() == reflect.Pointer {
		return toProperity(golangType.Elem(), traceSet, traceSlice)
	}

	rootJavascriptType, err := javascriptType(golangType)
	if err != nil {
		return nil, err
	}

	// case array/slice
	var items *Item
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemGolangType := golangType.Elem()
		elemJavascriptType, err := javascriptType(elemGolangType)
		if err != nil {
			return nil, err
		}
		elemProps, err := safeToProperty(elemGolangType, traceSet, traceSlice)
		if err != nil {
			return nil, err
		}
		items = &Item{
			// TODO: Add a test for slice of object
			Type:        elemJavascriptType,
			Properities: elemProps.Properities,
		}
	}

	// case map
	var additionalProperties *Property
	if golangType.Kind() == reflect.Map {
		if golangType.Key().Kind() != reflect.String {
			return nil, errWithTrace("only string keyed maps allowed", traceSlice)
		}
		additionalProperties, err = safeToProperty(golangType.Elem(), traceSet, traceSlice)
		if err != nil {
			return nil, err
		}
	}

	// case struct
	properities := map[string]*Property{}
	if golangType.Kind() == reflect.Struct {
		children := []reflect.StructField{}
		children = addStructFields(children, golangType)
		for _, child := range children {
			// compute child properties
			childJsonTag := child.Tag.Get("json")
			childName := strings.Split(childJsonTag, ",")[0]

			// skip non json annotated fields
			if childName == "" {
				continue
			}

			// recursively compute properties for this child field
			fieldProps, err := safeToProperty(child.Type, traceSet, traceSlice)
			if err != nil {
				return nil, errWithTrace(err.Error(), traceSlice)
			}
			properities[childName] = fieldProps
		}
	}

	return &Property{
		Type:                  rootJavascriptType,
		Items:                 items,
		Properities:           properities,
		AdditionalProperities: additionalProperties,
	}, nil
}
