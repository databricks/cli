package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetVariableFromProcessEnvVar(t *testing.T) {
	defaultVal := "default"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     &defaultVal,
	}

	// set value for variable as an environment variable
	t.Setenv("BUNDLE_VAR_foo", "process-env")

	err := setVariable(&variable, "foo")
	require.NoError(t, err)
	assert.Equal(t, *variable.Value, "process-env")
}

func TestSetVariableUsingDefaultValue(t *testing.T) {
	defaultVal := "default"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     &defaultVal,
	}

	err := setVariable(&variable, "foo")
	require.NoError(t, err)
	assert.Equal(t, *variable.Value, "default")
}

func TestSetVariableWhenAlreadyAValueIsAssigned(t *testing.T) {
	defaultVal := "default"
	val := "assigned-value"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     &defaultVal,
		Value:       &val,
	}

	// since a value is already assigned to the variable, it would not be overridden
	// by the default value
	err := setVariable(&variable, "foo")
	require.NoError(t, err)
	assert.Equal(t, *variable.Value, "assigned-value")
}

func TestSetVariableEnvVarValueDoesNotOverridePresetValue(t *testing.T) {
	defaultVal := "default"
	val := "assigned-value"
	variable := variable.Variable{
		Description: "a test variable",
		Default:     &defaultVal,
		Value:       &val,
	}

	// set value for variable as an environment variable
	t.Setenv("BUNDLE_VAR_foo", "process-env")

	// since a value is already assigned to the variable, it would not be overridden
	// by the value from environment
	err := setVariable(&variable, "foo")
	require.NoError(t, err)
	assert.Equal(t, *variable.Value, "assigned-value")
}

func TestSetVariablesErrorsIfAValueCouldNotBeResolved(t *testing.T) {
	variable := variable.Variable{
		Description: "a test variable with no default",
	}

	// fails because we could not resolve a value for the variable
	err := setVariable(&variable, "foo")
	assert.ErrorContains(t, err, "no value assigned to required variable foo. Assignment can be done through the \"--var\" flag or by setting the BUNDLE_VAR_foo environment variable")
}

func TestSetVariablesMutator(t *testing.T) {
	defaultValForA := "default-a"
	defaultValForB := "default-b"
	valForC := "assigned-val-c"
	bundle := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"a": {
					Description: "resolved to default value",
					Default:     &defaultValForA,
				},
				"b": {
					Description: "resolved from environment vairables",
					Default:     &defaultValForB,
				},
				"c": {
					Description: "has already been assigned a value",
					Value:       &valForC,
				},
			},
		},
	}

	t.Setenv("BUNDLE_VAR_b", "env-var-b")

	err := SetVariables().Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "default-a", *bundle.Config.Variables["a"].Value)
	assert.Equal(t, "env-var-b", *bundle.Config.Variables["b"].Value)
	assert.Equal(t, "assigned-val-c", *bundle.Config.Variables["c"].Value)
}
