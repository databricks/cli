package testserver

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

const missingJobGitProviderMessage = "git_source.git_provider must be one of: github,gitlab,bitbucketcloud,gitlabenterpriseedition,bitbucketserver,azuredevopsservices,githubenterprise,awscodecommit"

// errNoCodeInWorkspace marks a task there is nothing to execute for, e.g. code
// uploaded as a snapshot zip this server never unpacks. The gap is here, not in
// the job, so the task succeeds.
var errNoCodeInWorkspace = errors.New("task code is not in the workspace")

// taskFailureMessage is what a real workspace reports for a task that failed: a
// generic pointer at the run output, on both the task and the run. Measured on
// serverless against a spark_python_task that raises. The run-level message wraps
// it as "Task <key> failed with message: <this>." with a trailing period.
const taskFailureMessage = "Workload failed, see run output for details"

// taskFailure splits a failed task's output the way jobs/runs/get-output does:
// error carries the exception, error_trace the traceback.
type taskFailure struct {
	message string
	trace   string
}

func (e *taskFailure) Error() string {
	return e.message
}

// newTaskFailure takes the exception from the last line of the task's output,
// where a Python traceback ends, or err if the task wrote nothing.
func newTaskFailure(err error, output string) *taskFailure {
	trimmed := strings.TrimRight(output, "\r\n")
	lastLine := strings.TrimSpace(trimmed[strings.LastIndex(trimmed, "\n")+1:])
	return &taskFailure{message: cmp.Or(lastLine, err.Error()), trace: trimmed}
}

// venvPython returns the path to the Python executable in a venv.
// On Unix: venv/bin/python
// On Windows: venv\Scripts\python.exe
func venvPython(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// validateJobGitSource mirrors Jobs API validation for git_source requests.
func validateJobGitSource(gitSource *jobs.GitSource) *Response {
	if gitSource == nil || gitSource.GitProvider != "" {
		return nil
	}

	response := Response{
		StatusCode: 400,
		Body: map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    missingJobGitProviderMessage,
		},
	}
	return &response
}

func (s *FakeWorkspace) JobsCreate(req Request) Response {
	var request jobs.CreateJob
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	if response := validateJobGitSource(request.GitSource); response != nil {
		return *response
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
	creator := userForToken(req.Token).UserName
	s.Jobs[jobId] = jobs.Job{
		JobId:           jobId,
		Settings:        &jobSettings,
		CreatorUserName: creator,
		RunAsUserName:   creator,
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
	if response := validateJobGitSource(request.NewSettings.GitSource); response != nil {
		return *response
	}

	defer s.LockUnlock()()

	jobFixUps(&request.NewSettings)

	jobId := request.JobId
	prevjob, ok := s.Jobs[jobId]
	if !ok {
		return Response{StatusCode: 403, Body: "{}"}
	}

	// Known cloud quirk (see the test's Badness note): jobs/reset is a full
	// replace, but omitting run_as from new_settings does NOT clear it — cloud
	// keeps the previously configured identity. Mirror that so the local
	// testserver matches cloud against one golden.
	if request.NewSettings.RunAs == nil {
		request.NewSettings.RunAs = prevjob.Settings.RunAs
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

		// Sort depends_on by task_key to simulate the real API which returns
		// dependencies in a different order than submitted.
		slices.SortFunc(task.DependsOn, func(a, b jobs.TaskDependency) int {
			return cmp.Compare(a.TaskKey, b.TaskKey)
		})

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

			// The real Jobs API consumes apply_policy_default_values but does not
			// return it in GET responses; clear it so testserver matches cloud.
			task.NewCluster.ApplyPolicyDefaultValues = false
		}

		// Handle for_each_task inner cluster.
		if task.ForEachTask != nil && task.ForEachTask.Task.NewCluster != nil {
			// Same as above: not returned in GET responses.
			task.ForEachTask.Task.NewCluster.ApplyPolicyDefaultValues = false
		}
	}

	// Handle job cluster new_clusters.
	for i := range jobSettings.JobClusters {
		// Same as above: not returned in GET responses.
		jobSettings.JobClusters[i].NewCluster.ApplyPolicyDefaultValues = false
	}
}

// jobsGetTasksPageSize matches the real Databricks API limit of 100 tasks per jobs.get response.
// https://docs.databricks.com/api/workspace/jobs/get
const jobsGetTasksPageSize = 100

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

	// The backend checks permissions before existence, so a guest without access
	// gets a permission error rather than a 404, even after the job is deleted.
	if isGuestToken(req.Token) && !s.guestHasJobAccess(jobIdInt, userForToken(req.Token).UserName) {
		return jobReadPermissionDenied(userForToken(req.Token).UserName, jobIdInt)
	}

	job, ok := s.Jobs[jobIdInt]
	if !ok {
		// Match the real Jobs API, which echoes the job id in the error message.
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Job %d does not exist.", jobIdInt)},
		}
	}

	job = setSourceIfNotSet(job)

	// Paginate tasks to match real API behavior: max 100 tasks per response.
	// Use page_token (the offset as a string) to fetch subsequent pages.
	if job.Settings != nil && len(job.Settings.Tasks) > jobsGetTasksPageSize {
		offset := 0
		if pageToken := req.URL.Query().Get("page_token"); pageToken != "" {
			offset, err = strconv.Atoi(pageToken)
			if err != nil {
				return Response{
					StatusCode: 400,
					Body:       fmt.Sprintf("Failed to parse page_token: %s", err),
				}
			}
		}

		settingsCopy := *job.Settings
		job.Settings = &settingsCopy

		tasks := settingsCopy.Tasks
		end := min(offset+jobsGetTasksPageSize, len(tasks))
		job.Settings.Tasks = tasks[offset:end]

		if end < len(tasks) {
			job.HasMore = true
			job.NextPageToken = strconv.Itoa(end)
		}

		// On subsequent pages the real API returns empty arrays for properties
		// that were fully included on the first page.
		if offset > 0 {
			job.Settings.JobClusters = nil
			job.Settings.Environments = nil
			job.Settings.Parameters = nil
		}
	}

	return Response{Body: job}
}

