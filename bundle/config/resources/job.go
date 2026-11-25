package resources

import (
	"context"
	"net/url"
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

func (j *Job) InitializeURL(baseURL url.URL) {
	if j.ID == "" {
		return
	}
	j.URL = workspaceurls.ResourceURL(baseURL, "jobs", j.ID)
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
