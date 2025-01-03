package template

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateConfigAssignValuesFromFile(t *testing.T) {
	testDir := "./testdata/config-assign-from-file"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	err = c.assignValuesFromFile(filepath.Join(testDir, "config.json"))
	if assert.NoError(t, err) {
		assert.Equal(t, int64(1), c.values["int_val"])
		assert.InDelta(t, float64(2), c.values["float_val"].(float64), 0.0001)
		assert.Equal(t, true, c.values["bool_val"])
		assert.Equal(t, "hello", c.values["string_val"])
	}
}

func TestTemplateConfigAssignValuesFromFileDoesNotOverwriteExistingConfigs(t *testing.T) {
	testDir := "./testdata/config-assign-from-file"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	c.values = map[string]any{
		"string_val": "this-is-not-overwritten",
	}

	err = c.assignValuesFromFile(filepath.Join(testDir, "config.json"))
	if assert.NoError(t, err) {
		assert.Equal(t, int64(1), c.values["int_val"])
		assert.InDelta(t, float64(2), c.values["float_val"].(float64), 0.0001)
		assert.Equal(t, true, c.values["bool_val"])
		assert.Equal(t, "this-is-not-overwritten", c.values["string_val"])
	}
}

func TestTemplateConfigAssignValuesFromFileForInvalidIntegerValue(t *testing.T) {
	testDir := "./testdata/config-assign-from-file-invalid-int"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	err = c.assignValuesFromFile(filepath.Join(testDir, "config.json"))
	assert.EqualError(t, err, fmt.Sprintf("failed to load config from file %s: failed to parse property int_val: cannot convert \"abc\" to an integer", filepath.Join(testDir, "config.json")))
}

func TestTemplateConfigAssignValuesFromFileFiltersPropertiesNotInTheSchema(t *testing.T) {
	testDir := "./testdata/config-assign-from-file-unknown-property"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	err = c.assignValuesFromFile(filepath.Join(testDir, "config.json"))
	assert.NoError(t, err)

	// assert only the known property is loaded
	assert.Len(t, c.values, 1)
	assert.Equal(t, "i am a known property", c.values["string_val"])
}

func TestTemplateConfigAssignValuesFromDefaultValues(t *testing.T) {
	testDir := "./testdata/config-assign-from-default-value"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	r, err := newRenderer(ctx, nil, nil, os.DirFS("."), "./testdata/empty/template", "./testdata/empty/library")
	require.NoError(t, err)

	err = c.assignDefaultValues(r)
	if assert.NoError(t, err) {
		assert.Equal(t, int64(123), c.values["int_val"])
		assert.InDelta(t, float64(123), c.values["float_val"].(float64), 0.0001)
		assert.Equal(t, true, c.values["bool_val"])
		assert.Equal(t, "hello", c.values["string_val"])
	}
}

func TestTemplateConfigAssignValuesFromTemplatedDefaultValues(t *testing.T) {
	testDir := "./testdata/config-assign-from-templated-default-value"

	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS(testDir), "schema.json")
	require.NoError(t, err)

	r, err := newRenderer(ctx, nil, nil, os.DirFS("."), path.Join(testDir, "template/template"), path.Join(testDir, "template/library"))
	require.NoError(t, err)

	// Note: only the string value is templated.
	// The JSON schema package doesn't allow using a string default for integer types.
	err = c.assignDefaultValues(r)
	if assert.NoError(t, err) {
		assert.Equal(t, int64(123), c.values["int_val"])
		assert.InDelta(t, float64(123), c.values["float_val"].(float64), 0.0001)
		assert.Equal(t, true, c.values["bool_val"])
		assert.Equal(t, "world", c.values["string_val"])
	}
}

func TestTemplateConfigValidateValuesDefined(t *testing.T) {
	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS("testdata/config-test-schema"), "test-schema.json")
	require.NoError(t, err)

	c.values = map[string]any{
		"int_val":   1,
		"float_val": 1.0,
		"bool_val":  false,
	}

	err = c.validate()
	assert.EqualError(t, err, "validation for template input parameters failed. no value provided for required property string_val")
}

func TestTemplateConfigValidateTypeForValidConfig(t *testing.T) {
	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS("testdata/config-test-schema"), "test-schema.json")
	require.NoError(t, err)

	c.values = map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abcd",
	}

	err = c.validate()
	assert.NoError(t, err)
}

func TestTemplateConfigValidateTypeForUnknownField(t *testing.T) {
	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS("testdata/config-test-schema"), "test-schema.json")
	require.NoError(t, err)

	c.values = map[string]any{
		"unknown_prop": 1,
		"int_val":      1,
		"float_val":    1.1,
		"bool_val":     true,
		"string_val":   "abcd",
	}

	err = c.validate()
	assert.EqualError(t, err, "validation for template input parameters failed. property unknown_prop is not defined in the schema")
}

