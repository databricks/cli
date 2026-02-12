package testserver

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func (s *FakeWorkspace) JobsCreate(req Request) Response {
	var request jobs.CreateJob
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	jobId := nextID()

	jobSettings := jobs.JobSettings{}
	if err := jsonConvert(request, &jobSettings); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Cannot convert request to jobSettings: %s", err),
		}
	}

	jobFixUps(&jobSettings)

	// CreatorUserName field is used by TF to check if the resource exists or not. CreatorUserName should be non-empty for the resource to be considered as "exists"
	// https://github.com/databricks/terraform-provider-databricks/blob/main/permissions/permission_definitions.go#L108
	s.Jobs[jobId] = jobs.Job{
		JobId:           jobId,
		Settings:        &jobSettings,
		CreatorUserName: s.CurrentUser().UserName,
		RunAsUserName:   s.CurrentUser().UserName,
		CreatedTime:     nowMilli(),
	}
	return Response{Body: jobs.CreateResponse{JobId: jobId}}
}

func (s *FakeWorkspace) JobsReset(req Request) Response {
	var request jobs.ResetJob
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	jobFixUps(&request.NewSettings)

	jobId := request.JobId
	prevjob, ok := s.Jobs[jobId]
	if !ok {
		return Response{StatusCode: 403, Body: "{}"}
	}

	s.Jobs[jobId] = jobs.Job{
		JobId:           jobId,
		CreatorUserName: prevjob.CreatorUserName,
		RunAsUserName:   prevjob.RunAsUserName,
		CreatedTime:     prevjob.CreatedTime,
		Settings:        &request.NewSettings,
	}
	return Response{Body: ""}
}

func jobFixUps(jobSettings *jobs.JobSettings) {
	if jobSettings.EmailNotifications == nil {
		jobSettings.EmailNotifications = &jobs.JobEmailNotifications{}
	}

	if jobSettings.WebhookNotifications == nil {
		jobSettings.WebhookNotifications = &jobs.WebhookNotifications{}
	}

	jobSettings.ForceSendFields = append(jobSettings.ForceSendFields, "TimeoutSeconds")

	// Add task-level defaults that match AWS cloud behavior
	for i := range jobSettings.Tasks {
		task := &jobSettings.Tasks[i]

		// Set task email notifications to empty struct if not set
		if task.EmailNotifications == nil {
			task.EmailNotifications = &jobs.TaskEmailNotifications{}
		}

		// Set RunIf to ALL_SUCCESS (server-side default)
		if task.RunIf == "" {
			task.RunIf = jobs.RunIfAllSuccess
			task.ForceSendFields = append(task.ForceSendFields, "RunIf")
		}

		// Set TimeoutSeconds to 0 (server-side default)
		task.ForceSendFields = append(task.ForceSendFields, "TimeoutSeconds")

		// Add AWS-specific cluster defaults if new_cluster is present
		if task.NewCluster != nil {
			// Set AWS attributes with server-side defaults
			if task.NewCluster.AwsAttributes == nil {
				task.NewCluster.AwsAttributes = &compute.AwsAttributes{
					Availability: compute.AwsAvailabilitySpotWithFallback,
					ZoneId:       "us-east-1c",
				}
				task.NewCluster.AwsAttributes.ForceSendFields = append(
					task.NewCluster.AwsAttributes.ForceSendFields,
					"Availability",
					"ZoneId",
				)
			}

			// Set data security mode to SINGLE_USER (server-side default)
			if task.NewCluster.DataSecurityMode == "" {
				task.NewCluster.DataSecurityMode = compute.DataSecurityModeSingleUser
				task.NewCluster.ForceSendFields = append(task.NewCluster.ForceSendFields, "DataSecurityMode")
			}

			// Set enable_elastic_disk to false (server-side default)
			task.NewCluster.ForceSendFields = append(task.NewCluster.ForceSendFields, "EnableElasticDisk")
		}
	}
}

