package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/dyn"
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

func TestDuplicateIdOnLoadReturnsErrorForJobAndPipeline(t *testing.T) {
	_, diags := Load("./testdata/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml")
	assert.ErrorContains(t, diags.Error(), "multiple resources named foo (job at ./testdata/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml:10:7, pipeline at ./testdata/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml:13:7)")
}

func TestDuplicateIdOnLoadReturnsErrorForJobsAndExperiments(t *testing.T) {
	_, diags := Load("./testdata/duplicate_resource_names_in_root_job_and_experiment/databricks.yml")
	assert.ErrorContains(t, diags.Error(), "multiple resources named foo (job at ./testdata/duplicate_resource_names_in_root_jobs_and_experiments/databricks.yml:10:7, experiment at ./testdata/duplicate_resource_names_in_root_jobs_and_experiments/databricks.yml:18:7)")
}

func TestDuplicateIdOnMergeReturnsErrorForJobAndPipeline(t *testing.T) {
	root, diags := Load("./testdata/duplicate_resource_name_in_subconfiguration/databricks.yml")
	require.NoError(t, diags.Error())

	other, diags := Load("./testdata/duplicate_resource_name_in_subconfiguration/resources.yml")
	require.NoError(t, diags.Error())

	err := root.Merge(other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_name_in_subconfiguration/databricks.yml:10:7, pipeline at ./testdata/duplicate_resource_name_in_subconfiguration/resources.yml:4:7)")
}

func TestDuplicateIdOnMergeReturnsErrorForJobAndJob(t *testing.T) {
	root, diags := Load("./testdata/duplicate_resource_name_in_subconfiguration_job_and_job/databricks.yml")
	require.NoError(t, diags.Error())

	other, diags := Load("./testdata/duplicate_resource_name_in_subconfiguration_job_and_job/resources.yml")
	require.NoError(t, diags.Error())

	err := root.Merge(other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_name_in_subconfiguration_job_and_job/databricks.yml:10:7, job at ./testdata/duplicate_resource_name_in_subconfiguration_job_and_job/resources.yml:4:7)")
}

func TestGatherResourceIdentifiers(t *testing.T) {
	root, diags := Load("./testdata/gather_resource_identifiers/databricks.yml")
	require.NoError(t, diags.Error())

	actual, err := root.gatherResourceIdentifiers()
	assert.NoError(t, err)

	expected := map[string]dyn.Path{
		"foo": dyn.MustPathFromString("jobs.foo"),
		"bar": dyn.MustPathFromString("jobs.bar"),
		"zab": dyn.MustPathFromString("pipelines.zab"),
		"baz": dyn.MustPathFromString("pipelines.baz"),
		"zaz": dyn.MustPathFromString("experiments.zaz"),
		"zuz": dyn.MustPathFromString("experiments.zuz"),
	}
	assert.Equal(t, expected, actual)
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
