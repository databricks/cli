package jsonschema

import (
	"container/list"
	"fmt"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"
)

var skipTags = []string{
	// Fields tagged "readonly" should not be emitted in the schema as they are
	// computed at runtime, and should not be assigned a value by the bundle author.
	"readonly",

	// Annotation for internal bundle fields that should not be exposed to customers.
	// Fields can be tagged as "internal" to remove them from the generated schema.
	"internal",
}

type constructor struct {
	// Map of typ.PkgPath() + "." + typ.Name() to the schema for that type.
	// Example key: github.com/databricks/databricks-sdk-go/service/jobs.JobSettings
	definitions map[string]Schema

	// Map of typ.PkgPath() + "." + typ.Name() to the corresponding type. Used to
	// track types that have been seen to avoid infinite recursion.
	seen map[string]reflect.Type

	// The root type for which the schema is being generated.
	root reflect.Type
}

// JSON pointers use "/" as a delimiter to represent nested objects. This means
// we would instead need to use "~1" to represent "/" if we wish to refer to a
// key in a JSON object with a "/" in it. Instead of doing that we replace "/" with an
// additional level of nesting in the output map. Thus the $refs in the generated
// JSON schema can contain "/" without any issues.
// see: https://datatracker.ietf.org/doc/html/rfc6901
//
// For example:
// {"a/b/c": "value"} is converted to {"a": {"b": {"c": "value"}}}
// the $ref for "value" would be "#/$defs/a/b/c" in the generated JSON schema.
func (c *constructor) Definitions() map[string]any {
	defs := maps.Clone(c.definitions)

	// Remove the root type from the definitions. We don't need to include it in
	// the definitions because it will be inlined as the root of the generated JSON schema.
	delete(defs, typePath(c.root))

	if len(defs) == 0 {
		return nil
	}

	res := make(map[string]any)
	for k, v := range defs {
		parts := strings.Split(k, "/")
		cur := res
		for i, p := range parts {
			// Set the value for the last part.
			if i == len(parts)-1 {
				cur[p] = v
				break
			}

			// For all but the last part, create a new map value to add a level
			// of nesting.
			if _, ok := cur[p]; !ok {
				cur[p] = make(map[string]any)
			}
			cur = cur[p].(map[string]any)
		}
	}

	return res
}

// FromType converts a [reflect.Type] to a [Schema]. Nodes in the final JSON
// schema are guaranteed to be one level deep, which is done using defining $defs
// for every Go type and referring them using $ref in the corresponding node in
// the JSON schema.
//
// fns is a list of transformation functions that will be applied in order to all $defs
// in the schema.
func FromType(typ reflect.Type, fns []func(typ reflect.Type, s Schema) Schema) (Schema, error) {
	c := constructor{
		definitions: make(map[string]Schema),
		seen:        make(map[string]reflect.Type),
		root:        typ,
	}

	_, err := c.walk(typ)
	if err != nil {
		return Schema{}, err
	}

	for _, fn := range fns {
		for k := range c.definitions {
			c.definitions[k] = fn(c.seen[k], c.definitions[k])
		}
	}

	res := c.definitions[typePath(typ)]
	res.Definitions = c.Definitions()
	return res, nil
}

func TypePath(typ reflect.Type) string {
	return typePath(typ)
}

// typePath computes a unique string representation of the type. $ref in the generated
// JSON schema will refer to this path. See TestTypePath for examples outputs.
func typePath(typ reflect.Type) string {
	// Pointers have a typ.Name() of "". Dereference them to get the underlying type.
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() == reflect.Interface {
		return "interface"
	}

	// Recursively call typePath, to handle slices of slices / maps.
	if typ.Kind() == reflect.Slice {
		return path.Join("slice", typePath(typ.Elem()))
	}

	if typ.Kind() == reflect.Map {
		if typ.Key().Kind() != reflect.String {
			panic(fmt.Sprintf("found map with non-string key: %v", typ.Key()))
		}

		// Recursively call typePath, to handle maps of maps / slices.
		return path.Join("map", typePath(typ.Elem()))
	}

	switch {
	case typ.PkgPath() != "" && typ.Name() != "":
		return typ.PkgPath() + "." + typ.Name()
	case typ.Name() != "":
		return typ.Name()
	default:
		// Invariant. This function should return a non-empty string
		// for all types.
		panic("unexpected empty type name for type: " + typ.String())
	}
}