// JobsDelete deletes a job. A guest without admin/owner access gets a permission
// error, even after the job is gone (permission check precedes existence check).
func (s *FakeWorkspace) JobsDelete(req Request, jobId int64) Response {
	defer s.LockUnlock()()

	if isGuestToken(req.Token) && !s.guestHasJobAccess(jobId, userForToken(req.Token).UserName) {
		return jobDeletePermissionDenied(userForToken(req.Token).UserName, jobId)
	}

	if _, ok := s.Jobs[jobId]; !ok {
		return Response{StatusCode: 404}
	}
	delete(s.Jobs, jobId)
	return Response{}
}

const permissionDeniedErrorCode = "PERMISSION_DENIED"

func jobPermissionDenied(message string) Response {
	return Response{
		StatusCode: 403,
		Body:       map[string]string{"error_code": permissionDeniedErrorCode, "message": message},
	}
}

// jobManagePermissionDenied is returned when reading a job's permissions without
// Manage access. ElasticJobId mirrors the backend error's identifier shape.
func jobManagePermissionDenied(principal string, jobId int64) Response {
	return jobPermissionDenied(fmt.Sprintf("%s does not have Manage permissions on Job with ID: ElasticJobId(%d). Please contact the owner or an administrator for access.", principal, jobId))
}

func jobReadPermissionDenied(principal string, jobId int64) Response {
	return jobPermissionDenied(fmt.Sprintf("User %s does not have View or Admin or Manage Run or Owner permissions on job %d", principal, jobId))
}

