package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateYAMLFiles_SimpleFieldChange(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create a simple databricks.yml with a job
	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      timeout_seconds: 3600
      tasks:
        - task_key: "main_task"
          notebook_task:
            notebook_path: "/path/to/notebook"
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes map
	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    3600,
				Remote: 7200,
			},
		},
	}

	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Equal(t, yamlPath, fileChanges[0].Path)
	assert.Contains(t, fileChanges[0].OriginalContent, "timeout_seconds: 3600")
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 7200")
	assert.NotContains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")
}

func TestGenerateYAMLFiles_NestedFieldChange(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create a simple databricks.yml with a job
	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      tasks:
        - task_key: "main_task"
          notebook_task:
            notebook_path: "/path/to/notebook"
          timeout_seconds: 1800
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes map for nested field
	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"tasks[0].timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	// Generate YAML files
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	// Verify modified content contains the new value
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")

	// Parse YAML to verify structure
	var result map[string]any
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	// Navigate to verify the change
	resources := result["resources"].(map[string]any)
	jobs := resources["jobs"].(map[string]any)
	testJob := jobs["test_job"].(map[string]any)
	tasks := testJob["tasks"].([]any)
	task0 := tasks[0].(map[string]any)

	assert.Equal(t, 3600, task0["timeout_seconds"])
}

func TestGenerateYAMLFiles_ArrayKeyValueAccess(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create a simple databricks.yml with a job with multiple tasks
	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      tasks:
        - task_key: "setup_task"
          notebook_task:
            notebook_path: "/setup"
          timeout_seconds: 600
        - task_key: "main_task"
          notebook_task:
            notebook_path: "/main"
          timeout_seconds: 1800
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes map using key-value syntax
	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"tasks[task_key='main_task'].timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	// Generate YAML files
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	// Parse YAML to verify the correct task was updated
	var result map[string]any
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	resources := result["resources"].(map[string]any)
	jobs := resources["jobs"].(map[string]any)
	testJob := jobs["test_job"].(map[string]any)
	tasks := testJob["tasks"].([]any)

	// Verify setup_task (index 0) is unchanged
	task0 := tasks[0].(map[string]any)
	assert.Equal(t, "setup_task", task0["task_key"])
	assert.Equal(t, 600, task0["timeout_seconds"])

	// Verify main_task (index 1) is updated
	task1 := tasks[1].(map[string]any)
	assert.Equal(t, "main_task", task1["task_key"])
	assert.Equal(t, 3600, task1["timeout_seconds"])
}

func TestGenerateYAMLFiles_MultipleResourcesSameFile(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create databricks.yml with multiple jobs
	yamlContent := `resources:
  jobs:
    job1:
      name: "Job 1"
      timeout_seconds: 3600
    job2:
      name: "Job 2"
      timeout_seconds: 1800
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes for both jobs
	changes := map[string]deployplan.Changes{
		"resources.jobs.job1": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    3600,
				Remote: 7200,
			},
		},
		"resources.jobs.job2": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	// Generate YAML files
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)

	// Should only have one FileChange since both resources are in the same file
	require.Len(t, fileChanges, 1)
	assert.Equal(t, yamlPath, fileChanges[0].Path)

	// Verify both changes are applied
	assert.Contains(t, fileChanges[0].ModifiedContent, "job1")
	assert.Contains(t, fileChanges[0].ModifiedContent, "job2")

	// Parse and verify both jobs are updated
	var result map[string]any
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	resources := result["resources"].(map[string]any)
	jobs := resources["jobs"].(map[string]any)

	job1 := jobs["job1"].(map[string]any)
	assert.Equal(t, 7200, job1["timeout_seconds"])

	job2 := jobs["job2"].(map[string]any)
	assert.Equal(t, 3600, job2["timeout_seconds"])
}

func TestGenerateYAMLFiles_ResourceNotFound(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create a simple databricks.yml
	yamlContent := `resources:
  jobs:
    existing_job:
      name: "Existing Job"
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes for a non-existent resource
	changes := map[string]deployplan.Changes{
		"resources.jobs.nonexistent_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: 3600,
			},
		},
	}

	// Generate YAML files - should not error, just skip the missing resource
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)

	// Should return empty list since the resource was not found
	assert.Len(t, fileChanges, 0)
}

func TestGenerateYAMLFiles_InvalidFieldPath(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create a simple databricks.yml
	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      timeout_seconds: 3600
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Load the bundle (pass directory, not file)
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Initialize the bundle config
	mutator.DefaultMutators(ctx, b)

	// Create changes with invalid field path syntax
	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"invalid[[[path": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: 7200,
			},
		},
	}

	// Generate YAML files - should handle gracefully
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)

	// Should still return a FileChange, but the invalid field should be skipped
	// The timeout_seconds value should remain unchanged
	if len(fileChanges) > 0 {
		assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")

		// Parse and verify structure is maintained
		var result map[string]any
		err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
		require.NoError(t, err)

		resources := result["resources"].(map[string]any)
		jobs := resources["jobs"].(map[string]any)
		testJob := jobs["test_job"].(map[string]any)
		assert.Equal(t, 3600, testJob["timeout_seconds"])
	}
}

