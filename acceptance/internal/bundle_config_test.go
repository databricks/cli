package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMergeBundleConfig(t *testing.T) {
	tests := []struct {
		name         string
		initialYaml  string
		bundleConfig map[string]any
		expected     map[string]any
	}{
		{
			name:         "new_file",
			initialYaml:  "",
			bundleConfig: map[string]any{},
			expected:     map[string]any{},
		},
		{
			name:         "empty_config",
			initialYaml:  "{}",
			bundleConfig: map[string]any{},
			expected:     map[string]any{},
		},
		{
			name:        "simple_top_level",
			initialYaml: "{}",
			bundleConfig: map[string]any{
				"name": "test-bundle",
			},
			expected: map[string]any{
				"name": "test-bundle",
			},
		},
		{
			name:        "simple_top_level_new_file",
			initialYaml: "",
			bundleConfig: map[string]any{
				"name": "test-bundle",
			},
			expected: map[string]any{
				"name": "test-bundle",
			},
		},
		{
			name:        "nested_config",
			initialYaml: "{}",
			bundleConfig: map[string]any{
				"bundle": map[string]any{
					"name": "test-bundle",
				},
			},
			expected: map[string]any{
				"bundle": map[string]any{
					"name": "test-bundle",
				},
			},
		},
		{
			name:        "nested_config_new_file",
			initialYaml: "",
			bundleConfig: map[string]any{
				"bundle": map[string]any{
					"name": "test-bundle",
				},
			},
			expected: map[string]any{
				"bundle": map[string]any{
					"name": "test-bundle",
				},
			},
		},
		{
			name: "merge_with_existing_config",
			initialYaml: `
bundle:
  name: original-name
  target: dev
`,
			bundleConfig: map[string]any{
				"bundle": map[string]any{
					"name": "default-name",
				},
			},
			expected: map[string]any{
				"bundle": map[string]any{
					"name":   "original-name",
					"target": "dev",
				},
			},
		},
		{
			name:        "merge_with_existing_config_2",
			initialYaml: `resources: {}`,
			bundleConfig: map[string]any{
				"bundle": map[string]any{
					"name": "new-name",
				},
			},
			expected: map[string]any{
				"bundle": map[string]any{
					"name": "new-name",
				},
				"resources": map[string]any{},
			},
		},

		{
			name: "multiple_nested_levels",
			initialYaml: `
resources:
  jobs:
    myjob:
      hello: world
  pipelines:
    mypipeline:
      name: 123
`,
			bundleConfig: map[string]any{
				"resources": map[string]any{
					"jobs": map[string]any{
						"myjob": map[string]any{
							"name": "My Job",
						},
					},
				},
			},
			expected: map[string]any{
				"resources": map[string]any{
					"jobs": map[string]any{
						"myjob": map[string]any{
							"name":  "My Job",
							"hello": "world",
						},
					},
					"pipelines": map[string]any{
						"mypipeline": map[string]any{
							"name": 123,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := MergeBundleConfig(tt.initialYaml, tt.bundleConfig)
			assert.NoError(t, err)

			var result map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(out), &result))

			assert.Equal(t, tt.expected, result)
		})
	}
}
