package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestApplyChangesToYAML_SimpleFieldChange(t *testing.T) {
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

func TestApplyChangesToYAML_NestedFieldChange(t *testing.T) {
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

func TestApplyChangesToYAML_ArrayKeyValueAccess(t *testing.T) {
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

func TestApplyChangesToYAML_MultipleResourcesSameFile(t *testing.T) {
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

func TestApplyChangesToYAML_ResourceNotFound(t *testing.T) {
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

func TestApplyChangesToYAML_InvalidFieldPath(t *testing.T) {
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

func TestApplyChangesToYAML_Include(t *testing.T) {
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

func TestApplyChangesToYAML_WithStructValues(t *testing.T) {
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

func TestApplyChangesToYAML_PreserveComments(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	yamlContent := `# test_comment0
resources:
	# test_comment1
  jobs:
    test_job:
		# test_comment2
      name: "Test Job"
		# test_comment3
      timeout_seconds: 3600
		# test_comment4
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
				Remote: 7200,
			},
			"name": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: "New Test Job",
			},
			"tags": &deployplan.ChangeDesc{
				Action: deployplan.Update,
				Remote: map[string]string{
					"test": "value",
				},
			},
		},
	}

	fileChanges, err := ApplyChangesToYAML(ctx, b, changes)
	require.NoError(t, err)

	assert.Equal(t, yamlPath, fileChanges[0].Path)

	assert.Contains(t, fileChanges[0].ModifiedContent, "# test_comment0")
	assert.Contains(t, fileChanges[0].ModifiedContent, "# test_comment1")
	assert.Contains(t, fileChanges[0].ModifiedContent, "# test_comment2")
}
