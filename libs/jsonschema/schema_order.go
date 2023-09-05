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
	order := make(map[string]*int)
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
		oa := order[a.Name]
		ob := order[b.Name]
		cmp := 0
		switch {
		case oa != nil && ob != nil:
			// Compare the order values if both are set.
			cmp = *oa - *ob
		case oa == nil && ob != nil:
			// If only one is set, the one that is set comes first.
			cmp = 1
		case oa != nil && ob == nil:
			// If only one is set, the one that is set comes first.
			cmp = -1
		}

		// If we have a non-zero comparison, return it.
		if cmp != 0 {
			return cmp
		}

		// If the order is the same, compare by name.
		return strings.Compare(a.Name, b.Name)
	})

	return out
}
