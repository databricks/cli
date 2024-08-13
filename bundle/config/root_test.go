package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/variable"
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
	root.initializeDynamicValue()
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
				Default: map[string]interface{}{
					"key": "value",
				},
			},
		},
		Targets: map[string]*Target{
			"development": {
				Variables: map[string]*variable.Variable{
					"foo": {
						Default:     "bar",
						Description: "wrong",
					},
					"complex": {
						Type:        "wrong",
						Description: "wrong",
						Default: map[string]interface{}{
							"key1": "value1",
						},
					},
				},
			},
		},
	}
	root.initializeDynamicValue()
	require.NoError(t, root.MergeTargetOverrides("development"))
	assert.Equal(t, "bar", root.Variables["foo"].Default)
	assert.Equal(t, "foo var", root.Variables["foo"].Description)

	assert.Equal(t, "foo2", root.Variables["foo2"].Default)
	assert.Equal(t, "foo2 var", root.Variables["foo2"].Description)

	assert.Equal(t, map[string]interface{}{
		"key1": "value1",
	}, root.Variables["complex"].Default)
	assert.Equal(t, "complex var", root.Variables["complex"].Description)

}
