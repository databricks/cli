package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTargets(t *testing.T) {
	mappings, err := LoadTargetMappings("../../Taskfile.yml")
	require.NoError(t, err)

	tests := []struct {
		name    string
		files   []string
		targets []string
	}{
		{
			name: "experimental_ssh",
			files: []string{
				"experimental/ssh/main.go",
				"experimental/ssh/lib/server.go",
			},
			targets: []string{"test-exp-ssh"},
		},
		{
			name: "pipelines",
			files: []string{
				"cmd/pipelines/main.go",
			},
			targets: []string{"test-pipelines"},
		},
		{
			name: "acceptance_apps_triggers_aitools",
			files: []string{
				"acceptance/apps/basic/script",
			},
			targets: []string{"test-exp-aitools"},
		},
		{
			name: "non_matching",
			files: []string{
				"bundle/config.go",
				"cmd/bundle/deploy.go",
			},
			targets: []string{"test"},
		},
		{
			name: "go_mod_triggers_all",
			files: []string{
				"go.mod",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines"},
		},
		{
			name: "go_sum_triggers_all",
			files: []string{
				"go.sum",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines"},
		},
		{
			name: "go_mod_with_other_files_triggers_all",
			files: []string{
				"experimental/ssh/main.go",
				"go.mod",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines"},
		},
		{
			name: "setup_build_environment_triggers_all",
			files: []string{
				".github/actions/setup-build-environment/action.yml",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines"},
		},
		{
			name:    "empty_files",
			files:   []string{},
			targets: []string{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := GetTargets(tt.files, mappings)
			assert.Equal(t, tt.targets, targets)
		})
	}
}

func TestLoadTargetMappingsMissingFile(t *testing.T) {
	_, err := LoadTargetMappings("nonexistent.yml")
	assert.Error(t, err)
}
