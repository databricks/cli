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
// TODO: Run all tests to see code coverage and add tests for missing assertions

// defines schema for a json object
type Schema struct {
	// Type of the object
	Type JavascriptType `json:"type"`

	// keys are named properties of the object
	// values are json schema for the values of the named properties
	Properties map[string]*Schema `json:"properties,omitempty"`

	// the schema for all values of the array
	Items *Schema `json:"items,omitempty"`

	// the schema for any properties not mentioned in the Schema.Properties field.
	// we leverage this to validate Maps in bundle configuration
	// OR
	// a boolean type with value false
	//
	// Its type during runtime will either be *Schema or bool
	AdditionalProperties interface{} `json:"additionalProperties,omitempty"`

	// required properties for the object. Any propertites listed here should
	// also be listed in Schema.Properties
	Required []string `json:"required,omitempty"`
}

/*
	This function translates golang types into json schema. Here is the mapping
	between json schema types and golang types

	- GolangType               ->    Javascript type / Json Schema2

	Javascript Primitives:
	- bool                     ->    boolean
	- string                   ->    string
	- int (all variants)       ->    number
	- float (all variants)     ->    number

	Json Schema2 Fields:
	- map[string]MyStruct      ->   {
										type: object
										additionalProperties: {}
									}
	for details visit: https://json-schema.org/understanding-json-schema/reference/object.html#additional-properties

	- []MyStruct               ->   {
										type: array
										items: {}
									}

	for details visit: https://json-schema.org/understanding-json-schema/reference/array.html#items


	- []MyStruct               ->   {
										type: object
										properties: {}
										additionalProperties: false
									}
	for details visit: https://json-schema.org/understanding-json-schema/reference/object.html#properties
*/
func NewSchema(golangType reflect.Type) (*Schema, error) {
	seenTypes := map[reflect.Type]struct{}{}
	debugTrace := list.New()
	rootProp, err := toSchema(golangType, seenTypes, debugTrace)
	if err != nil {
		return nil, errWithTrace(err.Error(), debugTrace)
	}
	return &Schema{
		Type:                 rootProp.Type,
		Properties:           rootProp.Properties,
		AdditionalProperties: rootProp.AdditionalProperties,
		Items:                rootProp.Items,
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

// TODO: document that only string keys allowed in maps
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

// A wrapper over toSchema function to detect cycles in the bundle config struct
func safeToSchema(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Schema, error) {
	// WE ERROR OUT IF THERE ARE CYCLES IN THE JSON SCHEMA
	// There are mechanisms to deal with cycles though recursive identifiers in json
	// schema. However if we use them, we would need to make sure we are able to detect
	// cycles two properties (directly or indirectly) pointing to each other
	//
	// see: https://json-schema.org/understanding-json-schema/structuring.html#recursion
	// for details
	_, ok := seenTypes[golangType]
	if ok {
		fmt.Println("[DEBUG] traceSet: ", seenTypes)
		return nil, fmt.Errorf("cycle detected")
	}
	// Update set of types in current path
	seenTypes[golangType] = struct{}{}
	props, err := toSchema(golangType, seenTypes, debugTrace)
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

// TODO: see what kind of schema will be generated for interface{}

// params:
//   golangType: golang type for which json schema properties to generate
//   seenTypes : set of golang types already seen in path during recursion.
//               Used to identify cycles.
//   debugTrace: linked list of golang types encounted. In case of errors this
//               helps log where the error originated from
func toSchema(golangType reflect.Type, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Schema, error) {
	// *Struct and Struct generate identical json schemas
	// TODO: add test case for pointer
	if golangType.Kind() == reflect.Pointer {
		return toSchema(golangType.Elem(), seenTypes, debugTrace)
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
	var items *Schema
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemGolangType := golangType.Elem()
		elemJavascriptType, err := javascriptType(elemGolangType)
		if err != nil {
			return nil, err
		}
		elemProps, err := safeToSchema(elemGolangType, seenTypes, debugTrace)
		if err != nil {
			return nil, err
		}
		items = &Schema{
			// TODO: Add a test for slice of object
			Type:                 elemJavascriptType,
			Properties:           elemProps.Properties,
			AdditionalProperties: elemProps.AdditionalProperties,
			Items:                elemProps.Items,
		}
		// TODO: what if there is an array of maps. Add additional properties to
		// TODO: what if there are maps of maps
	}

	// case map
	var additionalProperties interface{}
	if golangType.Kind() == reflect.Map {
		if golangType.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("only string keyed maps allowed")
		}
		// TODO: Add a test for map of maps, and map of slices. Check that there
		// is already a test for map of objects and map of primites
		additionalProperties, err = safeToSchema(golangType.Elem(), seenTypes, debugTrace)
		if err != nil {
			return nil, err
		}
	}

	// case struct
	properties := map[string]*Schema{}
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
			fieldProps, err := safeToSchema(child.Type, seenTypes, debugTrace)
			if err != nil {
				return nil, err
			}
			properties[childName] = fieldProps

			// remove current field from debug trace
			back := debugTrace.Back()
			debugTrace.Remove(back)

			// set Schema.AdditionalProperties to false
			additionalProperties = false
		}
	}

	return &Schema{
		Type:                 rootJavascriptType,
		Items:                items,
		Properties:           properties,
		AdditionalProperties: additionalProperties,
	}, nil
}