func (s *FakeWorkspace) JobsGet(req Request) Response {
	id := req.URL.Query().Get("job_id")
	jobIdInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse job id: %s: %v", err, id),
		}
	}

	defer s.LockUnlock()()

	job, ok := s.Jobs[jobIdInt]
	if !ok {
		return Response{StatusCode: 404}
	}

	job = setSourceIfNotSet(job)
	return Response{Body: job}
}

func (s *FakeWorkspace) JobsList() Response {
	defer s.LockUnlock()()

	list := make([]jobs.BaseJob, 0, len(s.Jobs))
	for _, job := range s.Jobs {
		job = setSourceIfNotSet(job)
		baseJob := jobs.BaseJob{}
		if err := jsonConvert(job, &baseJob); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("failed to convert job to base job: %s", err),
			}
		}
		list = append(list, baseJob)
	}

	sort.Slice(list, func(i, j int) bool { return list[i].JobId < list[j].JobId })
	return Response{Body: jobs.ListJobsResponse{Jobs: list}}
}

func (s *FakeWorkspace) JobsRunNow(req Request) Response {
	var request jobs.RunNow
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	job, ok := s.Jobs[request.JobId]
	if !ok {
		return Response{StatusCode: 404}
	}

	runId := nextID()
	runName := "run-name"
	if job.Settings != nil && job.Settings.Name != "" {
		runName = job.Settings.Name
	}

	// Build task list with individual RunIds, mirroring cloud behavior.
	// Execute PythonWheelTasks locally and store their output.
	var tasks []jobs.RunTask
	if job.Settings != nil {
		for _, t := range job.Settings.Tasks {
			taskRunId := nextID()
			taskRun := jobs.RunTask{
				RunId:   taskRunId,
				TaskKey: t.TaskKey,
				State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateSuccess,
				},
			}
			tasks = append(tasks, taskRun)

			if t.PythonWheelTask != nil {
				logs, err := s.executePythonWheelTask(t)
				if err != nil {
					taskRun.State.ResultState = jobs.RunResultStateFailed
					s.JobRunOutputs[taskRunId] = jobs.RunOutput{
						Error: err.Error(),
					}
				} else {
					s.JobRunOutputs[taskRunId] = jobs.RunOutput{
						Logs: logs,
					}
				}
			}
		}
	}

	s.JobRuns[runId] = jobs.Run{
		RunId:      runId,
		JobId:      request.JobId,
		State:      &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		RunPageUrl: fmt.Sprintf("%s/?o=900800700600#job/%d/run/%d", s.url, request.JobId, runId),
		RunType:    jobs.RunTypeJobRun,
		RunName:    runName,
		Tasks:      tasks,
	}

	return Response{Body: jobs.RunNowResponse{RunId: runId}}
}