func jobDeletePermissionDenied(principal string, jobId int64) Response {
	return jobPermissionDenied(fmt.Sprintf("User %s does not have Admin or Owner permissions on job %d", principal, jobId))
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

	slices.SortFunc(list, func(a, b jobs.BaseJob) int { return cmp.Compare(a.JobId, b.JobId) })
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

	// A resent request with the same non-empty token returns the run that token started.
	if request.IdempotencyToken != "" {
		if runId, ok := s.JobRunIdempotency[request.IdempotencyToken]; ok {
			return Response{Body: jobs.RunNowResponse{RunId: runId}}
		}
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
			} else if t.SparkPythonTask != nil {
				logs, err = s.executeSparkPythonTask(t)
			}

			switch {
			case errors.Is(err, errNoCodeInWorkspace):
				// Nothing ran, so the task keeps its SUCCESS state.
			case err != nil:
				taskRun.State.ResultState = jobs.RunResultStateFailed
				taskRun.State.StateMessage = taskFailureMessage
				runOutput := jobs.RunOutput{Error: err.Error()}
				// A task this server could not even start (e.g. uv failed) has no
				// traceback to report.
				if failure, ok := errors.AsType[*taskFailure](err); ok {
					runOutput.ErrorTrace = failure.trace
				}
				s.JobRunOutputs[taskRunId] = runOutput
			case logs != "":
				s.JobRunOutputs[taskRunId] = jobs.RunOutput{
					Logs: logs,
				}
			}
		}
	}

	s.JobRuns[runId] = jobs.Run{
		RunId:                runId,
		JobId:                request.JobId,
		State:                &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		RunPageUrl:           fmt.Sprintf("%s/?o=900800700600#job/%d/run/%d", s.url, request.JobId, runId),
		RunType:              jobs.RunTypeJobRun,
		RunName:              runName,
		Tasks:                tasks,
		JobParameters:        runJobParameters(job.Settings, request.JobParameters),
		OverridingParameters: runOverridingParameters(request),
	}

	if request.IdempotencyToken != "" {
		s.JobRunIdempotency[request.IdempotencyToken] = runId
	}

	return Response{Body: jobs.RunNowResponse{RunId: runId}}
}

// runJobParameters mirrors how GetRun resolves job-level parameters: every
// parameter the job defines, with the run's overrides applied on top, sorted by
// name for deterministic output.
func runJobParameters(settings *jobs.JobSettings, overrides map[string]string) []jobs.JobParameter {
	resolved := map[string]jobs.JobParameter{}
	if settings != nil {
		for _, p := range settings.Parameters {
			resolved[p.Name] = jobs.JobParameter{Name: p.Name, Default: p.Default, Value: p.Default}
		}
	}
	for name, value := range overrides {
		p := resolved[name]
		p.Name = name
		p.Value = value
		resolved[name] = p
	}
	if len(resolved) == 0 {
		return nil
	}
	result := make([]jobs.JobParameter, 0, len(resolved))
	for _, p := range resolved {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b jobs.JobParameter) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return result
}

// runOverridingParameters mirrors how GetRun echoes the run's overriding
// parameters. Returns nil when the request set none.
func runOverridingParameters(request jobs.RunNow) *jobs.RunParameters {
	p := jobs.RunParameters{
		DbtCommands:       request.DbtCommands,
		JarParams:         request.JarParams,
		NotebookParams:    request.NotebookParams,
		PipelineParams:    request.PipelineParams,
		PythonNamedParams: request.PythonNamedParams,
		PythonParams:      request.PythonParams,
		SparkSubmitParams: request.SparkSubmitParams,
		SqlParams:         request.SqlParams,
	}
	if len(p.DbtCommands) == 0 && len(p.JarParams) == 0 && len(p.NotebookParams) == 0 &&
		p.PipelineParams == nil && len(p.PythonNamedParams) == 0 && len(p.PythonParams) == 0 &&
		len(p.SparkSubmitParams) == 0 && len(p.SqlParams) == 0 {
		return nil
	}
	return &p
}

// JobsSubmit handles jobs/runs/submit, the one-time run endpoint used by
// `databricks ssh connect` (via client.Jobs.Submit) and the generic
// `databricks jobs submit` command. It records the submitted spec and returns a
// run ID so acceptance tests can assert the request body (e.g. the serverless
// environments / base_environment) and poll runs/get for the resulting run.
//
// Unlike JobsRunNow, the submitted tasks are not executed locally: the SSH
// bootstrap submits a notebook task that only exists in the workspace, and the
// value of this handler for tests is the recorded request, not task output.
func (s *FakeWorkspace) JobsSubmit(req Request) Response {
	var request jobs.SubmitRun
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	if response := validateJobGitSource(request.GitSource); response != nil {
		return *response
	}

	defer s.LockUnlock()()

	runId := nextID()

	// The default run name for one-time runs is "Untitled" (Jobs API behavior).
	runName := cmp.Or(request.RunName, "Untitled")

	// Report each task as RUNNING in both the V1 (state) and V2 (status) shapes.
	// The generic `jobs submit` waiter polls the V1 run-level state, which
	// JobsGetRun drives to TERMINATED on the next poll, while `ssh connect`'s
	// waitForJobToStart polls the V2 per-task status.
	var tasks []jobs.RunTask
	for _, t := range request.Tasks {
		tasks = append(tasks, jobs.RunTask{
			RunId:   nextID(),
			TaskKey: t.TaskKey,
			State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateRunning,
			},
			Status: &jobs.RunStatus{
				State: jobs.RunLifecycleStateV2StateRunning,
			},
		})
	}

	s.JobRuns[runId] = jobs.Run{
		RunId:      runId,
		State:      &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		RunPageUrl: fmt.Sprintf("%s/?o=900800700600#job/run/%d", s.url, runId),
		RunType:    jobs.RunTypeSubmitRun,
		RunName:    runName,
		Tasks:      tasks,
	}

	// No tunnel server runs locally, so synthesize the metadata.json it would
	// publish; `ssh connect` polls for it before connecting.
	if strings.HasPrefix(runName, sshTunnelBootstrapRunPrefix) {
		s.writeSSHTunnelMetadata(request)
	}

	return Response{Body: jobs.SubmitRunResponse{RunId: runId}}
}

