package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectVisitedPaths is a helper function that collects all visited paths for testing
func collectVisitedPaths(t *testing.T, root config.Root) []dyn.Path {
	return collectVisitedPathsOf(t, root, func(resources.LocalPath) bool { return true })
}

// collectVisitedPathsOf collects the visited paths of the fields for which keep returns true.
func collectVisitedPathsOf(t *testing.T, root config.Root, keep func(resources.LocalPath) bool) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitPaths(value, func(p dyn.Path, field resources.LocalPath, v dyn.Value) (dyn.Value, error) {
			if keep(field) {
				actual = append(actual, p)
			}
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}

// The visit order decides which error is reported first, so it is pinned:
// resource types in canonical order, each in the order of its declarations,
// then artifacts.
func TestVisitPathsOrder(t *testing.T) {
	root := config.Root{
		Artifacts: config.Artifacts{
			"artifact0": &config.Artifact{Path: "foo"},
		},
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"app0": {App: apps.App{SourceCodePath: "foo"}},
			},
			Pipelines: map[string]*resources.Pipeline{
				"pipeline0": {CreatePipeline: pipelines.CreatePipeline{RootPath: "foo"}},
			},
			Jobs: map[string]*resources.Job{
				"job0": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{
					TaskKey:      "task0",
					NotebookTask: &jobs.NotebookTask{NotebookPath: "foo"},
					Libraries:    []compute.Library{{Whl: "foo.whl"}},
				}}}},
			},
		},
	}

	actual := collectVisitedPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].notebook_task.notebook_path"),
		dyn.MustPathFromString("resources.jobs.job0.tasks[0].libraries[0].whl"),
		dyn.MustPathFromString("resources.pipelines.pipeline0.root_path"),
		dyn.MustPathFromString("resources.apps.app0.source_code_path"),
		dyn.MustPathFromString("artifacts.artifact0.path"),
	}

	assert.Equal(t, expected, actual)
}

// allLocalPathsConfig sets every declared local path field once. Dependencies are
// set twice because one pattern is declared twice, once per filter.
const allLocalPathsConfig = `
artifacts:
  artifact0:
    path: ./artifact

resources:
  jobs:
    job0:
      tasks:
        - task_key: task0
          notebook_task:
            notebook_path: ./notebook.py
          spark_python_task:
            python_file: ./file.py
          dbt_task:
            project_directory: ./dbt
          sql_task:
            file:
              path: ./query.sql
          alert_task:
            workspace_path: ./alert.json
          libraries:
            - requirements: ./requirements.txt
            - whl: ./dist/task.whl
            - jar: ./dist/task.jar
          ai_runtime_task:
            deployments:
              - command_path: ./command.sh
          for_each_task:
            inputs: "[1]"
            task:
              task_key: nested0
              notebook_task:
                notebook_path: ./notebook.py
              spark_python_task:
                python_file: ./file.py
              dbt_task:
                project_directory: ./dbt
              sql_task:
                file:
                  path: ./query.sql
              alert_task:
                workspace_path: ./alert.json
              libraries:
                - requirements: ./requirements.txt
                - whl: ./dist/task.whl
                - jar: ./dist/task.jar
              ai_runtime_task:
                deployments:
                  - command_path: ./command.sh
      environment_variables:
        - environment_variables_key: default
          spec:
            files:
              - ./a.env
      environments:
        - environment_key: env0
          spec:
            project_environment: ./pyproject.toml
            dependencies:
              - ./dist/env.whl
              - -r ./requirements.txt

  job_runs:
    run0:
      lifecycle:
        triggers:
          - on_file_change: ./src

  pipelines:
    pipeline0:
      root_path: ./pipeline
      libraries:
        - notebook:
            path: ./notebook.py
        - file:
            path: ./file.py
        - glob:
            include: ./transformations/**
      environment:
        dependencies:
          - ./dist/env.whl
          - -r ./requirements.txt
      testing:
        sources:
          - glob:
              include: ./tests/**
        environment:
          dependencies:
            - ./dist/test.whl
            - -r ./test-requirements.txt

  dashboards:
    dashboard0:
      file_path: ./dashboard.lvdash.json

  genie_spaces:
    genie0:
      file_path: ./genie.json

  apps:
    app0:
      source_code_path: ./app
      thumbnail_path: ./thumbnail.png

  alerts:
    alert0:
      file_path: ./alert.dbalert.json
`

