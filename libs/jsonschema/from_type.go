package jsonschema

import (
	"container/list"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"
)

// TODO: Maybe can be removed?
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

type constructor struct {
	// Map of typ.PkgPath() + "." + typ.Name() to the schema for that type.
	// Example key: github.com/databricks/databricks-sdk-go/service/jobs.JobSettings
	definitions map[string]Schema

	// Transformation function to apply after generating a node in the schema.
	fn func(s Schema) Schema
}

// The $defs block in a JSON schema cannot contain "/", otherwise it will not be
// correctly parsed by a JSON schema validator. So we replace "/" with an additional
// level of nesting in the output map.
//
// For example:
// {"a/b/c": "value"} is converted to {"a": {"b": {"c": "value"}}}
func (c *constructor) nestedDefinitions() any {
	if len(c.definitions) == 0 {
		return nil
	}

	res := make(map[string]any)
	for k, v := range c.definitions {
		parts := strings.Split(k, "/")
		cur := res
		for i, p := range parts {
			if i == len(parts)-1 {
				cur[p] = v
				break
			}

			if _, ok := cur[p]; !ok {
				cur[p] = make(map[string]any)
			}
			cur = cur[p].(map[string]any)
		}
	}

	return res
}

// TODO: Skip generating schema for interface fields.
func FromType(typ reflect.Type, fn func(s Schema) Schema) (Schema, error) {
	c := constructor{
		definitions: make(map[string]Schema),
		fn:          fn,
	}

	_, err := c.walk(typ)
	if err != nil {
		return InvalidSchema, err
	}

	res := c.definitions[typePath(typ)]
	// No need to include the root type in the definitions.
	delete(c.definitions, typePath(typ))
	res.Definitions = c.nestedDefinitions()
	return res, nil
}

func typePath(typ reflect.Type) string {
	// typ.Name() resolves to "" for any type.
	if typ.Kind() == reflect.Interface {
		return "interface"
	}

	// For built-in types, return the type name directly.
	if typ.PkgPath() == "" {
		return typ.Name()
	}

	return strings.Join([]string{typ.PkgPath(), typ.Name()}, ".")
}

// TODO: would a worked based model fit better here? Is this internal API not
// the right fit?
func (c *constructor) walk(typ reflect.Type) (string, error) {
	// Dereference pointers if necessary.
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	typPath := typePath(typ)

	// Return value directly if it's already been processed.
	if _, ok := c.definitions[typPath]; ok {
		return "", nil
	}

	var s Schema
	var err error

	// TODO: Narrow / widen down the number of Go types handled here.
	switch typ.Kind() {
	case reflect.Struct:
		s, err = c.fromTypeStruct(typ)
	case reflect.Slice:
		s, err = c.fromTypeSlice(typ)
	case reflect.Map:
		s, err = c.fromTypeMap(typ)
		// TODO: Should the primitive functions below be inlined?
	case reflect.String:
		s = Schema{Type: StringType}
	case reflect.Bool:
		s = Schema{Type: BooleanType}
	// TODO: Add comment about reduced coverage of primitive Go types in the code paths here.
	case reflect.Int:
		s = Schema{Type: IntegerType}
	case reflect.Float32, reflect.Float64:
		s = Schema{Type: NumberType}
	case reflect.Interface:
		// An interface value can never be serialized from text, and thus is explicitly
		// set to null and disallowed in the schema.
		s = Schema{Type: NullType}
	default:
		return "", fmt.Errorf("unsupported type: %s", typ.Kind())
	}
	if err != nil {
		return "", err
	}

	if c.fn != nil {
		s = c.fn(s)
	}

	// Store definition for the type if it's part of a Go package and not a built-in type.
	// TODO: Apply transformation at the end, to all definitions instead of
	// during recursive traversal?
	c.definitions[typPath] = s
	return typPath, nil
}

// This function returns all member fields of the provided type.
// If the type has embedded (aka anonymous) fields, this function traverses
// those in a breadth first manner
func getStructFields(typ reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}
	bfsQueue := list.New()

	for i := 0; i < typ.NumField(); i++ {
		bfsQueue.PushBack(typ.Field(i))
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

func (c *constructor) fromTypeStruct(typ reflect.Type) (Schema, error) {
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
		if jsonTags[0] == "" || jsonTags[0] == "-" || !structField.IsExported() {
			continue
		}
		// "omitempty" tags in the Go SDK structs represent fields that not are
		// required to be present in the API payload. Thus its absence in the
		// tags list indicates that the field is required.
		if !slices.Contains(jsonTags, "omitempty") {
			res.Required = append(res.Required, jsonTags[0])
		}

		// Trigger call to fromType, to recursively generate definitions for
		// the struct field.
		typPath, err := c.walk(structField.Type)
		if err != nil {
			return InvalidSchema, err
		}

		refPath := path.Join("#/$defs", typPath)
		// For non-built-in types, refer to the definition.
		res.Properties[jsonTags[0]] = &Schema{
			Reference: &refPath,
		}
	}

	return res, nil
}

// TODO: Add comments explaining the translation between struct, map, slice and
// the JSON schema representation.
func (c *constructor) fromTypeSlice(typ reflect.Type) (Schema, error) {
	if typ.Kind() != reflect.Slice {
		return InvalidSchema, fmt.Errorf("expected slice, got %s", typ.Kind())
	}

	res := Schema{
		Type: ArrayType,
	}

	// Trigger call to fromType, to recursively generate definitions for
	// the slice element.
	typPath, err := c.walk(typ.Elem())
	if err != nil {
		return InvalidSchema, err
	}

	refPath := path.Join("#/$defs", typPath)

	// For non-built-in types, refer to the definition
	res.Items = &Schema{
		Reference: &refPath,
	}
	return res, nil
}

func (c *constructor) fromTypeMap(typ reflect.Type) (Schema, error) {
	if typ.Kind() != reflect.Map {
		return InvalidSchema, fmt.Errorf("expected map, got %s", typ.Kind())
	}

	if typ.Key().Kind() != reflect.String {
		return InvalidSchema, fmt.Errorf("found map with non-string key: %v", typ.Key())
	}

	res := Schema{
		Type: ObjectType,
	}

	// Trigger call to fromType, to recursively generate definitions for
	// the map value.
	typPath, err := c.walk(typ.Elem())
	if err != nil {
		return InvalidSchema, err
	}

	refPath := path.Join("#/$defs", typPath)

	// For non-built-in types, refer to the definition
	res.AdditionalProperties = &Schema{
		Reference: &refPath,
	}
	return res, nil
}