func TestTemplateConfigValidateTypeForInvalidType(t *testing.T) {
	ctx := context.Background()
	c, err := newConfig(ctx, os.DirFS("testdata/config-test-schema"), "test-schema.json")
	require.NoError(t, err)

	c.values = map[string]any{
		"int_val":    "this-should-be-an-int",
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abcd",
	}

	err = c.validate()
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

func TestTemplateValidateSchemaVersion(t *testing.T) {
	version := latestSchemaVersion
	schema := jsonschema.Schema{
		Extension: jsonschema.Extension{
			Version: &version,
		},
	}
	assert.NoError(t, validateSchema(&schema))

	version = latestSchemaVersion + 1
	schema = jsonschema.Schema{
		Extension: jsonschema.Extension{
			Version: &version,
		},
	}
	assert.EqualError(t, validateSchema(&schema), fmt.Sprintf("template schema version %d is not supported by this version of the CLI. Please upgrade your CLI to the latest version", version))

	version = 5000
	schema = jsonschema.Schema{
		Extension: jsonschema.Extension{
			Version: &version,
		},
	}
	assert.EqualError(t, validateSchema(&schema), "template schema version 5000 is not supported by this version of the CLI. Please upgrade your CLI to the latest version")

	version = 0
	schema = jsonschema.Schema{
		Extension: jsonschema.Extension{
			Version: &version,
		},
	}
	assert.NoError(t, validateSchema(&schema))
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

func TestTemplateSchemaErrorsWithEmptyDescription(t *testing.T) {
	ctx := context.Background()
	_, err := newConfig(ctx, os.DirFS("./testdata/config-test-schema"), "invalid-test-schema.json")
	assert.EqualError(t, err, "template property property-without-description is missing a description")
}

func testRenderer() *renderer {
	return &renderer{
		config: map[string]any{
			"fruit": "apples",
		},
		baseTemplate: template.New(""),
	}
}

func TestPromptIsSkippedWhenEmpty(t *testing.T) {
	c := config{
		ctx:    context.Background(),
		values: make(map[string]any),
		schema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"always-skip": {
					Type:    "string",
					Default: "I like {{.fruit}}",
					Extension: jsonschema.Extension{
						SkipPromptIf: &jsonschema.Schema{},
					},
				},
			},
		},
	}

	// We should always skip the prompt here. An empty JSON schema by definition
	// matches all possible configurations.
	skip, err := c.skipPrompt(jsonschema.Property{
		Name:   "always-skip",
		Schema: c.schema.Properties["always-skip"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "I like apples", c.values["always-skip"])
}

func TestPromptSkipErrorsWithEmptyDefault(t *testing.T) {
	c := config{
		ctx:    context.Background(),
		values: make(map[string]any),
		schema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"no-default": {
					Type: "string",
					Extension: jsonschema.Extension{
						SkipPromptIf: &jsonschema.Schema{},
					},
				},
			},
		},
	}

	_, err := c.skipPrompt(jsonschema.Property{
		Name:   "no-default",
		Schema: c.schema.Properties["no-default"],
	}, testRenderer())
	assert.EqualError(t, err, "property no-default has skip_prompt_if set but no default value")
}

func TestPromptIsSkippedIfValueIsAssigned(t *testing.T) {
	c := config{
		ctx:    context.Background(),
		values: make(map[string]any),
		schema: &jsonschema.Schema{
			Properties: map[string]*jsonschema.Schema{
				"already-assigned": {
					Type:    "string",
					Default: "some-default-value",
				},
			},
		},
	}

	c.values["already-assigned"] = "some-value"
	skip, err := c.skipPrompt(jsonschema.Property{
		Name:   "already-assigned",
		Schema: c.schema.Properties["already-assigned"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "some-value", c.values["already-assigned"])
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
							Required: []string{"abc", "def"},
						},
					},
				},
			},
		},
	}

	// No skip condition defined. Prompt should not be skipped.
	skip, err := c.skipPrompt(jsonschema.Property{
		Name:   "abc",
		Schema: c.schema.Properties["abc"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)

	// No values assigned to config. Prompt should not be skipped.
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)
	assert.NotContains(t, c.values, "xyz")

	// Values do not match skip condition. Prompt should not be skipped.
	c.values["abc"] = "foo"
	c.values["def"] = 123
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)
	assert.NotContains(t, c.values, "xyz")

	// Values do not match skip condition. Prompt should not be skipped.
	c.values["abc"] = "foobar"
	c.values["def"] = 1234
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)
	assert.NotContains(t, c.values, "xyz")

	// Values match skip condition. Prompt should be skipped. Default value should
	// be assigned to "xyz".
	c.values["abc"] = "foobar"
	c.values["def"] = 123
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "hello-world", c.values["xyz"])
}

func TestPromptIsSkippedAnyOf(t *testing.T) {
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
							AnyOf: []jsonschema.Schema{
								{
									Properties: map[string]*jsonschema.Schema{
										"abc": {
											Const: "foobar",
										},
										"def": {
											Const: 123,
										},
									},
									Required: []string{"abc", "def"},
								},
								{
									Properties: map[string]*jsonschema.Schema{
										"abc": {
											Const: "barfoo",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// No skip condition defined. Prompt should not be skipped.
	skip, err := c.skipPrompt(jsonschema.Property{
		Name:   "abc",
		Schema: c.schema.Properties["abc"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)

	// Values do not match skip condition. Prompt should not be skipped.
	c.values = map[string]any{
		"abc": "foobar",
		"def": 1234,
	}
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)
	assert.NotContains(t, c.values, "xyz")

	// def is missing value. Prompt should not be skipped.
	c.values = map[string]any{
		"abc": "foobar",
	}
	_, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.False(t, skip)
	assert.NotContains(t, c.values, "xyz")

	// abc is missing value. Prompt should be skipped because abc is optional
	// in second condition.
	c.values = map[string]any{
		"def": 123,
	}
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "hello-world", c.values["xyz"])

	// Values match skip condition. Prompt should be skipped. Default value should
	// be assigned to "xyz".
	c.values = map[string]any{
		"abc": "foobar",
		"def": 123,
	}
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "hello-world", c.values["xyz"])

	// Values match skip condition. Prompt should be skipped. Default value should
	// be assigned to "xyz".
	c.values = map[string]any{
		"abc": "barfoo",
	}
	skip, err = c.skipPrompt(jsonschema.Property{
		Name:   "xyz",
		Schema: c.schema.Properties["xyz"],
	}, testRenderer())
	assert.NoError(t, err)
	assert.True(t, skip)
	assert.Equal(t, "hello-world", c.values["xyz"])
}