func TestGenerateYAMLFiles_Include(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	// Create a temporary directory for the bundle
	tmpDir := t.TempDir()

	// Create main databricks.yml with bundle config and includes
	mainYAML := `bundle:
  name: test-bundle

include:
  - "targets/*.yml"
`

	mainPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(mainPath, []byte(mainYAML), 0o644)
	require.NoError(t, err)

	// Create targets subdirectory
	targetsDir := filepath.Join(tmpDir, "targets")
	err = os.MkdirAll(targetsDir, 0o755)
	require.NoError(t, err)

	// Create included file with dev_job resource
	devYAML := `resources:
  jobs:
    dev_job:
      name: "Dev Job"
      timeout_seconds: 1800
`

	devPath := filepath.Join(targetsDir, "dev.yml")
	err = os.WriteFile(devPath, []byte(devYAML), 0o644)
	require.NoError(t, err)

	// Load the bundle
	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	// Process includes and other default mutators
	mutator.DefaultMutators(ctx, b)

	// Create changes for the dev_job (which was defined in included file)
	changes := map[string]deployplan.Changes{
		"resources.jobs.dev_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	// Generate YAML files
	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	// Verify changes are written to targets/dev.yml (where resource was defined)
	assert.Equal(t, devPath, fileChanges[0].Path)
	assert.Contains(t, fileChanges[0].OriginalContent, "timeout_seconds: 1800")
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")
	assert.NotContains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 1800")
}

func TestGenerateYAMLFiles_TargetOverride(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	mainYAML := `bundle:
  name: test-bundle

targets:
  dev:
    resources:
      jobs:
        dev_job:
          name: "Dev Job"
          timeout_seconds: 1800
`

	mainPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(mainPath, []byte(mainYAML), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	diags := bundle.Apply(ctx, b, mutator.SelectTarget("dev"))
	require.NoError(t, diags.Error())

	changes := map[string]deployplan.Changes{
		"resources.jobs.dev_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Equal(t, mainPath, fileChanges[0].Path)
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")
}

func TestGenerateYAMLFiles_WithStructValues(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      timeout_seconds: 3600
      email_notifications:
        on_success:
          - old@example.com
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	type EmailNotifications struct {
		OnSuccess []string `json:"on_success,omitempty" yaml:"on_success,omitempty"`
		OnFailure []string `json:"on_failure,omitempty" yaml:"on_failure,omitempty"`
	}

	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"email_notifications": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: &EmailNotifications{
					OnSuccess: []string{"success@example.com"},
					OnFailure: []string{"failure@example.com"},
				},
			},
		},
	}

	fileChanges, err := GenerateYAMLFiles(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Equal(t, yamlPath, fileChanges[0].Path)
	assert.Contains(t, fileChanges[0].OriginalContent, "on_success:")
	assert.Contains(t, fileChanges[0].OriginalContent, "old@example.com")
	assert.Contains(t, fileChanges[0].ModifiedContent, "success@example.com")
	assert.Contains(t, fileChanges[0].ModifiedContent, "failure@example.com")

	type JobsConfig struct {
		Name               string              `yaml:"name"`
		TimeoutSeconds     int                 `yaml:"timeout_seconds"`
		EmailNotifications *EmailNotifications `yaml:"email_notifications,omitempty"`
	}

	type ResourcesConfig struct {
		Jobs map[string]JobsConfig `yaml:"jobs"`
	}

	type RootConfig struct {
		Resources ResourcesConfig `yaml:"resources"`
	}

	var result RootConfig
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	testJob := result.Resources.Jobs["test_job"]
	assert.Equal(t, "Test Job", testJob.Name)
	assert.Equal(t, 3600, testJob.TimeoutSeconds)
	require.NotNil(t, testJob.EmailNotifications)
	assert.Equal(t, []string{"success@example.com"}, testJob.EmailNotifications.OnSuccess)
	assert.Equal(t, []string{"failure@example.com"}, testJob.EmailNotifications.OnFailure)
}

func TestResourceKeyToDynPath(t *testing.T) {
	tests := []struct {
		name        string
		resourceKey string
		wantErr     bool
		wantLen     int
	}{
		{
			name:        "simple resource key",
			resourceKey: "resources.jobs.my_job",
			wantErr:     false,
			wantLen:     3,
		},
		{
			name:        "empty resource key",
			resourceKey: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := resourceKeyToDynPath(tt.resourceKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, path, tt.wantLen)
			}
		})
	}
}

