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

func TestRewriteAiRuntimeCodeSourceExtractsBlock(t *testing.T) {
	yml := []byte(`
resources:
  jobs:
    train:
      tasks:
        - task_key: t1
          ai_runtime_task:
            experiment: my-exp
            code_source:
              root_path: ./src
              include_paths: [src, configs]
              git:
                commit: abc123
`)
	root, diags := LoadFromBytes("databricks.yml", yml)
	require.NoError(t, diags.Error())

	// The block is stashed keyed by (job, task_key).
	extras, ok := root.AiRuntimeExtras["train"]["t1"]
	require.True(t, ok, "expected code_source stashed under train/t1")
	require.NotNil(t, extras.CodeSource)
	assert.Equal(t, "./src", extras.CodeSource.RootPath)
	assert.Equal(t, []string{"src", "configs"}, extras.CodeSource.IncludePaths)
	require.NotNil(t, extras.CodeSource.Git)
	assert.Equal(t, "abc123", extras.CodeSource.Git.Commit)

	// The code_source key is removed from the task, and no unknown-field warning fired.
	task := root.Resources.Jobs["train"].Tasks[0]
	require.NotNil(t, task.AiRuntimeTask)
	assert.Equal(t, "my-exp", task.AiRuntimeTask.Experiment)
	assert.Empty(t, diags, "expected no diagnostics (no unknown-field warning)")
}

func TestRewriteAiRuntimeCodeSourceLeavesCodeSourcePathAlone(t *testing.T) {
	yml := []byte(`
resources:
  jobs:
    train:
      tasks:
        - task_key: t1
          ai_runtime_task:
            experiment: my-exp
            code_source_path: ./src
`)
	root, diags := LoadFromBytes("databricks.yml", yml)
	require.NoError(t, diags.Error())
	assert.Nil(t, root.AiRuntimeExtras)
	assert.Equal(t, "./src", root.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath)
}

func TestRewriteAiRuntimeCodeSourceRejectsMissingTaskKey(t *testing.T) {
	yml := []byte(`
resources:
  jobs:
    train:
      tasks:
        - ai_runtime_task:
            experiment: my-exp
            code_source:
              root_path: ./src
`)
	_, diags := LoadFromBytes("databricks.yml", yml)
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "requires the task to set a task_key")
}

func TestRewriteAiRuntimeCodeSourceRejectsTargetOverride(t *testing.T) {
	yml := []byte(`
targets:
  dev:
    resources:
      jobs:
        train:
          tasks:
            - task_key: t1
              ai_runtime_task:
                experiment: my-exp
                code_source:
                  root_path: ./src
`)
	_, diags := LoadFromBytes("databricks.yml", yml)
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "not supported inside a target override")
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
	require.NoError(t, root.initializeDynamicValue())
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
	require.NoError(t, root.initializeDynamicValue())
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
		value    dyn.Value
		expected bool
	}{
		{
			value: dyn.V(map[string]dyn.Value{
				"type":        dyn.V("string"),
				"default":     dyn.V("foo"),
				"description": dyn.V("foo var"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"type":        dyn.V("string"),
				"lookup":      dyn.V("foo"),
				"description": dyn.V("foo var"),
			}),
			expected: false,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"type":    dyn.V("string"),
				"default": dyn.V("foo"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"type":   dyn.V("string"),
				"lookup": dyn.V("foo"),
			}),
			expected: false,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"description": dyn.V("string"),
				"default":     dyn.V("foo"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"description": dyn.V("string"),
				"lookup":      dyn.V("foo"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"default": dyn.V("foo"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"lookup": dyn.V("foo"),
			}),
			expected: true,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"type": dyn.V("string"),
			}),
			expected: false,
		},
		{
			value: dyn.V(map[string]dyn.Value{
				"type":        dyn.V("string"),
				"default":     dyn.V("foo"),
				"description": dyn.V("foo var"),
				"lookup":      dyn.V("foo"),
			}),
			expected: false,
		},
	}

	for i, tc := range testCases {
		assert.Equal(t, tc.expected, isFullVariableOverrideDef(tc.value), "test case %d", i)
	}
}
