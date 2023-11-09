package template

import (
	"context"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T) *config {
	c, err := newConfig(context.Background(), "./testdata/config-test-schema/test-schema.json")
	require.NoError(t, err)
	return c
}

func TestTemplateConfigAssignValuesFromFile(t *testing.T) {
	c := testConfig(t)

	err := c.assignValuesFromFile("./testdata/config-assign-from-file/config.json")
	assert.NoError(t, err)

	assert.Equal(t, int64(1), c.values["int_val"])
	assert.Equal(t, float64(2), c.values["float_val"])
	assert.Equal(t, true, c.values["bool_val"])
	assert.Equal(t, "hello", c.values["string_val"])
}

func TestTemplateConfigAssignValuesFromFileForInvalidIntegerValue(t *testing.T) {
	c := testConfig(t)

	err := c.assignValuesFromFile("./testdata/config-assign-from-file-invalid-int/config.json")
	assert.EqualError(t, err, "failed to load config from file ./testdata/config-assign-from-file-invalid-int/config.json: failed to parse property int_val: cannot convert \"abc\" to an integer")
}

func TestTemplateConfigAssignValuesFromFileDoesNotOverwriteExistingConfigs(t *testing.T) {
	c := testConfig(t)
	c.values = map[string]any{
		"string_val": "this-is-not-overwritten",
	}

	err := c.assignValuesFromFile("./testdata/config-assign-from-file/config.json")
	assert.NoError(t, err)

	assert.Equal(t, int64(1), c.values["int_val"])
	assert.Equal(t, float64(2), c.values["float_val"])
	assert.Equal(t, true, c.values["bool_val"])
	assert.Equal(t, "this-is-not-overwritten", c.values["string_val"])
}

func TestTemplateConfigAssignDefaultValues(t *testing.T) {
	c := testConfig(t)

	ctx := context.Background()
	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/template-in-path/template", "./testdata/template-in-path/library", t.TempDir())
	require.NoError(t, err)

	err = c.assignDefaultValues(r)
	assert.NoError(t, err)

	assert.Len(t, c.values, 2)
	assert.Equal(t, "my_file", c.values["string_val"])
	assert.Equal(t, int64(123), c.values["int_val"])
}

func TestTemplateConfigValidateValuesDefined(t *testing.T) {
	c := testConfig(t)
	c.values = map[string]any{
		"int_val":   1,
		"float_val": 1.0,
		"bool_val":  false,
	}

	err := c.validate()
	assert.EqualError(t, err, "validation for template input parameters failed. no value provided for required property string_val")
}

func TestTemplateConfigValidateTypeForValidConfig(t *testing.T) {
	c := testConfig(t)
	c.values = map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abcd",
	}

	err := c.validate()
	assert.NoError(t, err)
}

func TestTemplateConfigValidateTypeForUnknownField(t *testing.T) {
	c := testConfig(t)
	c.values = map[string]any{
		"unknown_prop": 1,
		"int_val":      1,
		"float_val":    1.1,
		"bool_val":     true,
		"string_val":   "abcd",
	}

	err := c.validate()
	assert.EqualError(t, err, "validation for template input parameters failed. property unknown_prop is not defined in the schema")
}

func TestTemplateConfigValidateTypeForInvalidType(t *testing.T) {
	c := testConfig(t)
	c.values = map[string]any{
		"int_val":    "this-should-be-an-int",
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abcd",
	}

	err := c.validate()
	assert.EqualError(t, err, "validation for template input parameters failed. incorrect type for property int_val: expected type integer, but value is \"this-should-be-an-int\"")
}

func TestTemplateValidateSchema(t *testing.T) {
	var err error
	toSchema := func(s string) *jsonschema.Schema {
		return &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"foo": {
					Type: jsonschema.Type(s),
				},
			},
		}
	}

	err = validateSchema(toSchema("string"))
	assert.NoError(t, err)

	err = validateSchema(toSchema("boolean"))
	assert.NoError(t, err)

	err = validateSchema(toSchema("number"))
	assert.NoError(t, err)

	err = validateSchema(toSchema("integer"))
	assert.NoError(t, err)

	err = validateSchema(toSchema("object"))
	assert.EqualError(t, err, "property type object is not supported by bundle templates")

	err = validateSchema(toSchema("array"))
	assert.EqualError(t, err, "property type array is not supported by bundle templates")
}

func TestTemplateEnumValidation(t *testing.T) {
	schema := jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"abc": {
				Type: "integer",
				Enum: []any{1, 2, 3, 4},
			},
		},
	}

	c := &config{
		schema: &schema,
		values: map[string]any{
			"abc": 5,
		},
	}
	assert.EqualError(t, c.validate(), "validation for template input parameters failed. expected value of property abc to be one of [1 2 3 4]. Found: 5")

	c = &config{
		schema: &schema,
		values: map[string]any{
			"abc": 4,
		},
	}
	assert.NoError(t, c.validate())
}

func TestAssignDefaultValuesWithTemplatedDefaults(t *testing.T) {
	c := testConfig(t)
	ctx := context.Background()
	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/templated-defaults/template", "./testdata/templated-defaults/library", t.TempDir())
	require.NoError(t, err)

	err = c.assignDefaultValues(r)
	assert.NoError(t, err)
	assert.Equal(t, "my_file", c.values["string_val"])
}

func TestTemplateSchemaErrorsWithEmptyDescription(t *testing.T) {
	_, err := newConfig(context.Background(), "./testdata/config-test-schema/invalid-test-schema.json")
	assert.EqualError(t, err, "template property property-without-description is missing a description")
}

func TestPromptIsSkipped(t *testing.T) {
	c := config{
		ctx:    context.Background(),
		values: make(map[string]any),
		schema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"abc": {
					Type: "string",
				},
				"def": {
					Type: "integer",
				},
				"xyz": {
					Type:    "string",
					Default: "hello-world",
					Extension: jsonschema.Extension{
						SkipPromptIf: &jsonschema.Schema{
							Properties: map[string]*jsonschema.Schema{
								"abc": {
									Const: "foobar",
								},
								"def": {
									Const: 123,
								},
							},
						},
					},
				},
			},
		},
	}

	// No skip condition defined. Prompt should not be skipped.
	assert.False(t, c.isSkipped(jsonschema.Property{
		Name:   "abc",
		Schema: c.schema.Properties["abc"],
	}))

	// No values assigned to config. Prompt should not be skipped.
	assert.False(t, c.isSkipped(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}))
	assert.NotContains(t, c.values, "xyz")

	// Values do not match skip condition. Prompt should not be skipped.
	c.values["abc"] = "foo"
	c.values["def"] = 123
	assert.False(t, c.isSkipped(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}))
	assert.NotContains(t, c.values, "xyz")

	// Values do not match skip condition. Prompt should not be skipped.
	c.values["abc"] = "foobar"
	c.values["def"] = 1234
	assert.False(t, c.isSkipped(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}))
	assert.NotContains(t, c.values, "xyz")

	// Values match skip condition. Prompt should be skipped. Default value should
	// be assigned to "xyz".
	c.values["abc"] = "foobar"
	c.values["def"] = 123
	assert.True(t, c.isSkipped(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}))
	assert.Equal(t, "hello-world", c.values["xyz"])
}
