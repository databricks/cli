package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetVariableFromProcessEnvVar(t *testing.T) {
	defaultVal := "default"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     defaultVal,
	}

	// set value for variable as an environment variable
	t.Setenv("BUNDLE_VAR_foo", "process-env")
	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	v, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	require.NoError(t, err)

	err = convert.ToTyped(&variable, v)
	require.NoError(t, err)
	assert.Equal(t, "process-env", variable.Value)
}

func TestSetVariableUsingDefaultValue(t *testing.T) {
	defaultVal := "default"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     defaultVal,
	}

	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	v, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	require.NoError(t, err)

	err = convert.ToTyped(&variable, v)
	require.NoError(t, err)
	assert.Equal(t, "default", variable.Value)
}

func TestSetVariableWhenAlreadyAValueIsAssigned(t *testing.T) {
	defaultVal := "default"
	val := "assigned-value"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     defaultVal,
		Value:       val,
	}

	// since a value is already assigned to the variable, it would not be overridden
	// by the default value
	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	v, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	require.NoError(t, err)

	err = convert.ToTyped(&variable, v)
	require.NoError(t, err)
	assert.Equal(t, "assigned-value", variable.Value)
}

func TestSetVariableEnvVarValueDoesNotOverridePresetValue(t *testing.T) {
	defaultVal := "default"
	val := "assigned-value"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     defaultVal,
		Value:       val,
	}

	// set value for variable as an environment variable
	t.Setenv("BUNDLE_VAR_foo", "process-env")

	// since a value is already assigned to the variable, it would not be overridden
	// by the value from environment
	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	v, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	require.NoError(t, err)

	err = convert.ToTyped(&variable, v)
	require.NoError(t, err)
	assert.Equal(t, "assigned-value", variable.Value)
}

func TestSetVariablesErrorsIfAValueCouldNotBeResolved(t *testing.T) {
	variable := variable.Variable{
		Description: "a test variable with no default",
	}

	// fails because we could not resolve a value for the variable
	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	_, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	assert.ErrorContains(t, err, "no value assigned to required variable foo. Assignment can be done using \"--var\", by setting the BUNDLE_VAR_foo environment variable, or in .databricks/bundle/<target>/variable-overrides.json file")
}

func TestSetVariablesMutator(t *testing.T) {
	defaultValForA := "default-a"
	defaultValForB := "default-b"
	valForC := "assigned-val-c"
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"a": {
					Description: "resolved to default value",
					Default:     defaultValForA,
				},
				"b": {
					Description: "resolved from environment vairables",
					Default:     defaultValForB,
				},
				"c": {
					Description: "has already been assigned a value",
					Value:       valForC,
				},
			},
		},
	}

	t.Setenv("BUNDLE_VAR_b", "env-var-b")

	diags := bundle.Apply(context.Background(), b, SetVariables())
	require.NoError(t, diags.Error())
	assert.Equal(t, "default-a", b.Config.Variables["a"].Value)
	assert.Equal(t, "env-var-b", b.Config.Variables["b"].Value)
	assert.Equal(t, "assigned-val-c", b.Config.Variables["c"].Value)
}

func TestSetComplexVariablesViaEnvVariablesIsNotAllowed(t *testing.T) {
	defaultVal := "default"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     defaultVal,
		Type:        variable.VariableTypeComplex,
	}

	// set value for variable as an environment variable
	t.Setenv("BUNDLE_VAR_foo", "process-env")

	v, err := convert.FromTyped(variable, dyn.NilValue)
	require.NoError(t, err)

	_, err = setVariable(context.Background(), v, &variable, "foo", dyn.NilValue)
	assert.ErrorContains(t, err, "setting via environment variables (BUNDLE_VAR_foo) is not supported for complex variable foo")
}
