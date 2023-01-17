package schema

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
)

// TODO: Add tests for the error cases, forcefully triggering them
// TODO: Add required validation for omitempty
// TODO: Add example documentation
// TODO: Do final checks for more validation that can be added to json schema

type Schema struct {
	Type                 JavascriptType       `json:"type"`
	Properties           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperties *Property            `json:"additionalProperties,omitempty"`
}

type Property struct {
	Type                 JavascriptType       `json:"type"`
	Items                *Item                `json:"items,omitempty"`
	Properties           map[string]*Property `json:"properties,omitempty"`
	AdditionalProperties *Property            `json:"additionalProperties,omitempty"`
}

type Item struct {
	Type       JavascriptType       `json:"type"`
	Properties map[string]*Property `json:"properties,omitempty"`
}

// NOTE about loops in golangType: Right now we error out if there is a loop
// in the types traversed when generating the json schema. This can be solved
// using $refs but there is complexity around making sure we do not create json
// schemas where properties indirectly refer to each other, which would be an
// invalid json schema. See https://json-schema.org/understanding-json-schema/structuring.html#recursion
// for more details
func NewSchema(golangType reflect.Type) (*Schema, error) {
	seenTypes := map[reflect.Type]struct{}{}
	debugTrace := list.New()
	rootProp, err := toProperty(golangType, seenTypes, debugTrace)
	if err != nil {
		return nil, errWithTrace(err.Error(), debugTrace)
	}
	return &Schema{
		Type:                 rootProp.Type,
		Properties:           rootProp.Properties,
		AdditionalProperties: rootProp.AdditionalProperties,
	}, nil
}

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

func errWithTrace(prefix string, trace *list.List) error {
	traceString := "root"
	curr := trace.Front()
	for curr != nil {
		traceString += " -> " + curr.Value.(string)
		curr = curr.Next()
	}
	return fmt.Errorf("[ERROR] " + prefix + ". traversal trace: " + traceString)
}

// A wrapper over toProperty function with checks for an cycles to avoid being
// stuck in an loop when traversing the config struct
func safeToProperty(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Property, error) {
	// detect cycles. Fail if a cycle is detected
	_, ok := seenTypes[golangType]
	if ok {
		fmt.Println("[DEBUG] traceSet: ", seenTypes)
		return nil, fmt.Errorf("cycle detected")
	}
	// Update set of types in current path
	seenTypes[golangType] = struct{}{}
	props, err := toProperty(golangType, seenTypes, debugTrace)
	if err != nil {
		return nil, err
	}
	delete(seenTypes, golangType)
	return props, nil
}

// Adds the member fields of golangType to the passed slice. Needed because
// golangType can contain embedded fields (aka anonymous)
//
// The function traverses the embedded fields in a breadth first manner
//
// params:
// 	  fields: slice to which member fields of golangType will be added to
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

// params:
//   golangType: golang type for which json schema properties to generate
//   seenTypes : set of golang types already seen in path during recursion.
//               Used to identify cycles.
//   debugTrace: linked list of golang types encounted. In case of errors this
//               helps log where the error originated from
func toProperty(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Property, error) {
	// *Struct and Struct generate identical json schemas
	// TODO: add test case for pointer
	if golangType.Kind() == reflect.Pointer {
		return toProperty(golangType.Elem(), seenTypes, debugTrace)
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
			Type:       elemJavascriptType,
			Properties: elemProps.Properties,
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
	properties := map[string]*Property{}
	if golangType.Kind() == reflect.Struct {
		children := []reflect.StructField{}
		children = addStructFields(children, golangType)
		for _, child := range children {
			// compute child properties
			childJsonTag := child.Tag.Get("json")
			childName := strings.Split(childJsonTag, ",")[0]

			// add current field to debug trace
			debugTrace.PushBack(childName)

			// skip fields that are not annotated or annotated with "-"
			if childName == "" || childName == "-" {
				continue
			}

			// recursively compute properties for this child field
			fieldProps, err := safeToProperty(child.Type, seenTypes, debugTrace)
			if err != nil {
				return nil, err
			}
			properties[childName] = fieldProps

			// remove current field from debug trace
			back := debugTrace.Back()
			debugTrace.Remove(back)
		}
	}

	return &Property{
		Type:                 rootJavascriptType,
		Items:                items,
		Properties:           properties,
		AdditionalProperties: additionalProperties,
	}, nil
}
