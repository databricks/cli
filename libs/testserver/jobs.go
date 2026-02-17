package testserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// venvPython returns the path to the Python executable in a venv.
// On Unix: venv/bin/python
// On Windows: venv\Scripts\python.exe
func venvPython(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

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

			var logs string
			var err error

			if t.PythonWheelTask != nil {
				// Apply python_params override from RunNow request if provided
				taskToExecute := t
				if len(request.PythonParams) > 0 {
					taskToExecute.PythonWheelTask.Parameters = request.PythonParams
				}
				logs, err = s.executePythonWheelTask(job.Settings, taskToExecute)
			} else if t.NotebookTask != nil {
				logs, err = s.executeNotebookTask(t, request.NotebookParams)
			}

			if err != nil {
				taskRun.State.ResultState = jobs.RunResultStateFailed
				s.JobRunOutputs[taskRunId] = jobs.RunOutput{
					Error: err.Error(),
				}
			} else if logs != "" {
				s.JobRunOutputs[taskRunId] = jobs.RunOutput{
					Logs: logs,
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
// For tasks using existing_cluster_id, the venv is cached per cluster to match
// cloud behavior where libraries are cached on running clusters.
// For serverless tasks (environment_key), dependencies are loaded from the environment spec.
func (s *FakeWorkspace) executePythonWheelTask(jobSettings *jobs.JobSettings, task jobs.Task) (string, error) {
	env, cleanup, err := s.getOrCreateClusterEnv(task)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Collect wheel paths from either task libraries or environment dependencies
	var whlPaths []string
	if len(task.Libraries) > 0 {
		// Cluster-based task with libraries
		for _, lib := range task.Libraries {
			if lib.Whl != "" {
				whlPaths = append(whlPaths, lib.Whl)
			}
		}
	} else if task.EnvironmentKey != "" && jobSettings != nil {
		// Serverless task with environment_key
		for _, envItem := range jobSettings.Environments {
			if envItem.EnvironmentKey == task.EnvironmentKey && envItem.Spec != nil {
				whlPaths = append(whlPaths, envItem.Spec.Dependencies...)
				break
			}
		}
	}

	// Install only wheels not yet present in this cluster env,
	// matching cloud behavior where same library path is not reinstalled.
	var newWhlPaths []string
	for _, whlPath := range whlPaths {
		if env.installedLibs[whlPath] {
			continue
		}
		data := s.files[whlPath].Data
		if len(data) == 0 {
			return "", fmt.Errorf("wheel file not found in workspace: %s", whlPath)
		}
		localPath := filepath.Join(env.dir, filepath.Base(whlPath))
		if err := os.WriteFile(localPath, data, 0o644); err != nil {
			return "", fmt.Errorf("failed to write wheel file: %w", err)
		}
		newWhlPaths = append(newWhlPaths, localPath)
		env.installedLibs[whlPath] = true
	}

	if len(newWhlPaths) > 0 {
		installArgs := []string{"pip", "install", "-q", "--python", venvPython(env.venvDir)}
		installArgs = append(installArgs, newWhlPaths...)
		if out, err := exec.Command("uv", installArgs...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("uv pip install failed: %s\n%s", err, out)
		}
	}

	if len(env.installedLibs) == 0 {
		return "", errors.New("no wheel libraries found in task")
	}

	// Run the entry point using runpy with sys.argv[0] set to the package name,
	// matching Databricks cloud behavior.
	wt := task.PythonWheelTask
	script := fmt.Sprintf("import sys; sys.argv[0] = %q; from runpy import run_module; run_module(%q, run_name='__main__')", wt.PackageName, wt.PackageName)
	runArgs := []string{"-c", script}
	runArgs = append(runArgs, wt.Parameters...)

	cmd := exec.Command(venvPython(env.venvDir), runArgs...)
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

	// Normalize trailing newlines to match cloud behavior (exactly one trailing newline)
	return strings.TrimRight(string(output), "\r\n") + "\n", nil
}

// executeNotebookTask executes a notebook task by running the notebook as a Python script.
// The wrapper feature transforms python_wheel_task into notebook_task that calls the wheel.
func (s *FakeWorkspace) executeNotebookTask(task jobs.Task, notebookParams map[string]string) (string, error) {
	if task.NotebookTask == nil {
		return "", errors.New("task has no notebook_task")
	}

	// Read notebook file from workspace (lock already held by caller)
	notebookPath := task.NotebookTask.NotebookPath
	if !strings.HasPrefix(notebookPath, "/") {
		notebookPath = "/" + notebookPath
	}

	// Try both with and without .py extension (notebooks are stored with .py but referenced without)
	notebookData := s.files[notebookPath].Data
	if len(notebookData) == 0 {
		notebookData = s.files[notebookPath+".py"].Data
	}
	if len(notebookData) == 0 {
		return "", fmt.Errorf("notebook not found in workspace: %s (also tried .py)", notebookPath)
	}

	// Create a temporary Python environment for notebook execution
	tmpDir, err := os.MkdirTemp("", "notebook-task-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Preprocess notebook to extract wheel paths and remove Databricks-specific syntax
	processedNotebook, whlPaths := s.preprocessNotebook(string(notebookData), notebookParams)

	// Write processed notebook to temp file
	notebookFile := filepath.Join(tmpDir, "notebook.py")
	if err := os.WriteFile(notebookFile, []byte(processedNotebook), 0o644); err != nil {
		return "", fmt.Errorf("failed to write notebook file: %w", err)
	}

	// Determine Python version from cluster config
	pythonVersion := sparkVersionToPython(task)

	// Create venv for notebook execution
	venvDir := filepath.Join(tmpDir, ".venv")
	uvArgs := []string{"venv", "-q", "--python", pythonVersion, venvDir}
	if out, err := exec.Command("uv", uvArgs...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("uv venv failed: %s\n%s", err, out)
	}

	// Install wheels from %pip commands
	if len(whlPaths) > 0 {
		var localWhlPaths []string
		for _, whlPath := range whlPaths {
			// Read wheel from workspace
			data := s.files[whlPath].Data
			if len(data) == 0 {
				return "", fmt.Errorf("wheel file not found in workspace: %s", whlPath)
			}
			localPath := filepath.Join(tmpDir, filepath.Base(whlPath))
			if err := os.WriteFile(localPath, data, 0o644); err != nil {
				return "", fmt.Errorf("failed to write wheel file: %w", err)
			}
			localWhlPaths = append(localWhlPaths, localPath)
		}

		installArgs := []string{"pip", "install", "-q", "--python", venvPython(venvDir)}
		installArgs = append(installArgs, localWhlPaths...)
		if out, err := exec.Command("uv", installArgs...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("uv pip install failed: %s\n%s", err, out)
		}
	}

	// Execute notebook with Python
	cmd := exec.Command(venvPython(venvDir), notebookFile)

	// Add testserver directory to PYTHONPATH so dbutils.py can be imported
	_, filename, _, _ := runtime.Caller(0)
	testserverDir := filepath.Dir(filename)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+testserverDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("notebook task execution failed: %s\n%s", err, output)
	}

	// Normalize trailing newlines to match cloud behavior (exactly one trailing newline)
	return strings.TrimRight(string(output), "\r\n") + "\n", nil
}

// getOrCreateClusterEnv returns a cached venv for existing clusters or creates
// a fresh one for new clusters. The cleanup function is non-nil only for new
// clusters (whose venvs should be removed after use).
func (s *FakeWorkspace) getOrCreateClusterEnv(task jobs.Task) (*clusterEnv, func(), error) {
	clusterID := task.ExistingClusterId

	if clusterID != "" {
		if env, ok := s.clusterVenvs[clusterID]; ok {
			return env, nil, nil
		}
	}

	tmpDir, err := os.MkdirTemp("", "wheel-task-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	pythonVersion := sparkVersionToPython(task)
	venvDir := filepath.Join(tmpDir, ".venv")

	uvArgs := []string{"venv", "-q", "--python", pythonVersion, venvDir}
	if out, err := exec.Command("uv", uvArgs...).CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, nil, fmt.Errorf("uv venv failed: %s\n%s", err, out)
	}

	env := &clusterEnv{
		dir:           tmpDir,
		venvDir:       venvDir,
		installedLibs: map[string]bool{},
	}

	// Cache venv for existing clusters; use cleanup for new clusters.
	if clusterID != "" {
		s.clusterVenvs[clusterID] = env
		return env, nil, nil
	}

	return env, func() { os.RemoveAll(tmpDir) }, nil
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

	// First check if output exists directly for this run ID
	output, ok := s.JobRunOutputs[runIdInt]
	if ok {
		return Response{Body: output}
	}

	// If not, check if this is a job run ID with tasks
	jobRun, ok := s.JobRuns[runIdInt]
	if ok && len(jobRun.Tasks) > 0 {
		// For single-task jobs, return the task's output
		taskRunId := jobRun.Tasks[0].RunId
		if taskOutput, ok := s.JobRunOutputs[taskRunId]; ok {
			return Response{Body: taskOutput}
		}
	}

	return Response{Body: jobs.RunOutput{}}
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

// preprocessNotebook converts a Databricks notebook to executable Python by:
// - Removing %python magic commands
// - Extracting wheel paths from %pip install commands
// - Removing %pip commands (wheels will be installed via uv)
// - Mocking dbutils functions
// - Converting dbutils.notebook.exit() to print()
func (s *FakeWorkspace) preprocessNotebook(notebook string, params map[string]string) (string, []string) {
	var whlPaths []string
	var result []string

	// Import dbutils mock
	result = append(result, "# Import dbutils mock for local execution")
	result = append(result, "from dbutils import DBUtils")
	if pythonParams, ok := params["__python_params"]; ok {
		result = append(result, fmt.Sprintf("dbutils = DBUtils({'__python_params': %q})", pythonParams))
	} else {
		result = append(result, "dbutils = DBUtils()")
	}
	result = append(result, "")

	lines := strings.Split(notebook, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip %python magic commands
		if trimmed == "%python" {
			continue
		}

		// Extract wheel path from %pip install and skip the line
		if strings.HasPrefix(trimmed, "%pip install") {
			// Extract path from "%pip install --force-reinstall /path/to/wheel.whl"
			parts := strings.Fields(trimmed)
			for i, part := range parts {
				if strings.HasSuffix(part, ".whl") {
					whlPaths = append(whlPaths, part)
					break
				}
				// Handle case where path is in next field
				if (part == "--force-reinstall" || part == "-U") && i+1 < len(parts) {
					if strings.HasSuffix(parts[i+1], ".whl") {
						whlPaths = append(whlPaths, parts[i+1])
						break
					}
				}
			}
			continue
		}

		// dbutils is now mocked at the beginning, so no need to replace calls

		result = append(result, line)
	}

	return strings.Join(result, "\n"), whlPaths
}
