package jsonschema

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaValidateTypeNames(t *testing.T) {
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

func TestSchemaLoadIntegers(t *testing.T) {
	schema, err := Load("./testdata/schema-load-int/schema-valid.json")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), schema.Properties["abc"].Default)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, schema.Properties["abc"].Enum)
	assert.Equal(t, int64(5), schema.Properties["def"].Const)
}

func TestSchemaLoadIntegersWithInvalidDefault(t *testing.T) {
	_, err := Load("./testdata/schema-load-int/schema-invalid-default.json")
	assert.EqualError(t, err, "failed to parse default value for property abc: expected integer value, got: 1.1")
}

func TestSchemaLoadIntegersWithInvalidEnums(t *testing.T) {
	_, err := Load("./testdata/schema-load-int/schema-invalid-enum.json")
	assert.EqualError(t, err, "failed to parse enum value 2.4 at index 1 for property abc: expected integer value, got: 2.4")
}

func TestSchemaLoadIntergersWithInvalidConst(t *testing.T) {
	_, err := Load("./testdata/schema-load-int/schema-invalid-const.json")
	assert.EqualError(t, err, "failed to parse const value for property def: expected integer value, got: 5.1")
}

func TestSchemaValidateDefaultType(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "number",
				Default: "abc",
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "type validation for default value of property foo failed: expected type float, but value is \"abc\"")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "boolean",
				Default: true,
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}

func TestSchemaValidateEnumType(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type: "boolean",
				Enum: []any{true, "false"},
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "type validation for enum at index 1 failed for property foo: expected type boolean, but value is \"false\"")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type: "boolean",
				Enum: []any{true, false},
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}

func TestSchemaValidateErrorWhenDefaultValueIsNotInEnums(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Default: "abc",
				Enum:    []any{"def", "ghi"},
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "list of enum values for property foo does not contain default value abc: [def ghi]")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Default: "abc",
				Enum:    []any{"def", "ghi", "abc"},
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}

func TestSchemaValidatePatternType(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "number",
				Pattern: "abc",
			},
		},
	}
	assert.EqualError(t, s.validate(), "property \"foo\" has a non-empty regex pattern \"abc\" specified. Patterns are only supported for string properties")

	s = &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Pattern: "abc",
			},
		},
	}
	assert.NoError(t, s.validate())
}

func TestSchemaValidateIncorrectRegex(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type: "string",
				// invalid regex, missing the closing brace
				Pattern: "(abc",
			},
		},
	}
	assert.EqualError(t, s.validate(), "invalid regex pattern \"(abc\" provided for property \"foo\": error parsing regexp: missing closing ): `(abc`")
}

func TestSchemaDefaultValueIsNotValidatedAgainstPattern(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Pattern: "abc",
				Default: "def",
			},
		},
	}
	assert.NoError(t, s.validate())
}

func TestSchemaValidatePatternEnum(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Pattern: "a.*c",
				Enum:    []any{"abc", "def", "abbc"},
			},
		},
	}
	assert.EqualError(t, s.validate(), "enum value \"def\" at index 1 for property \"foo\" does not match specified regex pattern: \"a.*c\"")

	s = &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Pattern: "a.*d",
				Enum:    []any{"abd", "axybgd", "abbd"},
			},
		},
	}
	assert.NoError(t, s.validate())
}

func TestValidateSchemaMinimumCliVersionWithInvalidSemver(t *testing.T) {
	s := &Schema{
		Extension: Extension{
			MinDatabricksCliVersion: "1.0.5",
		},
	}
	err := s.validateSchemaMinimumCliVersion("v2.0.1")()
	assert.ErrorContains(t, err, "invalid minimum CLI version \"1.0.5\" specified. Please specify the version in the format v0.0.0")

	s.MinDatabricksCliVersion = "v1.0.5"
	err = s.validateSchemaMinimumCliVersion("v2.0.1")()
	assert.NoError(t, err)
}

func TestValidateSchemaMinimumCliVersion(t *testing.T) {
	s := &Schema{
		Extension: Extension{
			MinDatabricksCliVersion: "v1.0.5",
		},
	}
	err := s.validateSchemaMinimumCliVersion("v2.0.1")()
	assert.NoError(t, err)

	err = s.validateSchemaMinimumCliVersion("v1.0.5")()
	assert.NoError(t, err)

	err = s.validateSchemaMinimumCliVersion("v1.0.6")()
	assert.NoError(t, err)

	err = s.validateSchemaMinimumCliVersion("v1.0.4")()
	assert.ErrorContains(t, err, `minimum CLI version "v1.0.5" is greater than current CLI version "v1.0.4". Please upgrade your current Databricks CLI`)

	err = s.validateSchemaMinimumCliVersion("v0.0.1")()
	assert.ErrorContains(t, err, "minimum CLI version \"v1.0.5\" is greater than current CLI version \"v0.0.1\". Please upgrade your current Databricks CLI")

	err = s.validateSchemaMinimumCliVersion("v0.0.0-dev")()
	assert.NoError(t, err)
}

func TestValidateSchemaConstTypes(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:  "string",
				Const: "abc",
			},
		},
	}
	err := s.validate()
	assert.NoError(t, err)

	s = &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:  "string",
				Const: 123,
			},
		},
	}
	err = s.validate()
	assert.EqualError(t, err, "type validation for const value of property foo failed: expected type string, but value is 123")
}

func TestValidateSchemaSkippedPropertiesHaveDefaults(t *testing.T) {
	s := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:      "string",
				Extension: Extension{SkipPromptIf: &Schema{}},
			},
		},
	}
	err := s.validate()
	assert.EqualError(t, err, "property \"foo\" has a skip_prompt_if clause but no default value")

	s = &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:      "string",
				Default:   "abc",
				Extension: Extension{SkipPromptIf: &Schema{}},
			},
		},
	}
	err = s.validate()
	assert.NoError(t, err)
}

func TestSchema_LoadFS(t *testing.T) {
	fsys := os.DirFS("./testdata/schema-load-int")
	_, err := LoadFS(fsys, "schema-valid.json")
	assert.NoError(t, err)
}

func TestSchemaGetDefinition(t *testing.T) {
	s := &Schema{
		Definitions: map[string]any{
			"foo": map[string]any{
				"type": "string",
			},
			"bar": map[string]any{
				"baz": map[string]any{
					"type": "integer",
				},
			},
			"invalid": "abc",
		},
	}

	ns, err := s.GetDefinition("#/$defs/foo")
	assert.NoError(t, err)
	assert.Equal(t, &Schema{
		Type: "string",
	}, ns)

	ns, err = s.GetDefinition("#/$defs/bar/baz")
	assert.NoError(t, err)
	assert.Equal(t, &Schema{
		Type: "integer",
	}, ns)

	_, err = s.GetDefinition("a/b/c")
	assert.EqualError(t, err, "invalid reference \"a/b/c\". References must start with #/$defs/")

	_, err = s.GetDefinition("#/$defs/invalid/abc")
	assert.EqualError(t, err, "invalid reference \"#/$defs/invalid/abc\". Failed to resolve reference at \"invalid\"")

	_, err = s.GetDefinition("#/$defs")
	assert.EqualError(t, err, "invalid reference \"#/$defs\". Expected more than 2 tokens")
}
