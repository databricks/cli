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

	err = validate(toSchema("string"))
	assert.NoError(t, err)

	err = validate(toSchema("boolean"))
	assert.NoError(t, err)

	err = validate(toSchema("number"))
	assert.NoError(t, err)

	err = validate(toSchema("integer"))
	assert.NoError(t, err)

	err = validate(toSchema("int"))
	assert.EqualError(t, err, "type int is not a recognized json schema type. Please use \"integer\" instead")

	err = validate(toSchema("float"))
	assert.EqualError(t, err, "type float is not a recognized json schema type. Please use \"number\" instead")

	err = validate(toSchema("bool"))
	assert.EqualError(t, err, "type bool is not a recognized json schema type. Please use \"boolean\" instead")

	err = validate(toSchema("foobar"))
	assert.EqualError(t, err, "type foobar is not a recognized json schema type")
}