const (
	sshTunnelBootstrapRunPrefix = "ssh-server-bootstrap-"
	sshTunnelBootstrapNotebook  = "ssh-server-bootstrap"
	sshTunnelServerPort         = 7772
	sshTunnelClusterID          = "1234-567890-serverless"
)

// writeSSHTunnelMetadata publishes the metadata.json a real tunnel server would
// write next to the bootstrap notebook. Callers must hold the workspace lock.
func (s *FakeWorkspace) writeSSHTunnelMetadata(request jobs.SubmitRun) {
	for _, t := range request.Tasks {
		if t.NotebookTask == nil {
			continue
		}
		metadataPath := strings.TrimSuffix(t.NotebookTask.NotebookPath, sshTunnelBootstrapNotebook) + "metadata.json"
		metadata, err := json.Marshal(map[string]any{
			"port":       sshTunnelServerPort,
			"cluster_id": sshTunnelClusterID,
		})
		if err != nil {
			continue
		}
		s.files[metadataPath] = FileEntry{
			Info: workspace.ObjectInfo{ObjectType: "FILE", Path: metadataPath},
			Data: metadata,
		}
	}
}

// executePythonWheelTask runs a python wheel task locally using uv.
// For tasks using existing_cluster_id, the venv is cached per cluster to match
// cloud behavior where libraries are cached on running clusters.
// For serverless tasks (environment_key), dependencies are loaded from the environment spec.
// wheelExtrasSuffix matches a trailing pip extras suffix on a wheel path, e.g.
// "[train]" in "foo.whl[train]". Anchored right after ".whl" so it never touches
// bracket groups elsewhere in the path.
var wheelExtrasSuffix = regexp.MustCompile(`(?i)(\.whl)\[[^\]]*\]$`)

// stripWheelExtras removes a trailing pip extras suffix from a wheel path.
func stripWheelExtras(p string) string {
	return wheelExtrasSuffix.ReplaceAllString(p, "$1")
}

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
		// A dependency may carry a pip extras suffix (e.g. "foo.whl[train]").
		// The uploaded file is stored under the bare name, so strip the suffix to
		// locate it. We install the bare wheel; extras only affect which transitive
		// deps cloud pulls, which this offline install does not model.
		filePath := stripWheelExtras(whlPath)
		if env.installedLibs[filePath] {
			continue
		}
		data := s.files[filePath].Data
		if len(data) == 0 {
			return "", fmt.Errorf("%w: wheel file not found in workspace: %s", errNoCodeInWorkspace, filePath)
		}
		localPath := filepath.Join(env.dir, filepath.Base(filePath))
		if err := os.WriteFile(localPath, data, 0o644); err != nil {
			return "", fmt.Errorf("failed to write wheel file: %w", err)
		}
		newWhlPaths = append(newWhlPaths, localPath)
		env.installedLibs[filePath] = true
	}

	if len(newWhlPaths) > 0 {
		installArgs := []string{"pip", "install", "-q", "--python", venvPython(env.venvDir)}
		installArgs = append(installArgs, newWhlPaths...)
		if out, err := exec.Command("uv", installArgs...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("uv pip install failed: %s\n%s", err, out)
		}
	}

	if len(env.installedLibs) == 0 {
		return "", fmt.Errorf("%w: no wheel libraries found in task", errNoCodeInWorkspace)
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
		return string(output), newTaskFailure(err, string(output))
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
		return "", fmt.Errorf("%w: notebook not found in workspace: %s (also tried .py)", errNoCodeInWorkspace, notebookPath)
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
		return string(output), newTaskFailure(err, string(output))
	}

	// Normalize trailing newlines to match cloud behavior (exactly one trailing newline)
	return strings.TrimRight(string(output), "\r\n") + "\n", nil
}

