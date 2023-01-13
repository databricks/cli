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
	Type                  JavascriptType       `json:"type"`
	Properities           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperties,omitempty"`
}

type Property struct {
	Type                  JavascriptType       `json:"type"`
	Items                 *Item                `json:"items,omitempty"`
	Properities           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperities *Property            `json:"additionalProperties,omitempty"`
}

// TODO: panic for now, add support for adding schemas to $defs in case of cycles

type Item struct {
	Type        JavascriptType       `json:"type"`
	Properities map[string]*Property `json:"properties,omitempty"`
}

func NewSchema(golangType reflect.Type) (*Schema, error) {
	seenTypes := map[reflect.Type]struct{}{}
	debugTrace := list.New()
	rootProp, err := toProperity(golangType, seenTypes, debugTrace)
	if err != nil {
		return nil, errWithTrace(err.Error(), debugTrace)
	}
	return &Schema{
		Type:                  rootProp.Type,
		Properities:           rootProp.Properities,
		AdditionalProperities: rootProp.AdditionalProperities,
	}, nil
}

// TODO: add tests for errors being triggered

type JavascriptType string

const (
	Invalid JavascriptType = "invalid"
	Boolean JavascriptType = "boolean"
	String  JavascriptType = "string"
	Number  JavascriptType = "number"
	Object  JavascriptType = "object"
	Array   JavascriptType = "array"
)

func javascriptType(golangType reflect.Type) (JavascriptType, error) {
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
func errWithTrace(prefix string, trace *list.List) error {
	traceString := "root"
	curr := trace.Front()
	for curr.Next() != nil {
		traceString += " -> " + curr.Value.(string)
		curr = curr.Next()
	}
	return fmt.Errorf("[ERROR] " + prefix + ". traveral trace: " + traceString)
}

// TODO: handle case of self referential pointers in structs
// TODO: add handling of embedded types

// TODO: add tests for the error cases, forcefully triggering them

// checks and errors out for cycles
// wraps the error with context
func safeToProperty(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Property, error) {
	// detect cycles. Fail if a cycle is detected
	// TODO: Add references here for cycles
	// TODO: move this check somewhere nicer
	_, ok := seenTypes[golangType]
	if ok {
		fmt.Println("[DEBUG] traceSet: ", seenTypes)
		return nil, fmt.Errorf("cycle detected")
	}
	// add current child field to history
	seenTypes[golangType] = struct{}{}
	props, err := toProperity(golangType, seenTypes, debugTrace)
	if err != nil {
		return nil, err
	}
	delete(seenTypes, golangType)
	return props, nil
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

// TODO: add ignore for -

// TODO: add doc string explaining numHistoryOccurances
func toProperity(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Property, error) {
	// *Struct and Struct generate identical json schemas
	// TODO: add test case for pointer
	if golangType.Kind() == reflect.Pointer {
		return toProperity(golangType.Elem(), seenTypes, debugTrace)
	}

	// TODO: add test case for interfaces
	if golangType.Kind() == reflect.Interface {
		return nil, nil
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
		elemProps, err := safeToProperty(elemGolangType, seenTypes, debugTrace)
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
			return nil, fmt.Errorf("only string keyed maps allowed")
		}
		additionalProperties, err = safeToProperty(golangType.Elem(), seenTypes, debugTrace)
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

			// add current field to debug trace
			debugTrace.PushBack(childName)

			// skip non json annotated fields
			if childName == "" {
				continue
			}

			// recursively compute properties for this child field
			fieldProps, err := safeToProperty(child.Type, seenTypes, debugTrace)
			if err != nil {
				return nil, err
			}
			properities[childName] = fieldProps

			// remove current field from debug trace
			back := debugTrace.Back()
			debugTrace.Remove(back)
		}
	}

	return &Property{
		Type:                  rootJavascriptType,
		Items:                 items,
		Properities:           properities,
		AdditionalProperities: additionalProperties,
	}, nil
}
