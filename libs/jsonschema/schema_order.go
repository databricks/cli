package jsonschema

import (
	"slices"
	"strings"
)

// Property defines a single property of a struct schema.
// This type is not used in the schema itself but rather to
// return the pair of a property name and its schema.
type Property struct {
	Name   string
	Schema *Schema
}

// OrderedProperties returns the properties of the schema ordered according
// to the value of their `order` extension. If this extension is not set, the
// properties are ordered alphabetically.
func (s *Schema) OrderedProperties() []Property {
	order := make(map[string]int)
	out := make([]Property, 0, len(s.Properties))
	for key, property := range s.Properties {
		order[key] = property.Order
		out = append(out, Property{
			Name:   key,
			Schema: property,
		})
	}

	// Sort the properties by order and then by name.
	slices.SortFunc(out, func(a, b Property) int {
		k := order[a.Name] - order[b.Name]
		if k != 0 {
			return k
		}
		return strings.Compare(a.Name, b.Name)
	})

	return out
}
