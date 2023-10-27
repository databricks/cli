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
	root, err := Load("../tests/basic/databricks.yml")
	require.NoError(t, err)
	assert.Equal(t, "basic", root.Bundle.Name)
}

func TestRootMergeStruct(t *testing.T) {
	root := &Root{
		Path: "path",
		Workspace: Workspace{
			Host:    "foo",
			Profile: "profile",
		},
	}
	other := &Root{
		Path: "path",
		Workspace: Workspace{
			Host: "bar",
		},
	}
	assert.NoError(t, root.Merge(other))
	assert.Equal(t, "bar", root.Workspace.Host)
	assert.Equal(t, "profile", root.Workspace.Profile)
}

func TestRootMergeMap(t *testing.T) {
	root := &Root{
		Path: "path",
		Targets: map[string]*Target{
			"development": {
				Workspace: &Workspace{
					Host:    "foo",
					Profile: "profile",
				},
			},
		},
	}
	other := &Root{
		Path: "path",
		Targets: map[string]*Target{
			"development": {
				Workspace: &Workspace{
					Host: "bar",
				},
			},
		},
	}
	assert.NoError(t, root.Merge(other))
	assert.Equal(t, &Workspace{Host: "bar", Profile: "profile"}, root.Targets["development"].Workspace)
}

func TestDuplicateIdOnLoadReturnsError(t *testing.T) {
	_, err := Load("./testdata/duplicate_resource_names_in_root/databricks.yml")
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_names_in_root/databricks.yml, pipeline at ./testdata/duplicate_resource_names_in_root/databricks.yml)")
}

func TestDuplicateIdOnMergeReturnsError(t *testing.T) {
	root, err := Load("./testdata/duplicate_resource_name_in_subconfiguration/databricks.yml")
	require.NoError(t, err)

	other, err := Load("./testdata/duplicate_resource_name_in_subconfiguration/resources.yml")
	require.NoError(t, err)

	err = root.Merge(other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_name_in_subconfiguration/databricks.yml, pipeline at ./testdata/duplicate_resource_name_in_subconfiguration/resources.yml)")
}

func TestInitializeVariables(t *testing.T) {
	fooDefault := "abc"
	root := &Root{
		Variables: map[string]*variable.Variable{
			"foo": {
				Default:     &fooDefault,
				Description: "an optional variable since default is defined",
			},
			"bar": {
				Description: "a required variable",
			},
		},
	}

	err := root.InitializeVariables([]string{"foo=123", "bar=456"})
	assert.NoError(t, err)
	assert.Equal(t, "123", *(root.Variables["foo"].Value))
	assert.Equal(t, "456", *(root.Variables["bar"].Value))
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
	assert.Equal(t, "123=567", *(root.Variables["foo"].Value))
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
	}
	env := &Target{Mode: Development}
	require.NoError(t, root.MergeTargetOverrides(env))
	assert.Equal(t, Development, root.Bundle.Mode)
}
