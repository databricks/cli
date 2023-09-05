package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderedProperties(t *testing.T) {
	newInt := func(i int) *int {
		return &i
	}

	s := Schema{
		Properties: map[string]*Schema{
			"bbb": {
				Type: StringType,
			},
			"ccc": {
				Type: StringType,
			},
			"ddd": {
				Type: StringType,
			},
			"zzz1": {
				Type: StringType,
				Extension: Extension{
					Order: newInt(-1),
				},
			},
			"zzz2": {
				Type: StringType,
				Extension: Extension{
					Order: newInt(-2),
				},
			},
			"aaa1": {
				Type: StringType,
				Extension: Extension{
					Order: newInt(1),
				},
			},
			"aaa2": {
				Type: StringType,
				Extension: Extension{
					Order: newInt(2),
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

	assert.Equal(t, []string{"zzz2", "zzz1", "aaa1", "aaa2", "bbb", "ccc", "ddd"}, names)
}
