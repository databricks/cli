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

	tmpDir := t.TempDir()

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

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    3600,
				Remote: 7200,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Equal(t, yamlPath, fileChanges[0].Path)
	assert.Contains(t, fileChanges[0].OriginalContent, "timeout_seconds: 3600")
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 7200")
	assert.NotContains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")
}

func TestGenerateYAMLFiles_NestedFieldChange(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

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

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"tasks[0].timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")

	var result map[string]any
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	resources := result["resources"].(map[string]any)
	jobs := resources["jobs"].(map[string]any)
	testJob := jobs["test_job"].(map[string]any)
	tasks := testJob["tasks"].([]any)
	task0 := tasks[0].(map[string]any)

	assert.Equal(t, 3600, task0["timeout_seconds"])
}

func TestGenerateYAMLFiles_ArrayKeyValueAccess(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

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

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"tasks[task_key='main_task'].timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	var result map[string]any
	err = yaml.Unmarshal([]byte(fileChanges[0].ModifiedContent), &result)
	require.NoError(t, err)

	resources := result["resources"].(map[string]any)
	jobs := resources["jobs"].(map[string]any)
	testJob := jobs["test_job"].(map[string]any)
	tasks := testJob["tasks"].([]any)

	task0 := tasks[0].(map[string]any)
	assert.Equal(t, "setup_task", task0["task_key"])
	assert.Equal(t, 600, task0["timeout_seconds"])

	task1 := tasks[1].(map[string]any)
	assert.Equal(t, "main_task", task1["task_key"])
	assert.Equal(t, 3600, task1["timeout_seconds"])
}

func TestGenerateYAMLFiles_MultipleResourcesSameFile(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

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

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

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

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)

	require.Len(t, fileChanges, 1)
	assert.Equal(t, yamlPath, fileChanges[0].Path)

	assert.Contains(t, fileChanges[0].ModifiedContent, "job1")
	assert.Contains(t, fileChanges[0].ModifiedContent, "job2")

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

	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    existing_job:
      name: "Existing Job"
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.nonexistent_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: 3600,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)

	assert.Len(t, fileChanges, 0)
}

func TestGenerateYAMLFiles_InvalidFieldPath(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
      timeout_seconds: 3600
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.test_job": {
			"invalid[[[path": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: 7200,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)

	if len(fileChanges) > 0 {
		assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 3600")

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

	tmpDir := t.TempDir()

	mainYAML := `bundle:
  name: test-bundle

include:
  - "targets/*.yml"
`

	mainPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(mainPath, []byte(mainYAML), 0o644)
	require.NoError(t, err)

	targetsDir := filepath.Join(tmpDir, "targets")
	err = os.MkdirAll(targetsDir, 0o755)
	require.NoError(t, err)

	devYAML := `resources:
  jobs:
    dev_job:
      name: "Dev Job"
      timeout_seconds: 1800
`

	devPath := filepath.Join(targetsDir, "dev.yml")
	err = os.WriteFile(devPath, []byte(devYAML), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := map[string]deployplan.Changes{
		"resources.jobs.dev_job": {
			"timeout_seconds": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Old:    1800,
				Remote: 3600,
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

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

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
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

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
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

func TestApplyChanges_CreatesIntermediateNodes(t *testing.T) {
	ctx := context.Background()

	// Resource without tags field
	resource := dyn.V(map[string]dyn.Value{
		"name": dyn.V("test_job"),
	})

	// Change that requires creating tags map
	changes := deployplan.Changes{
		"tags['test']": &deployplan.ChangeDesc{
			Remote: "val",
		},
	}

	result, err := applyChanges(ctx, resource, changes)
	require.NoError(t, err)

	// Verify tags map was created
	tags, err := dyn.GetByPath(result, dyn.Path{dyn.Key("tags")})
	require.NoError(t, err)
	assert.Equal(t, dyn.KindMap, tags.Kind())

	// Verify test key was set
	testVal, err := dyn.GetByPath(result, dyn.Path{dyn.Key("tags"), dyn.Key("test")})
	require.NoError(t, err)
	assert.Equal(t, "val", testVal.MustString())
}