// Walk the Go type, generating $defs for every type encountered, and populating
// the corresponding $ref in the JSON schema.
func (c *constructor) walk(typ reflect.Type) (string, error) {
	// Dereference pointers if necessary.
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	typPath := typePath(typ)

	// Return early if the type has already been seen, to avoid infinite recursion.
	if _, ok := c.seen[typPath]; ok {
		return typPath, nil
	}
	c.seen[typPath] = typ

	var s Schema
	var err error

	switch typ.Kind() {
	case reflect.Struct:
		s, err = c.fromTypeStruct(typ)
	case reflect.Slice:
		s, err = c.fromTypeSlice(typ)
	case reflect.Map:
		s, err = c.fromTypeMap(typ)
	case reflect.String:
		s = Schema{Type: StringType}
	case reflect.Bool:
		s = Schema{Type: BooleanType}
	case reflect.Int, reflect.Int32, reflect.Int64:
		s = Schema{Type: IntegerType}
	case reflect.Float32, reflect.Float64:
		s = Schema{Type: NumberType}
	case reflect.Interface:
		// We cannot determine the schema for fields of interface type just based
		// on the type information. Thus we'll set the empty schema here and allow
		// arbitrary values.
		s = Schema{}
	default:
		return "", fmt.Errorf("unsupported type: %s", typ.Kind())
	}
	if err != nil {
		return "", err
	}

	// Store the computed JSON schema for the type.
	c.definitions[typPath] = s
	return typPath, nil
}

// This function returns all member fields of the provided type.
// If the type has embedded (aka anonymous) fields, this function traverses
// those in a breadth first manner
//
// BFS is important because we want the a field defined at a higher level embedded
// struct to be given preference over a field with the same name defined at a lower
// level embedded struct. For example see: TestHigherLevelEmbeddedFieldIsInSchema
func getStructFields(typ reflect.Type) []reflect.StructField {
	var fields []reflect.StructField
	bfsQueue := list.New()

	for i := range typ.NumField() {
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

		// Embedded types can only be struct{} or pointer to struct{}. Multiple
		// levels of pointers are not allowed by the Go compiler. So we only
		// dereference pointers once.
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		for i := range fieldType.NumField() {
			bfsQueue.PushBack(fieldType.Field(i))
		}
	}
	return fields
}

func (c *constructor) fromTypeStruct(typ reflect.Type) (Schema, error) {
	if typ.Kind() != reflect.Struct {
		return Schema{}, fmt.Errorf("expected struct, got %s", typ.Kind())
	}

	res := Schema{
		Type:                 ObjectType,
		Properties:           make(map[string]*Schema),
		Required:             []string{},
		AdditionalProperties: false,
	}

	structFields := getStructFields(typ)
	for _, structField := range structFields {
		bundleTags := strings.Split(structField.Tag.Get("bundle"), ",")
		// Fields marked as "readonly" or "internal" are skipped while generating
		// the schema
		skip := false
		for _, tag := range skipTags {
			if slices.Contains(bundleTags, tag) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		jsonTags := strings.Split(structField.Tag.Get("json"), ",")
		fieldName := jsonTags[0]
		// Do not include fields in the schema that will not be serialized during
		// JSON marshalling.
		if fieldName == "" || fieldName == "-" || !structField.IsExported() {
			continue
		}

		// Skip property if it is already present in the schema.
		// This can happen if the same field is defined multiple times across
		// a tree of embedded structs. For example see: TestHigherLevelEmbeddedFieldIsInSchema
		if _, ok := res.Properties[fieldName]; ok {
			continue
		}

		// "omitempty" tags in the Go SDK structs represent fields that not are
		// required to be present in the API payload. Thus its absence in the
		// tags list indicates that the field is required.
		if !slices.Contains(jsonTags, "omitempty") {
			res.Required = append(res.Required, fieldName)
		}

		// Walk the fields of the struct.
		typPath, err := c.walk(structField.Type)
		if err != nil {
			return Schema{}, err
		}

		// For every property in the struct, add a $ref to the corresponding
		// $defs block.
		refPath := path.Join("#/$defs", typPath)
		res.Properties[fieldName] = &Schema{
			Reference: &refPath,
		}
	}

	return res, nil
}

func (c *constructor) fromTypeSlice(typ reflect.Type) (Schema, error) {
	if typ.Kind() != reflect.Slice {
		return Schema{}, fmt.Errorf("expected slice, got %s", typ.Kind())
	}

	res := Schema{
		Type: ArrayType,
	}

	// Walk the slice element type.
	typPath, err := c.walk(typ.Elem())
	if err != nil {
		return Schema{}, err
	}

	refPath := path.Join("#/$defs", typPath)

	// Add a $ref to the corresponding $defs block for the slice element type.
	res.Items = &Schema{
		Reference: &refPath,
	}
	return res, nil
}

func (c *constructor) fromTypeMap(typ reflect.Type) (Schema, error) {
	if typ.Kind() != reflect.Map {
		return Schema{}, fmt.Errorf("expected map, got %s", typ.Kind())
	}

	res := Schema{
		Type: ObjectType,
	}

	// Walk the map value type.
	typPath, err := c.walk(typ.Elem())
	if err != nil {
		return Schema{}, err
	}

	refPath := path.Join("#/$defs", typPath)

	// Add a $ref to the corresponding $defs block for the map value type.
	res.AdditionalProperties = &Schema{
		Reference: &refPath,
	}
	return res, nil
}
