package golang

import (
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

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
	default:
		panic("No idea what to do for: " + typ.FriendlyName())
	}

	return out
}

func SortedKeys[M ~map[K]V, K string, V any](m M) []K {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}

func processBlock(in *tfjson.SchemaBlock) {
	fmt.Println("struct {")

	for _, k := range SortedKeys(in.Attributes) {
		v := in.Attributes[k]
		if v.Deprecated {
			continue
		}

		// if v.Required {
		// 	out.Required = append(out.Required, k)
		// }

		annotations := []string{k}
		if !v.Required {
			annotations = append(annotations, "omitempty")
		}

		fmt.Print(strcase.ToCamel(k))
		fmt.Print(" ")
		fmt.Print(processAttributeType(v.AttributeType))
		fmt.Print(" ")
		fmt.Print("`")
		fmt.Print("json:\"")
		fmt.Print(strings.Join(annotations, ","))
		fmt.Print("\"`\n")
	}

	fmt.Print("\n")

	for _, k := range SortedKeys(in.NestedBlocks) {
		v := in.NestedBlocks[k]
		annotations := []string{k, "omitempty"}
		// if v.MinItems == 0 {
		// 	annotations = append(annotations, "omitempty")
		// }

		fmt.Print(strcase.ToCamel(k))
		fmt.Print(" ")

		switch v.NestingMode {
		case tfjson.SchemaNestingModeSingle:
			fmt.Print("*")
		case tfjson.SchemaNestingModeGroup:
			fmt.Print("*")
		case tfjson.SchemaNestingModeList:
			if v.MaxItems == 1 {
				fmt.Print("*")
			} else {
				fmt.Print("[]")
			}
		case tfjson.SchemaNestingModeSet:
			fmt.Print("[]")
		case tfjson.SchemaNestingModeMap:
			panic("map")
		}

		processBlock(v.Block)

		fmt.Print(" ")
		fmt.Print("`")
		fmt.Print("json:\"")
		fmt.Print(strings.Join(annotations, ","))
		fmt.Print("\"`\n")
		fmt.Print("\n")
	}

	// out.Description = in.Description
	// return &out

	fmt.Print("}")
	// fmt.Println("")
}

func Generate(schema *tfjson.ProviderSchema) error {
	fmt.Println("package tf")

	for _, k := range SortedKeys(schema.ResourceSchemas) {
		v := schema.ResourceSchemas[k]
		k = strcase.ToCamel(strings.TrimPrefix(k, "databricks_"))
		fmt.Printf("type %s ", k)
		processBlock(v.Block)
		fmt.Print("\n")
	}

	// Union of all of them

	fmt.Println("type Resources struct {")

	for _, k := range SortedKeys(schema.ResourceSchemas) {
		resource_name := k
		k = strcase.ToCamel(strings.TrimPrefix(k, "databricks_"))

		fmt.Print(k)
		fmt.Print(" ")
		fmt.Print("map[string]" + k)
		fmt.Print(" ")
		fmt.Print("`")
		fmt.Print("json:\"")
		fmt.Printf("%s,omitempty", resource_name)
		fmt.Print("\"`\n")
	}

	fmt.Print("}")
	fmt.Print("\n")

	return nil
}
