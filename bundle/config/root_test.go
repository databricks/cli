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

func TestLoadForEachTaskInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs string
		want   string
	}{
		{name: "YAML array", inputs: "[input_1, input_2]", want: `["input_1","input_2"]`},
		{name: "YAML object array", inputs: "[{name: input_1, enabled: true}]", want: `[{"enabled":true,"name":"input_1"}]`},
		{name: "empty YAML array", inputs: "[]", want: `[]`},
		{name: "YAML dates", inputs: "[2024-01-01, 2024-01-02]", want: `["2024-01-01","2024-01-02"]`},
		{name: "nested YAML timestamp", inputs: "[{when: 2024-01-01T01:02:03.123+02:00}]", want: `[{"when":"2024-01-01T01:02:03.123+02:00"}]`},
		{name: "quoted YAML elements", inputs: `["a\"b", "line\nbreak"]`, want: `["a\"b","line\nbreak"]`},
		{name: "escaped reference in YAML array", inputs: `["$${var.item}"]`, want: `["$${var.item}"]`},
		{name: "JSON string", inputs: `'[ "input_1", "input_2" ]'`, want: `[ "input_1", "input_2" ]`},
		{name: "variable reference string", inputs: `${var.items}`, want: `${var.items}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := "bundle:\n  name: foreach\nresources:\n  jobs:\n    fan:\n      tasks:\n        - task_key: fan\n          for_each_task:\n            inputs: " + tc.inputs + "\n            task:\n              task_key: inner\n"
			root, diags := LoadFromBytes("databricks.yml", []byte(config))
			require.NoError(t, diags.Error())
			inputs := root.Resources.Jobs["fan"].Tasks[0].ForEachTask.Inputs
			require.Equal(t, tc.want, inputs)

			body, err := json.Marshal(root.Resources.Jobs["fan"].Tasks[0].ForEachTask)
			require.NoError(t, err)
			var apiTask struct {
				Inputs string `json:"inputs"`
			}
			require.NoError(t, json.Unmarshal(body, &apiTask))
			assert.Equal(t, tc.want, apiTask.Inputs)
		})
	}
}

func TestLoadForEachTaskInputsRejectsArrayReferences(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs string
		prefix string
		target bool
	}{
		{name: "root element", inputs: `["${var.item}_x", y]`, prefix: "resources.jobs.fan"},
		{name: "nested object value", inputs: `[{name: "${var.item}"}]`, prefix: "resources.jobs.fan"},
		{name: "target override", inputs: `["${var.item}"]`, prefix: "targets.dev.resources.jobs.fan", target: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var config string
			if tc.target {
				config = "bundle:\n  name: foreach\nvariables:\n  item:\n    default: input_1\ntargets:\n  dev:\n    resources:\n      jobs:\n        fan:\n          tasks:\n            - task_key: fan\n              for_each_task:\n                inputs: " + tc.inputs + "\n                task:\n                  task_key: inner\n"
			} else {
				config = "bundle:\n  name: foreach\nvariables:\n  item:\n    default: input_1\nresources:\n  jobs:\n    fan:\n      tasks:\n        - task_key: fan\n          for_each_task:\n            inputs: " + tc.inputs + "\n            task:\n              task_key: inner\n"
			}
			root, diags := LoadFromBytes("databricks.yml", []byte(config))
			require.Nil(t, root)
			require.ErrorContains(t, diags.Error(), "variable reference ${var.item} in a YAML array")
			assert.ErrorContains(t, diags.Error(), tc.prefix+".tasks[0].for_each_task.inputs")
			assert.ErrorContains(t, diags.Error(), "databricks.yml:")
		})
	}
}

func TestLoadForEachTaskInputsInTarget(t *testing.T) {
	config := `bundle:
  name: foreach
resources:
  jobs:
    fan:
      tasks:
        - task_key: fan
          for_each_task:
            inputs: [root_1]
            task:
              task_key: inner
targets:
  dev:
    resources:
      jobs:
        fan:
          tasks:
            - task_key: fan
              for_each_task:
                inputs: [input_1, input_2]
                task:
                  task_key: inner
`
	root, diags := LoadFromBytes("databricks.yml", []byte(config))
	require.NoError(t, diags.Error())
	require.Equal(t, `["root_1"]`, root.Resources.Jobs["fan"].Tasks[0].ForEachTask.Inputs)
	require.Equal(t, `["input_1","input_2"]`, root.Targets["dev"].Resources.Jobs["fan"].Tasks[0].ForEachTask.Inputs)
	require.NoError(t, root.MergeTargetOverrides("dev"))
	require.Len(t, root.Resources.Jobs["fan"].Tasks, 2)
	assert.Equal(t, `["root_1"]`, root.Resources.Jobs["fan"].Tasks[0].ForEachTask.Inputs)
	assert.Equal(t, `["input_1","input_2"]`, root.Resources.Jobs["fan"].Tasks[1].ForEachTask.Inputs)
}

func TestLoadForEachTaskInputsPreservesVariableContainers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
	}{
		{
			name: "for-each task reference",
			config: `bundle:
  name: foreach
variables:
  each:
    type: complex
    default:
      inputs: '["item"]'
      task:
        task_key: inner
resources:
  jobs:
    fan:
      tasks:
        - task_key: fan
          for_each_task: ${var.each}
`,
		},
		{
			name: "task reference",
			config: `bundle:
  name: foreach
variables:
  task:
    type: complex
    default:
      task_key: fan
      for_each_task:
        inputs: '["item"]'
        task:
          task_key: inner
resources:
  jobs:
    fan:
      tasks:
        - ${var.task}
`,
		},
		{
			name: "task list reference",
			config: `bundle:
  name: foreach
variables:
  tasks:
    type: complex
    default:
      - task_key: fan
        for_each_task:
          inputs: '["item"]'
          task:
            task_key: inner
resources:
  jobs:
    fan:
      tasks: ${var.tasks}
`,
		},
		{
			name: "job reference",
			config: `bundle:
  name: foreach
variables:
  job:
    type: complex
    default:
      tasks:
        - task_key: fan
          for_each_task:
            inputs: '["item"]'
            task:
              task_key: inner
resources:
  jobs:
    fan: ${var.job}
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := LoadFromBytes("databricks.yml", []byte(tc.config))
			require.NoError(t, diags.Error())
			require.NotNil(t, root)
		})
	}
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
	assert.Equal(t, "123", root.Variables["foo"].Value)
	assert.Equal(t, "456", root.Variables["bar"].Value)
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
	assert.Equal(t, "123=567", root.Variables["foo"].Value)
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
