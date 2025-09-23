package mutator

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePaths(t *testing.T) {
	tmpDir := t.TempDir()
	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Tasks: []jobs.Task{
							{
								NotebookTask: &jobs.NotebookTask{
									NotebookPath: "../src/notebook.py",
								},
							},
						},
					}},
				},
			},
		},
		BundleRootPath: tmpDir,
	}

	// update config as if 'notebook_path' property is defined in resources/job_1.yml
	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	path := dyn.MustPathFromString("resources.jobs.job1.tasks[0].notebook_task.notebook_path")
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, path, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return dyn.NewValue(value.MustString(), []dyn.Location{location}), nil
		})
	})
	require.NoError(t, err)

	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	newValue, err := dyn.GetByPath(b.Config.Value(), path)
	require.NoError(t, err)
	require.Equal(t, "src/notebook.py", newValue.MustString())
}

func TestNormalizePath_absolutePath(t *testing.T) {
	value, err := normalizePath("/notebook.py", dyn.Location{}, "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "/notebook.py", value)
}

func TestNormalizePath_url(t *testing.T) {
	value, err := normalizePath("s3:///path/to/notebook.py", dyn.Location{}, "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "s3:///path/to/notebook.py", value)
}

func TestNormalizePath_requirementsFile(t *testing.T) {
	tmpDir := t.TempDir()
	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	value, err := normalizePath("-r ../requirements.txt", location, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)

	value, err = normalizePath("-r      ../requirements.txt", location, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)
}

func TestLocationDirectory(t *testing.T) {
	loc := dyn.Location{File: "file", Line: 1, Column: 2}
	dir, err := locationDirectory(loc)
	assert.NoError(t, err)
	assert.Equal(t, ".", dir)
}

func TestLocationDirectoryNoFile(t *testing.T) {
	loc := dyn.Location{}
	_, err := locationDirectory(loc)
	assert.Error(t, err)
}

func TestNormalizePath_unsupportedPipOptions(t *testing.T) {
	loc := dyn.Location{File: "/bundle/root/test.yml", Line: 1, Column: 1}

	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "unsupported single dash option",
			input:       "-x some-package",
			expectError: true,
			errorMsg:    "unsupported pip option '-x' in dependency '-x some-package'. Supported options are: -r, -e, -f, -i, --requirement, --editable, --find-links, --index-url, --extra-index-url, --trusted-host",
		},
		{
			name:        "unsupported double dash option",
			input:       "--unknown-flag /path/to/something",
			expectError: true,
			errorMsg:    "unsupported pip option '--unknown-flag' in dependency '--unknown-flag /path/to/something'. Supported options are: -r, -e, -f, -i, --requirement, --editable, --find-links, --index-url, --extra-index-url, --trusted-host",
		},
		{
			name:        "future pip option",
			input:       "--future-2026-option /path/to/something",
			expectError: true,
			errorMsg:    "unsupported pip option '--future-2026-option' in dependency '--future-2026-option /path/to/something'. Supported options are: -r, -e, -f, -i, --requirement, --editable, --find-links, --index-url, --extra-index-url, --trusted-host",
		},
		{
			name:        "pip option without space",
			input:       "-e",
			expectError: false,
		},
		{
			name:        "unknown option without space should pass through",
			input:       "--unknownflag",
			expectError: false,
		},
		{
			name:        "supported option should work",
			input:       "-e ../myproject",
			expectError: false,
		},
		{
			name:        "extra-index-url should be recognized but not normalized",
			input:       "--extra-index-url https://pypi.org/simple",
			expectError: false,
		},
		{
			name:        "trusted-host should be recognized but not normalized",
			input:       "--trusted-host pypi.org",
			expectError: false,
		},
		{
			name:        "absolute path in pip option should work",
			input:       "-e /Workspace/Users/lennart/abspath.whl",
			expectError: false,
		},
		{
			name:        "absolute path in long pip option should work",
			input:       "--editable /Workspace/Users/lennart/abspath.whl",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizePath(tt.input, loc, "/bundle/root")

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}
