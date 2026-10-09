package config

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootMarshalUnmarshal(t *testing.T) {
	// Marshal empty
	buf, err := json.Marshal(&Root{})
	require.NoError(t, err)

	// Unmarshal empty
	var root Root
	err = json.Unmarshal(buf, &root)
	require.NoError(t, err)

	// Compare
	assert.True(t, reflect.DeepEqual(Root{}, root))
}

func TestRootLoad(t *testing.T) {
	root, diags := Load("../tests/basic/databricks.yml")
	require.NoError(t, diags.Error())
	assert.Equal(t, "basic", root.Bundle.Name)
}

func TestInitializeVariables(t *testing.T) {
	fooDefault := "abc"
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Default:     fooDefault,
				Description: "an optional variable since default is defined",
			},
			"bar": {
				Description: "a required variable",
			},
		},
	}

	err := root.InitializeVariables([]string{"foo=123", "bar=456"})
	assert.NoError(t, err)
	assert.Equal(t, "123", (root.Variables["foo"].Value))
	assert.Equal(t, "456", (root.Variables["bar"].Value))
}

func TestInitializeVariablesWithAnEqualSignInValue(t *testing.T) {
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Description: "a variable called foo",
			},
		},
	}

	err := root.InitializeVariables([]string{"foo=123=567"})
	assert.NoError(t, err)
	assert.Equal(t, "123=567", (root.Variables["foo"].Value))
}

func TestInitializeVariablesInvalidFormat(t *testing.T) {
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Description: "a variable called foo",
			},
		},
	}

	err := root.InitializeVariables([]string{"foo"})
	assert.ErrorContains(t, err, "unexpected flag value for variable assignment: foo")
}

func TestInitializeVariablesUndefinedVariables(t *testing.T) {
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Description: "A required variable",
			},
		},
	}

	err := root.InitializeVariables([]string{"bar=567"})
	assert.ErrorContains(t, err, "variable bar has not been defined")
}

func TestRootMergeTargetOverridesWithMode(t *testing.T) {
	root := &Root{
		Bundle: Bundle{},
		Targets: map[string]*Target{
			"development": {
				Mode: Development,
			},
		},
	}
	require.NoError(t, root.MergeTargetOverrides("development"))
	assert.Equal(t, Development, root.Bundle.Mode)
}

func TestInitializeComplexVariablesViaFlagIsNotAllowed(t *testing.T) {
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Type: variable.VariableTypeComplex,
			},
		},
	}

	err := root.InitializeVariables([]string{"foo=123"})
	assert.ErrorContains(t, err, "setting variables of complex type via --var flag is not supported: foo")
}

func TestRootMergeTargetOverridesWithVariables(t *testing.T) {
	root := &Root{
		Bundle: Bundle{},
		Variables: map[string]*variable.Variable{
			"foo": {
				Default:     "foo",
				Description: "foo var",
			},
			"foo2": {
				Default:     "foo2",
				Description: "foo2 var",
			},
			"complex": {
				Type:        variable.VariableTypeComplex,
				Description: "complex var",
				Default: map[string]any{
					"key": "value",
				},
			},
		},
		Targets: map[string]*Target{
			"development": {
				Variables: map[string]*variable.TargetVariable{
					"foo": {
						Default:     "bar",
						Description: "wrong",
					},
					"complex": {
						Type:        "wrong",
						Description: "wrong",
						Default: map[string]any{
							"key1": "value1",
						},
					},
				},
			},
		},
	}
	require.NoError(t, root.MergeTargetOverrides("development"))
	assert.Equal(t, "bar", root.Variables["foo"].Default)
	assert.Equal(t, "foo var", root.Variables["foo"].Description)

	assert.Equal(t, "foo2", root.Variables["foo2"].Default)
	assert.Equal(t, "foo2 var", root.Variables["foo2"].Description)

	assert.Equal(t, map[string]any{
		"key1": "value1",
	}, root.Variables["complex"].Default)
	assert.Equal(t, "complex var", root.Variables["complex"].Description)
}

func TestIsFullVariableOverrideDef(t *testing.T) {
	testCases := []struct {
		value    map[string]any
		expected bool
	}{
		{
			value: map[string]any{
				"type":        "string",
				"default":     "foo",
				"description": "foo var",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"type":        "string",
				"lookup":      "foo",
				"description": "foo var",
			},
			expected: false,
		},
		{
			value: map[string]any{
				"type":    "string",
				"default": "foo",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"type":   "string",
				"lookup": "foo",
			},
			expected: false,
		},
		{
			value: map[string]any{
				"description": "string",
				"default":     "foo",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"description": "string",
				"lookup":      "foo",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"default": "foo",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"lookup": "foo",
			},
			expected: true,
		},
		{
			value: map[string]any{
				"type": "string",
			},
			expected: false,
		},
		{
			value: map[string]any{
				"type":        "string",
				"default":     "foo",
				"description": "foo var",
				"lookup":      "foo",
			},
			expected: false,
		},
	}

	for i, tc := range testCases {
		keys := slices.Collect(maps.Keys(tc.value))
		assert.Equal(t, tc.expected, isFullVariableOverrideDef(keys), "test case %d", i)
	}
}

func TestLoadFromBytesNotAMap(t *testing.T) {
	for _, content := range []string{"hello\n", "- a\n- b\n"} {
		r, diags := LoadFromBytes("databricks.yml", []byte(content))
		assert.Nil(t, r)
		assert.ErrorContains(t, diags.Error(), "failed to load databricks.yml")
	}
}

func TestMergeComplexVariableSequenceDefaults(t *testing.T) {
	a, diags := LoadFromBytes("a.yml", []byte("variables:\n  v:\n    type: complex\n    default: [1, 2]\n"))
	require.NoError(t, diags.Error())
	b, diags := LoadFromBytes("b.yml", []byte("variables:\n  v:\n    type: complex\n    default: [3]\n"))
	require.NoError(t, diags.Error())
	require.NoError(t, a.Merge(b))
	assert.Equal(t, []any{1, 2, 3}, a.Variables["v"].Default)
}

func TestTargetVariableShorthandThroughAliases(t *testing.T) {
	r, diags := LoadFromBytes("a.yml", []byte(`
x: &dev dev_value
y: &common {b: from_anchor, c: from_anchor}
variables:
  a: {default: d}
  b: {default: d}
  c: {default: d}
targets:
  dev:
    variables:
      <<: *common
      a: *dev
      c: explicit
`))
	require.NoError(t, diags.Error())
	require.NoError(t, r.MergeTargetOverrides("dev"))
	assert.Equal(t, "dev_value", r.Variables["a"].Default)
	assert.Equal(t, "from_anchor", r.Variables["b"].Default)
	assert.Equal(t, "explicit", r.Variables["c"].Default)
}

func TestNestedYAMLAnchorCycle(t *testing.T) {
	_, diags := LoadFromBytes("a.yml", []byte("variables:\n  v:\n    default: &x {a: [*x]}\n"))
	assert.ErrorContains(t, diags.Error(), `cyclic reference to anchor "x"`)
}

func TestSetReferenceAtPointerField(t *testing.T) {
	r, diags := LoadFromBytes("a.yml", []byte("resources:\n  jobs:\n    j:\n      name: j\n"))
	require.NoError(t, diags.Error())
	path := structpath.MustParsePath("resources.jobs.j.trigger")
	require.NoError(t, r.SetReference(path, "${var.t}"))
	s, ok := r.View().Lookup(path).AsString()
	require.True(t, ok)
	assert.Equal(t, "${var.t}", s)
}
