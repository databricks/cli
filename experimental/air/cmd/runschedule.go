package aircmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// buildCronSchedule maps the run YAML's schedule block onto the Jobs
// CronSchedule. pause_status is optional; an empty value lets the Jobs default
// (UNPAUSED) apply.
func buildCronSchedule(s *scheduleConfig) *jobs.CronSchedule {
	if s == nil {
		return nil
	}
	c := &jobs.CronSchedule{
		QuartzCronExpression: s.QuartzCronExpression,
		TimezoneId:           s.TimezoneID,
	}
	if s.PauseStatus != "" {
		c.PauseStatus = jobs.PauseStatus(s.PauseStatus)
	}
	return c
}

// buildJobSettings assembles the persistent-job settings for a scheduled
// workload: the same ai_runtime_task and environment as an ephemeral submit,
// carried on a Task (not SubmitTask) with the cron schedule attached.
func buildJobSettings(cfg *runConfig, commandPath, dlImage, usagePolicyID string, snap snapshotResult, deps []string) jobs.JobSettings {
	task := buildAiRuntimeTask(cfg, commandPath, snap)

	maxRetries := cfg.maxRetries()
	t := jobs.Task{
		TaskKey:        cfg.ExperimentName,
		RunIf:          jobs.RunIfAllSuccess,
		AiRuntimeTask:  &task,
		EnvironmentKey: aiRuntimeEnvironmentKey,
		MaxRetries:     maxRetries,
		// retry_on_timeout only makes sense when retries are allowed (matches the
		// ephemeral submit path).
		RetryOnTimeout:  maxRetries > 0,
		ForceSendFields: []string{"MaxRetries"},
	}

	return jobs.JobSettings{
		Name:           cfg.ExperimentName,
		BudgetPolicyId: usagePolicyID,
		TimeoutSeconds: cfg.timeoutSeconds(),
		Tasks:          []jobs.Task{t},
		Environments:   buildAiRuntimeEnvironments(dlImage, deps),
		Schedule:       buildCronSchedule(cfg.Schedule),
	}
}

// findJobByName resolves an existing job to update. The CLI keeps no state, so a
// scheduled workload is keyed on its (unique-by-convention) name: zero matches
// means create a new job, one means update it in place, and more than one is
// ambiguous — the caller can't tell which to overwrite.
func findJobByName(ctx context.Context, w *databricks.WorkspaceClient, name string) (int64, error) {
	it := w.Jobs.List(ctx, jobs.ListJobsRequest{Name: name, Limit: 100})
	var matches []int64
	for it.HasNext(ctx) {
		job, err := it.Next(ctx)
		if err != nil {
			return 0, err
		}
		// The list Name filter is a case-insensitive match; require an exact name
		// so "run" doesn't collide with "Run".
		if job.Settings != nil && job.Settings.Name == name {
			matches = append(matches, job.JobId)
		}
	}
	switch len(matches) {
	case 0:
		return 0, nil
	case 1:
		return matches[0], nil
	default:
		return 0, fmt.Errorf("found %d jobs named %q; the name is not unique, so air run cannot tell which to update — rename or delete the duplicates, or use a unique experiment_name", len(matches), name)
	}
}

// createScheduledJob turns a workload with a schedule into a persistent,
// scheduled Databricks job. It uploads the same launch artifacts as an ephemeral
// run, then upserts by name: an existing job with the same name is updated in
// place (so re-running doesn't pile up duplicates), otherwise a new one is
// created. It returns the job id, its URL, and whether the job was created (vs
// updated).
func createScheduledJob(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath string) (jobID int64, url string, created bool, err error) {
	prep, err := prepareWorkload(ctx, w, cfg, configPath)
	if err != nil {
		return 0, "", false, err
	}
	settings := buildJobSettings(cfg, prep.commandPath, prep.dlImage, prep.usagePolicyID, prep.snap, prep.deps)

	existingID, err := findJobByName(ctx, w, cfg.ExperimentName)
	if err != nil {
		return 0, "", false, err
	}

	if existingID != 0 {
		if err := w.Jobs.Reset(ctx, jobs.ResetJob{JobId: existingID, NewSettings: settings}); err != nil {
			return 0, "", false, err
		}
		jobID = existingID
	} else {
		resp, err := w.Jobs.Create(ctx, jobs.CreateJob{
			Name:           settings.Name,
			BudgetPolicyId: settings.BudgetPolicyId,
			TimeoutSeconds: settings.TimeoutSeconds,
			Tasks:          settings.Tasks,
			Environments:   settings.Environments,
			Schedule:       settings.Schedule,
		})
		if err != nil {
			return 0, "", false, err
		}
		jobID = resp.JobId
		created = true
	}

	url = strings.TrimRight(w.Config.Host, "/") + "/jobs/" + strconv.FormatInt(jobID, 10)
	return jobID, url, created, nil
}
