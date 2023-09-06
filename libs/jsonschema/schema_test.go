package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonSchemaValidate(t *testing.T) {
	var err error
	toSchema := func(s string) *Schema {
		return &Schema{
			Properties: map[string]*Schema{
				"foo": {
					Type: Type(s),
				},
			},
		}
	}

	err = toSchema("string").validate()
	assert.NoError(t, err)

	err = toSchema("boolean").validate()
	assert.NoError(t, err)

	err = toSchema("number").validate()
	assert.NoError(t, err)

	err = toSchema("integer").validate()
	assert.NoError(t, err)

	err = toSchema("int").validate()
	assert.EqualError(t, err, "type int is not a recognized json schema type. Please use \"integer\" instead")

	err = toSchema("float").validate()
	assert.EqualError(t, err, "type float is not a recognized json schema type. Please use \"number\" instead")

	err = toSchema("bool").validate()
	assert.EqualError(t, err, "type bool is not a recognized json schema type. Please use \"boolean\" instead")

	err = toSchema("foobar").validate()
	assert.EqualError(t, err, "type foobar is not a recognized json schema type")
}