// executeSparkPythonTask runs a spark_python_task locally by reading the
// python_file from the workspace and executing it in a uv-created venv.
// For tasks using existing_cluster_id, the venv is cached per cluster to match
// cloud behavior where libraries are cached on running clusters.
func (s *FakeWorkspace) executeSparkPythonTask(task jobs.Task) (string, error) {
	if task.SparkPythonTask == nil {
		return "", errors.New("task has no spark_python_task")
	}

	// Read python file from workspace (lock already held by caller)
	pythonPath := task.SparkPythonTask.PythonFile
	if !strings.HasPrefix(pythonPath, "/") {
		pythonPath = "/" + pythonPath
	}

	pythonData := s.files[pythonPath].Data
	if len(pythonData) == 0 {
		return "", fmt.Errorf("%w: python file not found in workspace: %s", errNoCodeInWorkspace, pythonPath)
	}

	env, cleanup, err := s.getOrCreateClusterEnv(task)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Write python file into the cluster env so it can be executed by the venv.
	pythonFile := filepath.Join(env.dir, filepath.Base(pythonPath))
	if err := os.WriteFile(pythonFile, pythonData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write python file: %w", err)
	}

	runArgs := []string{pythonFile}
	runArgs = append(runArgs, task.SparkPythonTask.Parameters...)

	output, err := exec.Command(venvPython(env.venvDir), runArgs...).CombinedOutput()
	if err != nil {
		return string(output), newTaskFailure(err, string(output))
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

// terminateRun completes the run, rolling task outcomes up into the run-level
// state the way the Jobs API does: one failed task fails the whole run, and the
// run reports INTERNAL_ERROR in the deprecated life_cycle_state even though its
// tasks are TERMINATED (status.state is TERMINATED with RUN_EXECUTION_ERROR).
func terminateRun(run *jobs.Run) {
	for i := range run.Tasks {
		// Tasks that were never executed (jobs/runs/submit) are still running.
		if run.Tasks[i].State.LifeCycleState != jobs.RunLifeCycleStateTerminated {
			run.Tasks[i].State.LifeCycleState = jobs.RunLifeCycleStateTerminated
			run.Tasks[i].State.ResultState = jobs.RunResultStateSuccess
		}
	}

	run.State = &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	}
	for _, task := range run.Tasks {
		if task.State.ResultState != jobs.RunResultStateSuccess {
			run.State.LifeCycleState = jobs.RunLifeCycleStateInternalError
			run.State.ResultState = task.State.ResultState
			run.State.StateMessage = fmt.Sprintf("Task %s failed with message: %s.", task.TaskKey, taskFailureMessage)
			return
		}
	}
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

	// Simulate cloud behavior: first poll returns RUNNING, next the terminal state.
	if run.State.LifeCycleState == jobs.RunLifeCycleStateRunning {
		// Transition stored state to TERMINATED for the next poll.
		terminateRun(&run)
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

// JobsCancelRun settles a run that is still going. The real API cancels
// asynchronously; the caller polls runs/get either way.
func (s *FakeWorkspace) JobsCancelRun(req Request) Response {
	var request jobs.CancelRun
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	run, ok := s.JobRuns[request.RunId]
	if !ok {
		return Response{StatusCode: 404}
	}

	// A run that already finished keeps the outcome it reached.
	if run.State.LifeCycleState == jobs.RunLifeCycleStateRunning {
		run.State = &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateCanceled,
		}
		s.JobRuns[request.RunId] = run
	}

	return Response{}
}

func (s *FakeWorkspace) JobsDeleteRun(req Request) Response {
	var request jobs.DeleteRun
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	return MapDelete(s, s.JobRuns, request.RunId)
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

	lines := strings.SplitSeq(notebook, "\n")
	for line := range lines {
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
