package jsonschema

import (
	"container/list"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

var InvalidSchema = Schema{
	Type: InvalidType,
}

// Fields tagged "readonly" should not be emitted in the schema as they are
// computed at runtime, and should not be assigned a value by the bundle author.
const readonlyTag = "readonly"

// Annotation for internal bundle fields that should not be exposed to customers.
// Fields can be tagged as "internal" to remove them from the generated schema.
const internalTag = "internal"

// Annotation for bundle fields that have been deprecated.
// Fields tagged as "deprecated" are removed/omitted from the generated schema.
const deprecatedTag = "deprecated"

// TODO: Test what happens with invalid cycles? Do integration tests fail?
// TODO: Call out in the PR description that recursive types like "for_each_task"
// are now supported.

type FromTypeOptions struct {
	// Transformation function to apply after generating the schema.
	Transform func(s Schema) Schema
}

// TODO: Skip generating schema for interface fields.
func FromType(typ reflect.Type, opts FromTypeOptions) (Schema, error) {
	// Dereference pointers if necessary.
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// An interface value can never be serialized from text, and thus is explicitly
	// set to null and disallowed in the schema.
	if typ.Kind() == reflect.Interface {
		return Schema{Type: NullType}, nil
	}

	var res Schema
	var err error

	// TODO: Narrow down the number of Go types handled here.
	switch typ.Kind() {
	case reflect.Struct:
		res, err = fromTypeStruct(typ, opts)
	case reflect.Slice:
		res, err = fromTypeSlice(typ, opts)
	case reflect.Map:
		res, err = fromTypeMap(typ, opts)
		// TODO: Should the primitive functions below be inlined?
	case reflect.String:
		res = Schema{Type: StringType}
	case reflect.Bool:
		res = Schema{Type: BooleanType}
	// case reflect.Int, reflect.Int32, reflect.Int64:
	// 	res = Schema{Type: IntegerType}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		res = Schema{Type: NumberType}
	default:
		return InvalidSchema, fmt.Errorf("unsupported type: %s", typ.Kind())
	}
	if err != nil {
		return InvalidSchema, err
	}

	if opts.Transform != nil {
		res = opts.Transform(res)
	}
	return res, nil
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

func fromTypeStruct(typ reflect.Type, opts FromTypeOptions) (Schema, error) {
	if typ.Kind() != reflect.Struct {
		return InvalidSchema, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	res := Schema{
		Type: ObjectType,

		Properties: make(map[string]*Schema),

		// TODO: Confirm that empty arrays are not serialized.
		Required: []string{},

		AdditionalProperties: false,
	}

	structFields := getStructFields(typ)

	for _, structField := range structFields {
		bundleTags := strings.Split(structField.Tag.Get("bundle"), ",")
		// Fields marked as "readonly", "internal" or "deprecated" are skipped
		// while generating the schema
		if slices.Contains(bundleTags, readonlyTag) ||
			slices.Contains(bundleTags, internalTag) ||
			slices.Contains(bundleTags, deprecatedTag) {
			continue
		}

		jsonTags := strings.Split(structField.Tag.Get("json"), ",")
		// Do not include fields in the schema that will not be serialized during
		// JSON marshalling.
		if jsonTags[0] == "" || jsonTags[0] == "-" {
			continue
		}
		// "omitempty" tags in the Go SDK structs represent fields that not are
		// required to be present in the API payload. Thus its absence in the
		// tags list indicates that the field is required.
		if !slices.Contains(jsonTags, "omitempty") {
			res.Required = append(res.Required, jsonTags[0])
		}

		s, err := FromType(structField.Type, opts)
		if err != nil {
			return InvalidSchema, err
		}
		res.Properties[jsonTags[0]] = &s
	}

	return res, nil
}

func fromTypeSlice(typ reflect.Type, opts FromTypeOptions) (Schema, error) {
	if typ.Kind() != reflect.Slice {
		return InvalidSchema, fmt.Errorf("expected slice, got %s", typ.Kind())
	}

	res := Schema{
		Type: ArrayType,
	}

	items, err := FromType(typ.Elem(), opts)
	if err != nil {
		return InvalidSchema, err
	}

	res.Items = &items
	return res, nil
}

func fromTypeMap(typ reflect.Type, opts FromTypeOptions) (Schema, error) {
	if typ.Kind() != reflect.Map {
		return InvalidSchema, fmt.Errorf("expected map, got %s", typ.Kind())
	}

	if typ.Key().Kind() != reflect.String {
		return InvalidSchema, fmt.Errorf("found map with non-string key: %v", typ.Key())
	}

	res := Schema{
		Type: ObjectType,
	}

	additionalProperties, err := FromType(typ.Elem(), opts)
	if err != nil {
		return InvalidSchema, err
	}
	res.AdditionalProperties = additionalProperties
	return res, nil
}
