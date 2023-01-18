package schema

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
)

// defines schema for a json object
type Schema struct {
	// Type of the object
	Type JavascriptType `json:"type,omitempty"`

	// Description of the object. This is rendered as inline documentation in the
	// IDE
	Description string `json:"description,omitempty"`

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
func NewSchema(golangType reflect.Type, docs *Docs) (*Schema, error) {
	seenTypes := map[reflect.Type]struct{}{}
	debugTrace := list.New()
	schema, err := safeToSchema(golangType, docs, "", seenTypes, debugTrace)
	if err != nil {
		return nil, errWithTrace(err.Error(), debugTrace)
	}
	return schema, nil
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
		if curr.Value.(string) != "" {
			traceString += " -> " + curr.Value.(string)
		}
		curr = curr.Next()
	}
	return fmt.Errorf("[ERROR] " + prefix + ". traversal trace: " + traceString)
}

// A wrapper over toSchema function to detect cycles in the bundle config struct
func safeToSchema(golangType reflect.Type, docs *Docs, debugTraceId string, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Schema, error) {
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

	// Add the json tag name of struct field to debug trace
	debugTrace.PushBack(debugTraceId)
	props, err := toSchema(golangType, docs, seenTypes, debugTrace)
	if err != nil {
		return nil, err
	}
	back := debugTrace.Back()
	debugTrace.Remove(back)
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

	for i := 0; i < golangType.NumField(); i++ {
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

// params:
//   golangType: golang type for which json schema properties to generate
//   seenTypes : set of golang types already seen in path during recursion.
//               Used to identify cycles.
//   debugTrace: linked list of golang types encounted. In case of errors this
//               helps log where the error originated from
func toSchema(golangType reflect.Type, docs *Docs, seenTypes map[reflect.Type]struct{}, debugTrace *list.List) (*Schema, error) {
	// *Struct and Struct generate identical json schemas
	if golangType.Kind() == reflect.Pointer {
		return safeToSchema(golangType.Elem(), docs, "", seenTypes, debugTrace)
	}

	if golangType.Kind() == reflect.Interface {
		return &Schema{}, nil
	}

	rootJavascriptType, err := javascriptType(golangType)
	if err != nil {
		return nil, err
	}

	var description string
	if docs != nil {
		description = docs.Documentation
	}

	// case array/slice
	var items *Schema
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemGolangType := golangType.Elem()
		elemJavascriptType, err := javascriptType(elemGolangType)
		if err != nil {
			return nil, err
		}
		elemProps, err := safeToSchema(elemGolangType, docs, "", seenTypes, debugTrace)
		if err != nil {
			return nil, err
		}
		items = &Schema{
			Type:                 elemJavascriptType,
			Properties:           elemProps.Properties,
			AdditionalProperties: elemProps.AdditionalProperties,
			Items:                elemProps.Items,
			Required:             elemProps.Required,
		}
	}

	// case map
	var additionalProperties interface{}
	if golangType.Kind() == reflect.Map {
		if golangType.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("only string keyed maps allowed")
		}
		additionalProperties, err = safeToSchema(golangType.Elem(), docs, "", seenTypes, debugTrace)
		if err != nil {
			return nil, err
		}
	}

	// case struct
	properties := map[string]*Schema{}
	required := []string{}
	if golangType.Kind() == reflect.Struct {
		children := []reflect.StructField{}
		children = addStructFields(children, golangType)
		for _, child := range children {
			// get child json tags
			childJsonTag := strings.Split(child.Tag.Get("json"), ",")
			childName := childJsonTag[0]

			// skip children that have no json tags, the first json tag is ""
			// or the first json tag is "-"
			if childName == "" || childName == "-" {
				continue
			}

			// get docs for the child if they exist
			var childDocs *Docs
			if docs != nil {
				if val, ok := docs.Children[childName]; ok {
					childDocs = &val
				}
			}

			// compute if the child is a required field. Determined by the
			// resence of "omitempty" in the json tags
			hasOmitEmptyTag := false
			for i := 1; i < len(childJsonTag); i++ {
				if childJsonTag[i] == "omitempty" {
					hasOmitEmptyTag = true
				}
			}
			if !hasOmitEmptyTag {
				required = append(required, childName)
			}

			// compute Schema.Properties for the child recursively
			fieldProps, err := safeToSchema(child.Type, childDocs, childName, seenTypes, debugTrace)
			if err != nil {
				return nil, err
			}
			properties[childName] = fieldProps
		}

		// set Schema.AdditionalProperties to false
		additionalProperties = false
	}

	return &Schema{
		Type:                 rootJavascriptType,
		Description:          description,
		Items:                items,
		Properties:           properties,
		AdditionalProperties: additionalProperties,
		Required:             required,
	}, nil
}
