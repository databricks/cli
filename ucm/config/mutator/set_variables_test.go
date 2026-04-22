package mutator

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetVariable_FromEnv(t *testing.T) {
	v := variable.Variable{Description: "x", Default: "d"}
	t.Setenv("DATABRICKS_UCM_VAR_foo", "from-env")

	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	dv, err = setVariable(t.Context(), dv, &v, "foo")
	require.NoError(t, err)

	require.NoError(t, convert.ToTyped(&v, dv))
	assert.Equal(t, "from-env", v.Value)
}

func TestSetVariable_FromDefault(t *testing.T) {
	v := variable.Variable{Description: "x", Default: "d"}
	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	dv, err = setVariable(t.Context(), dv, &v, "foo")
	require.NoError(t, err)

	require.NoError(t, convert.ToTyped(&v, dv))
	assert.Equal(t, "d", v.Value)
}

func TestSetVariable_PreserveExistingValue(t *testing.T) {
	v := variable.Variable{Description: "x", Default: "d", Value: "cli"}
	t.Setenv("DATABRICKS_UCM_VAR_foo", "from-env")

	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	dv, err = setVariable(t.Context(), dv, &v, "foo")
	require.NoError(t, err)

	require.NoError(t, convert.ToTyped(&v, dv))
	assert.Equal(t, "cli", v.Value)
}

func TestSetVariable_MissingValueErrors(t *testing.T) {
	v := variable.Variable{Description: "no default"}
	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	_, err = setVariable(t.Context(), dv, &v, "foo")
	assert.ErrorContains(t, err, `no value assigned to required variable foo`)
	assert.ErrorContains(t, err, `DATABRICKS_UCM_VAR_foo`)
}

func TestSetVariable_ComplexEnvRejected(t *testing.T) {
	v := variable.Variable{Description: "x", Default: "d", Type: variable.VariableTypeComplex}
	t.Setenv("DATABRICKS_UCM_VAR_foo", "scalar")

	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	_, err = setVariable(t.Context(), dv, &v, "foo")
	assert.ErrorContains(t, err, "not supported for complex variable foo")
}

func TestSetVariable_LookupLeftUntouched(t *testing.T) {
	v := variable.Variable{Lookup: &variable.Lookup{Metastore: "main"}}
	dv, err := convert.FromTyped(v, dyn.NilValue)
	require.NoError(t, err)
	dv2, err := setVariable(t.Context(), dv, &v, "foo")
	require.NoError(t, err)

	// Nothing assigned — lookup mutator resolves later.
	require.NoError(t, convert.ToTyped(&v, dv2))
	assert.Nil(t, v.Value)
}

func TestSetVariablesMutator_ResolutionLadder(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"a": {Description: "default path", Default: "def-a"},
				"b": {Description: "env path", Default: "def-b"},
				"c": {Description: "pre-set", Value: "cli-c"},
			},
		},
	}
	t.Setenv("DATABRICKS_UCM_VAR_b", "env-b")

	diags := ucm.Apply(t.Context(), u, SetVariables())
	require.NoError(t, diags.Error())
	assert.Equal(t, "def-a", u.Config.Variables["a"].Value)
	assert.Equal(t, "env-b", u.Config.Variables["b"].Value)
	assert.Equal(t, "cli-c", u.Config.Variables["c"].Value)
}
