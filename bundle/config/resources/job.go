package resources

import (
	"context"
	"net/url"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Job struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	jobs.JobSettings

	Permissions []JobPermission `json:"permissions,omitempty"`
}

func (j *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, j)
}

func (j Job) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(j)
}

func (j *Job) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	jobId, err := strconv.Atoi(id)
	if err != nil {
		return false, err
	}
	_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: int64(jobId),
	})
	if err != nil {
		log.Debugf(ctx, "job %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (j *Job) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "job",
		PluralName:    "jobs",
		SingularTitle: "Job",
		PluralTitle:   "Jobs",
	}
}

func (j *Job) InitializeURL(baseURL url.URL) {
	if j.ID == "" {
		return
	}
	j.URL = workspaceurls.ResourceURL(baseURL, "jobs", j.ID)
}

func (j *Job) GetName() string {
	return j.Name
}

func jobTaskLocalPaths(base dyn.Pattern) []LocalPath {
	return []LocalPath{
		{
			Pattern: base.Append(dyn.Key("notebook_task"), dyn.Key("notebook_path")),
			Mode:    TranslateModeNotebook,
		},
		{
			Pattern: base.Append(dyn.Key("spark_python_task"), dyn.Key("python_file")),
			Mode:    TranslateModeFile,
		},
		{
			Pattern: base.Append(dyn.Key("dbt_task"), dyn.Key("project_directory")),
			Mode:    TranslateModeDirectory,
		},
		{
			Pattern: base.Append(dyn.Key("sql_task"), dyn.Key("file"), dyn.Key("path")),
			Mode:    TranslateModeFile,
		},
		{
			Pattern: base.Append(dyn.Key("alert_task"), dyn.Key("workspace_path")),
			Mode:    TranslateModeFile,
		},
		{
			Pattern: base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("requirements")),
			Mode:    TranslateModeFile,
		},
		{
			// The AI Runtime task runs this bash script on each node; the backend
			// reads it as a workspace file, so translate the local path to its
			// remote (or immutable-snapshot) location like any other file.
			Pattern: base.Append(dyn.Key("ai_runtime_task"), dyn.Key("deployments"), dyn.AnyIndex(), dyn.Key("command_path")),
			Mode:    TranslateModeFile,
		},
	}
}

func jobTaskLibraryLocalPaths(base dyn.Pattern) []LocalPath {
	return []LocalPath{
		{
			Pattern:              base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
			Mode:                 TranslateModeLocalRelative,
			AllowOutsideSyncRoot: true,
		},
		{
			Pattern:              base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
			Mode:                 TranslateModeLocalRelative,
			AllowOutsideSyncRoot: true,
		},
	}
}

func (j *Job) LocalPaths() []LocalPath {
	tasks := dyn.NewPattern(dyn.Key("tasks"), dyn.AnyIndex())
	forEachTasks := tasks.Append(dyn.Key("for_each_task"), dyn.Key("task"))
	environmentSpec := dyn.NewPattern(dyn.Key("environments"), dyn.AnyIndex(), dyn.Key("spec"))
	dependencies := environmentSpec.Append(dyn.Key("dependencies"), dyn.AnyIndex())

	return slices.Concat(
		jobTaskLocalPaths(tasks),
		jobTaskLocalPaths(forEachTasks),
		[]LocalPath{
			{
				Pattern: dyn.NewPattern(dyn.Key("environment_variables"), dyn.AnyIndex(), dyn.Key("spec"), dyn.Key("files"), dyn.AnyIndex()),
				Mode:    TranslateModeFile,
			},
			{
				Pattern: environmentSpec.Append(dyn.Key("project_environment")),
				Mode:    TranslateModeFile,
			},
		},
		jobTaskLibraryLocalPaths(tasks),
		[]LocalPath{
			{
				Pattern:              dependencies,
				Mode:                 TranslateModeLocalRelativeWithPrefix,
				Filter:               OnlyLocalLibraries,
				AllowOutsideSyncRoot: true,
			},
			{
				Pattern:              dependencies,
				Mode:                 TranslateModeEnvironmentPipFlag,
				Filter:               OnlyPipFlagPaths,
				AllowOutsideSyncRoot: true,
			},
		},
		jobTaskLibraryLocalPaths(forEachTasks),
	)
}

// SkipLocalPaths reports whether the job uses a Git source, in which case its
// task paths refer to files in the repository. A job is nil if its YAML entry is empty.
func (j *Job) SkipLocalPaths() bool {
	return j != nil && j.GitSource != nil
}

func (j *Job) LibraryLocations() []LibraryLocation {
	tasks := dyn.NewPattern(dyn.Key("tasks"), dyn.AnyIndex())
	forEachTasks := tasks.Append(dyn.Key("for_each_task"), dyn.Key("task"))

	return []LibraryLocation{
		{
			Pattern: tasks.Append(dyn.Key("libraries")),
			Kind:    LibraryList,
		},
		{
			Pattern: forEachTasks.Append(dyn.Key("libraries")),
			Kind:    LibraryList,
		},
		{
			Pattern: dyn.NewPattern(dyn.Key("environments"), dyn.AnyIndex(), dyn.Key("spec"), dyn.Key("dependencies")),
			Kind:    DependencyList,
		},
		{
			// The AI Runtime task's code_source_path is a local archive (typically an
			// artifact-built .tar.gz) that must be uploaded and referenced by its remote
			// path, exactly like a wheel or jar library.
			Pattern: tasks.Append(dyn.Key("ai_runtime_task"), dyn.Key("code_source_path")),
			Kind:    LibraryFile,
		},
		{
			Pattern: forEachTasks.Append(dyn.Key("ai_runtime_task"), dyn.Key("code_source_path")),
			Kind:    LibraryFile,
		},
	}
}