// executePythonWheelTask runs a python wheel task locally using uv.
// It finds whl files in the task's libraries, installs them in a temp venv,
// and runs the entry point.
func (s *FakeWorkspace) executePythonWheelTask(task jobs.Task) (string, error) {
	tmpDir, err := os.MkdirTemp("", "wheel-task-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract whl files from the fake workspace to the temp dir.
	var whlPaths []string
	for _, lib := range task.Libraries {
		if lib.Whl == "" {
			continue
		}
		data := s.files[lib.Whl].Data
		if len(data) == 0 {
			return "", fmt.Errorf("wheel file not found in workspace: %s", lib.Whl)
		}
		localPath := filepath.Join(tmpDir, filepath.Base(lib.Whl))
		if err := os.WriteFile(localPath, data, 0o644); err != nil {
			return "", fmt.Errorf("failed to write wheel file: %w", err)
		}
		whlPaths = append(whlPaths, localPath)
	}

	if len(whlPaths) == 0 {
		return "", fmt.Errorf("no wheel libraries found in task")
	}

	// Determine Python version from spark_version (e.g. "13.3.x-snapshot-scala2.12" -> 3.10).
	pythonVersion := sparkVersionToPython(task)

	venvDir := filepath.Join(tmpDir, ".venv")

	// Create venv and install wheels using uv.
	uvArgs := []string{"venv", "-q", "--python", pythonVersion, venvDir}
	if out, err := exec.Command("uv", uvArgs...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("uv venv failed: %s\n%s", err, out)
	}

	installArgs := []string{"pip", "install", "-q", "--python", filepath.Join(venvDir, "bin", "python")}
	installArgs = append(installArgs, whlPaths...)
	if out, err := exec.Command("uv", installArgs...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("uv pip install failed: %s\n%s", err, out)
	}

	// Run the entry point using runpy with sys.argv[0] set to the package name,
	// matching Databricks cloud behavior.
	wt := task.PythonWheelTask
	script := fmt.Sprintf("import sys; sys.argv[0] = %q; from runpy import run_module; run_module(%q, run_name='__main__')", wt.PackageName, wt.PackageName)
	runArgs := []string{"-c", script}
	runArgs = append(runArgs, wt.Parameters...)

	cmd := exec.Command(filepath.Join(venvDir, "bin", "python"), runArgs...)
	if len(wt.NamedParameters) > 0 {
		cmd.Env = os.Environ()
		for k, v := range wt.NamedParameters {
			cmd.Args = append(cmd.Args, fmt.Sprintf("--%s=%s", k, v))
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("wheel task execution failed: %s\n%s", err, output)
	}

	return string(output), nil
}

// sparkVersionToPython maps Databricks Runtime spark_version to Python version.
func sparkVersionToPython(task jobs.Task) string {
	sv := ""
	if task.NewCluster != nil {
		sv = task.NewCluster.SparkVersion
	}

	// Extract major version from strings like "13.3.x-snapshot-scala2.12" or "15.4.x-scala2.12".
	parts := strings.SplitN(sv, ".", 2)
	if len(parts) >= 1 {
		major, err := strconv.Atoi(parts[0])
		if err == nil {
			switch {
			case major >= 16:
				return "3.12"
			case major >= 15:
				return "3.11"
			case major >= 13:
				return "3.10"
			default:
				return "3.9"
			}
		}
	}

	return "3.10"
}

func (s *FakeWorkspace) JobsGetRun(req Request) Response {
	runId := req.URL.Query().Get("run_id")
	runIdInt, err := strconv.ParseInt(runId, 10, 64)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse run id: %s: %v", err, runId),
		}
	}

	defer s.LockUnlock()()

	run, ok := s.JobRuns[runIdInt]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Simulate cloud behavior: first poll returns RUNNING, next returns TERMINATED SUCCESS.
	if run.State.LifeCycleState == jobs.RunLifeCycleStateRunning {
		// Transition stored state to TERMINATED for the next poll.
		run.State = &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		}
		for i := range run.Tasks {
			run.Tasks[i].State = &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateSuccess,
			}
		}
		s.JobRuns[runIdInt] = run

		// Return RUNNING for this poll (before the transition).
		runResp := run
		runResp.State = &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateRunning,
		}
		return Response{Body: runResp}
	}

	return Response{Body: run}
}

func (s *FakeWorkspace) JobsGetRunOutput(req Request) Response {
	runId := req.URL.Query().Get("run_id")
	runIdInt, err := strconv.ParseInt(runId, 10, 64)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse run id: %s: %v", err, runId),
		}
	}

	defer s.LockUnlock()()

	output, ok := s.JobRunOutputs[runIdInt]
	if !ok {
		return Response{Body: jobs.RunOutput{}}
	}

	return Response{Body: output}
}

func setSourceIfNotSet(job jobs.Job) jobs.Job {
	if job.Settings != nil {
		source := "WORKSPACE"
		if job.Settings.GitSource != nil {
			source = "GIT"
		}
		for _, task := range job.Settings.Tasks {
			if task.NotebookTask != nil {
				if task.NotebookTask.Source == "" {
					task.NotebookTask.Source = jobs.Source(source)
				}
				if task.DbtTask != nil {
					if task.DbtTask.Source == "" {
						task.DbtTask.Source = jobs.Source(source)
					}
				}
				if task.SparkPythonTask != nil {
					if task.SparkPythonTask.Source == "" {
						task.SparkPythonTask.Source = jobs.Source(source)
					}
				}
			}
		}
	}
	return job
}