func TestParseResourceKey(t *testing.T) {
	tests := []struct {
		name        string
		resourceKey string
		wantType    string
		wantName    string
		wantErr     bool
	}{
		{
			name:        "valid job resource",
			resourceKey: "resources.jobs.my_job",
			wantType:    "jobs",
			wantName:    "my_job",
			wantErr:     false,
		},
		{
			name:        "valid pipeline resource",
			resourceKey: "resources.pipelines.my_pipeline",
			wantType:    "pipelines",
			wantName:    "my_pipeline",
			wantErr:     false,
		},
		{
			name:        "invalid format - too few parts",
			resourceKey: "resources.jobs",
			wantErr:     true,
		},
		{
			name:        "invalid format - wrong prefix",
			resourceKey: "targets.jobs.my_job",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceType, resourceName, err := parseResourceKey(tt.resourceKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantType, resourceType)
				assert.Equal(t, tt.wantName, resourceName)
			}
		})
	}
}

func TestApplyChangesWithEnumTypes(t *testing.T) {
	ctx := context.Background()

	resource := dyn.V(map[string]dyn.Value{
		"edit_mode": dyn.V("EDITABLE"),
		"name":      dyn.V("test_job"),
	})

	changes := deployplan.Changes{
		"edit_mode": &deployplan.ChangeDesc{
			Remote: jobs.JobEditModeUiLocked,
		},
	}

	result, err := applyChanges(ctx, resource, changes)
	require.NoError(t, err)

	editMode, err := dyn.GetByPath(result, dyn.Path{dyn.Key("edit_mode")})
	require.NoError(t, err)
	assert.Equal(t, dyn.KindString, editMode.Kind())
	assert.Equal(t, "UI_LOCKED", editMode.MustString())
}

func TestApplyChangesWithPrimitiveTypes(t *testing.T) {
	ctx := context.Background()

	resource := dyn.V(map[string]dyn.Value{
		"name":        dyn.V("old_name"),
		"timeout":     dyn.V(100),
		"enabled":     dyn.V(false),
		"max_retries": dyn.V(1.5),
	})

	changes := deployplan.Changes{
		"name": &deployplan.ChangeDesc{
			Remote: "new_name",
		},
		"timeout": &deployplan.ChangeDesc{
			Remote: int64(200),
		},
		"enabled": &deployplan.ChangeDesc{
			Remote: true,
		},
		"max_retries": &deployplan.ChangeDesc{
			Remote: 2.5,
		},
	}

	result, err := applyChanges(ctx, resource, changes)
	require.NoError(t, err)

	name, err := dyn.GetByPath(result, dyn.Path{dyn.Key("name")})
	require.NoError(t, err)
	assert.Equal(t, "new_name", name.MustString())

	timeout, err := dyn.GetByPath(result, dyn.Path{dyn.Key("timeout")})
	require.NoError(t, err)
	assert.Equal(t, int64(200), timeout.MustInt())

	enabled, err := dyn.GetByPath(result, dyn.Path{dyn.Key("enabled")})
	require.NoError(t, err)
	assert.True(t, enabled.MustBool())

	maxRetries, err := dyn.GetByPath(result, dyn.Path{dyn.Key("max_retries")})
	require.NoError(t, err)
	assert.InDelta(t, 2.5, maxRetries.MustFloat(), 0.001)
}

func TestApplyChangesWithNilValues(t *testing.T) {
	ctx := context.Background()

	resource := dyn.V(map[string]dyn.Value{
		"name":        dyn.V("test_job"),
		"description": dyn.V("some description"),
	})

	changes := deployplan.Changes{
		"description": &deployplan.ChangeDesc{
			Remote: nil,
		},
	}

	result, err := applyChanges(ctx, resource, changes)
	require.NoError(t, err)

	description, err := dyn.GetByPath(result, dyn.Path{dyn.Key("description")})
	require.NoError(t, err)
	assert.Equal(t, dyn.KindNil, description.Kind())
}

func TestApplyChangesWithStructValues(t *testing.T) {
	ctx := context.Background()

	resource := dyn.V(map[string]dyn.Value{
		"name": dyn.V("test_job"),
		"settings": dyn.V(map[string]dyn.Value{
			"timeout": dyn.V(100),
		}),
	})

	type Settings struct {
		Timeout    int64  `json:"timeout"`
		MaxRetries *int64 `json:"max_retries,omitempty"`
	}

	maxRetries := int64(3)
	changes := deployplan.Changes{
		"settings": &deployplan.ChangeDesc{
			Remote: &Settings{
				Timeout:    200,
				MaxRetries: &maxRetries,
			},
		},
	}

	result, err := applyChanges(ctx, resource, changes)
	require.NoError(t, err)

	settings, err := dyn.GetByPath(result, dyn.Path{dyn.Key("settings")})
	require.NoError(t, err)
	assert.Equal(t, dyn.KindMap, settings.Kind())

	timeout, err := dyn.GetByPath(settings, dyn.Path{dyn.Key("timeout")})
	require.NoError(t, err)
	assert.Equal(t, int64(200), timeout.MustInt())

	retriesVal, err := dyn.GetByPath(settings, dyn.Path{dyn.Key("max_retries")})
	require.NoError(t, err)
	assert.Equal(t, int64(3), retriesVal.MustInt())
}
