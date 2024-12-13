package generator

import (
	"fmt"
	"slices"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	"github.com/zclconf/go-cty/cty"
)

type field struct {
	Name string
	Type string
	Tag  string
}

type structType struct {
	Name   string
	Fields []field
}

// walker represents the set of types to declare to
// represent a [tfjson.SchemaBlock] as Go structs.
// See the [walk] function for usage.
type walker struct {
	StructTypes []structType
}

func processAttributeType(typ cty.Type) string {
	var out string

	switch {
	case typ.IsPrimitiveType():
		switch {
		case typ.Equals(cty.Bool):
			out = "bool"
		case typ.Equals(cty.Number):
			out = "int"
		case typ.Equals(cty.String):
			out = "string"
		default:
			panic("No idea what to do for: " + typ.FriendlyName())
		}
	case typ.IsMapType():
		out = "map[string]" + processAttributeType(*typ.MapElementType())
	case typ.IsSetType():
		out = "[]" + processAttributeType(*typ.SetElementType())
	case typ.IsListType():
		out = "[]" + processAttributeType(*typ.ListElementType())
	case typ.IsObjectType():
		out = "any"
	default:
		panic("No idea what to do for: " + typ.FriendlyName())
	}

	return out
}

func nestedBlockKeys(block *tfjson.SchemaBlock) []string {
	keys := sortKeys(block.NestedBlocks)

	// Remove TF specific "timeouts" block.
	if i := slices.Index(keys, "timeouts"); i != -1 {
		keys = slices.Delete(keys, i, i+1)
	}

	return keys
}

func nestedField(name []string, k string, isRef bool) field {
	// Collect field properties.
	fieldName := strcase.ToCamel(k)
	fieldTypePrefix := ""
	if isRef {
		fieldTypePrefix = "*"
	} else {
		fieldTypePrefix = "[]"
	}
	fieldType := fmt.Sprintf("%s%s", fieldTypePrefix, strings.Join(append(name, strcase.ToCamel(k)), ""))
	fieldTag := fmt.Sprintf("%s,omitempty", k)

	return field{
		Name: fieldName,
		Type: fieldType,
		Tag:  fieldTag,
	}
}

func (w *walker) walk(block *tfjson.SchemaBlock, name []string) error {
	// Produce nested types before this block itself.
	// This ensures types are defined before they are referenced.
	for _, k := range nestedBlockKeys(block) {
		v := block.NestedBlocks[k]
		err := w.walk(v.Block, append(name, strcase.ToCamel(k)))
		if err != nil {
			return err
		}
	}

	// Declare type.
	typ := structType{
		Name: strings.Join(name, ""),
	}

	// Declare attributes.
	for _, k := range sortKeys(block.Attributes) {
		v := block.Attributes[k]

		// Assert the attribute type is always set.
		if v.AttributeType == cty.NilType && v.AttributeNestedType == nil {
			return fmt.Errorf("unexpected nil type for attribute %s", k)
		}

		// If there is a nested type, walk it and continue to next attribute.
		if v.AttributeNestedType != nil {
			nestedBlock := &tfjson.SchemaBlock{
				Attributes: v.AttributeNestedType.Attributes,
			}
			err := w.walk(nestedBlock, append(name, strcase.ToCamel(k)))
			if err != nil {
				return err
			}
			// Append to list of fields for type.
			typ.Fields = append(typ.Fields, nestedField(name, k, v.AttributeNestedType.NestingMode == tfjson.SchemaNestingModeSingle))
			continue
		}

		// Collect field properties.
		fieldName := strcase.ToCamel(k)
		fieldType := processAttributeType(v.AttributeType)
		fieldTag := k
		if v.Required && v.Optional {
			return fmt.Errorf("both required and optional are set for attribute %s", k)
		}
		if !v.Required {
			fieldTag = fmt.Sprintf("%s,omitempty", fieldTag)
		}

		// Append to list of fields for type.
		typ.Fields = append(typ.Fields, field{
			Name: fieldName,
			Type: fieldType,
			Tag:  fieldTag,
		})
	}

	// Declare nested blocks.
	for _, k := range nestedBlockKeys(block) {
		v := block.NestedBlocks[k]
		// Append to list of fields for type.
		typ.Fields = append(typ.Fields, nestedField(name, k, v.MaxItems == 1))
	}

	// Append type to list of structs.
	w.StructTypes = append(w.StructTypes, typ)
	return nil
}

// walk recursively traverses [tfjson.SchemaBlock] and returns the
// set of types to declare to represents it as Go structs.
func walk(block *tfjson.SchemaBlock, name []string) (*walker, error) {
	w := &walker{}
	err := w.walk(block, name)
	return w, err
}