// TestVisitPathsDeclarations pins every declared local path field: which value
// it matches, how and whether it is translated, and whether it may point outside
// the sync root. A change to a declaration shows up here in review.
func TestVisitPathsDeclarations(t *testing.T) {
	const (
		translatePaths           = "TranslatePaths"
		translatePathsDashboards = "TranslatePathsDashboards"
		normalizedOnly           = "none"
	)

	// The mutator in translate_paths.go that translates the fields of each TranslateWhen.
	translatedBy := map[resources.TranslateWhen]string{
		resources.TranslateAfterLoad: translatePaths,
		resources.TranslateOnLoad:    translatePathsDashboards,
		resources.TranslateNever:     normalizedOnly,
	}

	type declaration struct {
		path                 string
		mode                 resources.TranslateMode
		filter               resources.LocalPathFilter
		allowOutsideSyncRoot bool
		translatedBy         string
	}

	const (
		notebook       = resources.TranslateModeNotebook
		file           = resources.TranslateModeFile
		directory      = resources.TranslateModeDirectory
		glob           = resources.TranslateModeGlob
		localAbsDir    = resources.TranslateModeLocalAbsoluteDirectory
		localRelative  = resources.TranslateModeLocalRelative
		localRelPrefix = resources.TranslateModeLocalRelativeWithPrefix
		pipFlag        = resources.TranslateModeEnvironmentPipFlag

		all        = resources.AllLocalPaths
		onlyLocal  = resources.OnlyLocalLibraries
		onlyPipArg = resources.OnlyPipFlagPaths
	)

	expected := []declaration{
		// jobs: task fields.
		{"resources.jobs.job0.tasks[0].notebook_task.notebook_path", notebook, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].spark_python_task.python_file", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].dbt_task.project_directory", directory, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].sql_task.file.path", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].alert_task.workspace_path", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].libraries[0].requirements", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].ai_runtime_task.deployments[0].command_path", file, all, false, translatePaths},

		// jobs: the same task fields under for_each_task.
		{"resources.jobs.job0.tasks[0].for_each_task.task.notebook_task.notebook_path", notebook, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.spark_python_task.python_file", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.dbt_task.project_directory", directory, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.sql_task.file.path", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.alert_task.workspace_path", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.libraries[0].requirements", file, all, false, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.ai_runtime_task.deployments[0].command_path", file, all, false, translatePaths},

		// jobs: environment fields.
		{"resources.jobs.job0.environment_variables[0].spec.files[0]", file, all, false, translatePaths},
		{"resources.jobs.job0.environments[0].spec.project_environment", file, all, false, translatePaths},

		// jobs: libraries.
		{"resources.jobs.job0.tasks[0].libraries[1].whl", localRelative, all, true, translatePaths},
		{"resources.jobs.job0.tasks[0].libraries[2].jar", localRelative, all, true, translatePaths},
		{"resources.jobs.job0.environments[0].spec.dependencies[0]", localRelPrefix, onlyLocal, true, translatePaths},
		{"resources.jobs.job0.environments[0].spec.dependencies[1]", pipFlag, onlyPipArg, true, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.libraries[1].whl", localRelative, all, true, translatePaths},
		{"resources.jobs.job0.tasks[0].for_each_task.task.libraries[2].jar", localRelative, all, true, translatePaths},

		// job_runs.
		{"resources.job_runs.run0.lifecycle.triggers[0].on_file_change", localRelative, all, false, normalizedOnly},

		// pipelines.
		{"resources.pipelines.pipeline0.libraries[0].notebook.path", notebook, all, false, translatePaths},
		{"resources.pipelines.pipeline0.libraries[1].file.path", file, all, false, translatePaths},
		{"resources.pipelines.pipeline0.libraries[2].glob.include", glob, all, false, translatePaths},
		{"resources.pipelines.pipeline0.root_path", directory, all, false, translatePaths},
		{"resources.pipelines.pipeline0.testing.sources[0].glob.include", glob, all, false, translatePaths},
		{"resources.pipelines.pipeline0.environment.dependencies[0]", localRelPrefix, onlyLocal, true, translatePaths},
		{"resources.pipelines.pipeline0.testing.environment.dependencies[0]", localRelPrefix, onlyLocal, true, translatePaths},
		{"resources.pipelines.pipeline0.environment.dependencies[1]", pipFlag, onlyPipArg, true, translatePaths},
		{"resources.pipelines.pipeline0.testing.environment.dependencies[1]", pipFlag, onlyPipArg, true, translatePaths},

		// dashboards and genie_spaces.
		{"resources.dashboards.dashboard0.file_path", localRelative, all, false, translatePathsDashboards},
		{"resources.genie_spaces.genie0.file_path", localRelative, all, false, translatePathsDashboards},

		// apps.
		{"resources.apps.app0.source_code_path", directory, all, false, translatePaths},
		{"resources.apps.app0.thumbnail_path", localRelative, all, false, translatePaths},

		// alerts.
		{"resources.alerts.alert0.file_path", localRelative, all, false, normalizedOnly},

		// artifacts.
		{"artifacts.artifact0.path", localAbsDir, all, true, translatePaths},
	}

	// Each declaration is visited exactly once for the configuration above,
	// so this also fails for a declaration that is added without extending the table.
	require.Len(t, localPaths(), len(expected))

	root, diags := config.LoadFromBytes("databricks.yml", []byte(allLocalPathsConfig))
	require.Empty(t, diags)

	var actual []declaration
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitPaths(value, func(p dyn.Path, field resources.LocalPath, v dyn.Value) (dyn.Value, error) {
			by, ok := translatedBy[field.When]
			require.True(t, ok, "unknown TranslateWhen %d for %s", field.When, p)
			actual = append(actual, declaration{
				path:                 p.String(),
				mode:                 field.Mode,
				filter:               field.Filter,
				allowOutsideSyncRoot: field.AllowOutsideSyncRoot,
				translatedBy:         by,
			})
			return v, nil
		})
	})
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}

// isLibraryPath reports whether a field holds a library, which may be outside the sync root.
func isLibraryPath(field resources.LocalPath) bool {
	return field.AllowOutsideSyncRoot
}

func isNotLibraryPath(field resources.LocalPath) bool {
	return !field.AllowOutsideSyncRoot
}
