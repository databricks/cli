package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderedProperties(t *testing.T) {
	s := Schema{
		Properties: map[string]*Schema{
			"bbb": {
				Type: StringType,
				// Implied order: 0
			},
			"ccc": {
				Type: StringType,
				// Implied order: 0
			},
			"ddd": {
				Type: StringType,
				// Implied order: 0
			},
			"zzz1": {
				Type: StringType,
				Extension: Extension{
					Order: -1,
				},
			},
			"zzz2": {
				Type: StringType,
				Extension: Extension{
					Order: -2,
				},
			},
			"aaa1": {
				Type: StringType,
				Extension: Extension{
					Order: 1,
				},
			},
			"aaa2": {
				Type: StringType,
				Extension: Extension{
					Order: 2,
				},
			},
		},
	}

	// Test that the properties are ordered by order and then by name.
	properties := s.OrderedProperties()
	names := make([]string, len(properties))
	for i, property := range properties {
		names[i] = property.Name
	}

	assert.Equal(t, []string{"zzz2", "zzz1", "bbb", "ccc", "ddd", "aaa1", "aaa2"}, names)
}
