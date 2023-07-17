package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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
	root := &Root{}
	err := root.Load("../tests/basic/databricks.yaml")
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
		Environments: map[string]*Environment{
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
		Environments: map[string]*Environment{
			"development": {
				Workspace: &Workspace{
					Host: "bar",
				},
			},
		},
	}
	assert.NoError(t, root.Merge(other))
	assert.Equal(t, &Workspace{Host: "bar", Profile: "profile"}, root.Environments["development"].Workspace)
}

func TestDuplicateIdOnLoadReturnsError(t *testing.T) {
	root := &Root{}
	err := root.Load("./testdata/duplicate_resource_names_in_root/databricks.yaml")
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_names_in_root/databricks.yaml, pipeline at ./testdata/duplicate_resource_names_in_root/databricks.yaml)")
}

func TestDuplicateIdOnMergeReturnsError(t *testing.T) {
	root := &Root{}
	err := root.Load("./testdata/duplicate_resource_name_in_subconfiguration/databricks.yaml")
	require.NoError(t, err)

	other := &Root{}
	err = other.Load("./testdata/duplicate_resource_name_in_subconfiguration/resources.yml")
	require.NoError(t, err)

	err = root.Merge(other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at ./testdata/duplicate_resource_name_in_subconfiguration/databricks.yaml, pipeline at ./testdata/duplicate_resource_name_in_subconfiguration/resources.yml)")
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

func TestRootMergeEnvironmentWithMode(t *testing.T) {
	root := &Root{
		Bundle: Bundle{},
	}
	env := &Environment{Mode: Development}
	require.NoError(t, root.MergeEnvironment(env))
	assert.Equal(t, Development, root.Bundle.Mode)
}

func TestConfigFileNames_FindInPath(t *testing.T) {
	testCases := []struct {
		name     string
		files    []string
		expected string
		err      string
	}{
		{
			name:     "file found",
			files:    []string{"databricks.yaml"},
			expected: "BASE/databricks.yaml",
			err:      "",
		},
		{
			name:     "file found",
			files:    []string{"bundle.yml"},
			expected: "BASE/bundle.yml",
			err:      "",
		},
		{
			name:     "multiple files found",
			files:    []string{"databricks.yaml", "bundle.yaml"},
			expected: "",
			err:      "multiple bundle root configuration files found",
		},
		{
			name:     "file not found",
			files:    []string{},
			expected: "",
			err:      "no such file or directory",
		},
	}

	if runtime.GOOS == "windows" {
		testCases[2].err = "The system cannot find the file specified."
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			for _, file := range tc.files {
				f1, _ := os.Create(filepath.Join(projectDir, file))
				f1.Close()
			}

			result, err := FileNames.FindInPath(projectDir)

			expected := strings.Replace(tc.expected, "BASE/", projectDir+string(os.PathSeparator), 1)
			assert.Equal(t, expected, result)

			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
