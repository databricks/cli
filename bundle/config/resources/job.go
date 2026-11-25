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

func (*Job) Approval() Approval {
	return Approval{}
}

func (j *Job) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "job",
		PluralName:    "jobs",
		SingularTitle: "Job",
		PluralTitle:   "Jobs",
	}
}

func (*Job) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_VIEW",
			CanRun:    "CAN_MANAGE_RUN",
		},
		RequestObjectType:  "jobs",
		ResponseObjectType: "job",
		ObjectIDField:      "id",
		HasIsOwner:         true,
	}
}

func (*Job) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "jobs/%s"}, true
}

func (j *Job) InitializeURL(baseURL url.URL) {
	if j.ID == "" {
		return
	}
	j.URL = workspaceURL(j, baseURL, j.ID)
}

func (j *Job) GetName() string {
	return j.Name
}

func (*Job) RunAsSupport() RunAsSupport {
	return RunAsApplied
}

func (j *Job) ApplyRunAs(runAs *jobs.JobRunAs) {
	if j.RunAs != nil {
		return
	}
	j.RunAs = &jobs.JobRunAs{
		GroupName:            runAs.GroupName,
		ServicePrincipalName: runAs.ServicePrincipalName,
		UserName:             runAs.UserName,
	}
}

func (j *Job) ApplyLegacyRunAs(runAs *jobs.JobRunAs, _ string) {
	j.ApplyRunAs(runAs)
}

func (j *Job) ApplyPresets(p *PresetSettings) {
	j.Name = p.NamePrefix + j.Name
	if len(p.Tags) > 0 {
		if j.Tags == nil {
			// Note: only create this map if tags is not empty, to avoid inserting "tags: {}" entry in the config
			j.Tags = make(map[string]string, len(p.Tags))
		}
		for _, tag := range p.Tags {
			if j.Tags[tag.Key] == "" {
				j.Tags[tag.Key] = tag.Value
			}
		}
	}
	if j.MaxConcurrentRuns == 0 {
		j.MaxConcurrentRuns = p.JobsMaxConcurrentRuns
	}
	if p.TriggerPauseStatus != "" {
		paused := jobs.PauseStatusPaused
		if p.TriggerPauseStatus == TriggerPauseStatusUnpaused {
			paused = jobs.PauseStatusUnpaused
		}

		if j.Schedule != nil && j.Schedule.PauseStatus == "" {
			j.Schedule.PauseStatus = paused
		}
		if j.Continuous != nil && j.Continuous.PauseStatus == "" {
			j.Continuous.PauseStatus = paused
		}
		if j.Trigger != nil && j.Trigger.PauseStatus == "" {
			j.Trigger.PauseStatus = paused
		}
		for i := range j.Triggers {
			if j.Triggers[i].PauseStatus == "" {
				j.Triggers[i].PauseStatus = paused
			}
		}
	}
}

func (j *Job) FieldDefaults(DefaultsContext) []FieldDefault {
	return []FieldDefault{
		// The defaults are the same as for terraform provider latest version (v1.75.0)
		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/jobs/resource_job.go#L532
		{"name", "Untitled"},
		{"max_concurrent_runs", 1},
		{"schedule.pause_status", "UNPAUSED"},
		{"trigger.pause_status", "UNPAUSED"},
		{"continuous.pause_status", "UNPAUSED"},

		// Enable queueing for jobs by default, following the behavior from API 2.2+.
		// As of 2024-04, we're still using API 2.1 which has queueing disabled by default.
		{"queue", map[string]dyn.Value{
			"enabled": dyn.V(true),
		}},

		// https://github.com/databricks/terraform-provider-databricks/blob/v1.75.0/clusters/resource_cluster.go
		{"job_clusters[*].new_cluster.workload_type.clients.notebooks", true},
		{"job_clusters[*].new_cluster.workload_type.clients.jobs", true},
	}
}

func (j *Job) IsRunAsSet() bool {
	return j.RunAs != nil
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
