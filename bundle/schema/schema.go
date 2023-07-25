package schema

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/schema"
)

// This function translates golang types into json schema. Here is the mapping
// between json schema types and golang types
//
//   - GolangType               ->    Javascript type / Json Schema2
//
//   - bool                     ->    boolean
//
//   - string                   ->    string
//
//   - int (all variants)       ->    number
//
//   - float (all variants)     ->    number
//
//   - map[string]MyStruct      ->   { type: object, additionalProperties: {}}
//     for details visit: https://json-schema.org/understanding-json-schema/reference/object.html#additional-properties
//
//   - []MyStruct               ->   {type: array, items: {}}
//     for details visit: https://json-schema.org/understanding-json-schema/reference/array.html#items
//
//   - []MyStruct               ->   {type: object, properties: {}, additionalProperties: false}
//     for details visit: https://json-schema.org/understanding-json-schema/reference/object.html#properties
func New(golangType reflect.Type, docs *Docs) (*schema.Schema, error) {
	tracker := newTracker()
	schema, err := safeToSchema(golangType, docs, "", tracker)
	if err != nil {
		return nil, tracker.errWithTrace(err.Error(), "root")
	}
	return schema, nil
}

func jsonSchemaType(golangType reflect.Type) (schema.Type, error) {
	switch golangType.Kind() {
	case reflect.Bool:
		return schema.BooleanType, nil
	case reflect.String:
		return schema.StringType, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:

		return schema.NumberType, nil
	case reflect.Struct:
		return schema.ObjectType, nil
	case reflect.Map:
		if golangType.Key().Kind() != reflect.String {
			return schema.InvalidType, fmt.Errorf("only strings map keys are valid. key type: %v", golangType.Key().Kind())
		}
		return schema.ObjectType, nil
	case reflect.Array, reflect.Slice:
		return schema.ArrayType, nil
	default:
		return schema.InvalidType, fmt.Errorf("unhandled golang type: %s", golangType)
	}
}

// A wrapper over toSchema function to:
//  1. Detect cycles in the bundle config struct.
//  2. Update tracker
//
// params:
//
//   - golangType: Golang type to generate json schema for
//
//   - docs: Contains documentation to be injected into the generated json schema
//
//   - traceId: An identifier for the current type, to trace recursive traversal.
//     Its value is the first json tag in case of struct fields and "" in other cases
//     like array, map or no json tags
//
//   - tracker: Keeps track of types / traceIds seen during recursive traversal
func safeToSchema(golangType reflect.Type, docs *Docs, traceId string, tracker *tracker) (*schema.Schema, error) {
	// WE ERROR OUT IF THERE ARE CYCLES IN THE JSON SCHEMA
	// There are mechanisms to deal with cycles though recursive identifiers in json
	// schema. However if we use them, we would need to make sure we are able to detect
	// cycles where two properties (directly or indirectly) pointing to each other
	//
	// see: https://json-schema.org/understanding-json-schema/structuring.html#recursion
	// for details
	if tracker.hasCycle(golangType) {
		return nil, fmt.Errorf("cycle detected")
	}

	tracker.push(golangType, traceId)
	props, err := toSchema(golangType, docs, tracker)
	if err != nil {
		return nil, err
	}
	tracker.pop(golangType)
	return props, nil
}

// This function returns all member fields of the provided type.
// If the type has embedded (aka anonymous) fields, this function traverses
// those in a breadth first manner
func getStructFields(golangType reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}
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

func toSchema(golangType reflect.Type, docs *Docs, tracker *tracker) (*schema.Schema, error) {
	// *Struct and Struct generate identical json schemas
	if golangType.Kind() == reflect.Pointer {
		return safeToSchema(golangType.Elem(), docs, "", tracker)
	}
	if golangType.Kind() == reflect.Interface {
		return &schema.Schema{}, nil
	}

	rootJavascriptType, err := jsonSchemaType(golangType)
	if err != nil {
		return nil, err
	}
	jsonSchema := &schema.Schema{Type: rootJavascriptType}

	if docs != nil {
		jsonSchema.Description = docs.Description
	}

	// case array/slice
	if golangType.Kind() == reflect.Array || golangType.Kind() == reflect.Slice {
		elemGolangType := golangType.Elem()
		elemJavascriptType, err := jsonSchemaType(elemGolangType)
		if err != nil {
			return nil, err
		}
		var childDocs *Docs
		if docs != nil {
			childDocs = docs.Items
		}
		elemProps, err := safeToSchema(elemGolangType, childDocs, "", tracker)
		if err != nil {
			return nil, err
		}
		jsonSchema.Items = &schema.Schema{
			Type:                 elemJavascriptType,
			Properties:           elemProps.Properties,
			AdditionalProperties: elemProps.AdditionalProperties,
			Items:                elemProps.Items,
			Required:             elemProps.Required,
		}
	}

	// case map
	if golangType.Kind() == reflect.Map {
		if golangType.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("only string keyed maps allowed")
		}
		var childDocs *Docs
		if docs != nil {
			childDocs = docs.AdditionalProperties
		}
		jsonSchema.AdditionalProperties, err = safeToSchema(golangType.Elem(), childDocs, "", tracker)
		if err != nil {
			return nil, err
		}
	}

	// case struct
	if golangType.Kind() == reflect.Struct {
		children := getStructFields(golangType)
		properties := map[string]*schema.Schema{}
		required := []string{}
		for _, child := range children {
			bundleTag := child.Tag.Get("bundle")
			if bundleTag == "readonly" {
				continue
			}

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
				if val, ok := docs.Properties[childName]; ok {
					childDocs = val
				}
			}

			// compute if the child is a required field. Determined by the
			// presence of "omitempty" in the json tags
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
			fieldProps, err := safeToSchema(child.Type, childDocs, childName, tracker)
			if err != nil {
				return nil, err
			}
			properties[childName] = fieldProps
		}

		jsonSchema.AdditionalProperties = false
		jsonSchema.Properties = properties
		jsonSchema.Required = required
	}

	return jsonSchema, nil
}
