package run

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Default timeout for waiting for a job run to complete.
var jobRunTimeout time.Duration = 24 * time.Hour

type jobRunner struct {
	key

	bundle *bundle.Bundle
	job    *resources.Job
}

func (r *jobRunner) Name() string {
	if r.job == nil || r.job.JobSettings == nil {
		return ""
	}
	return r.job.JobSettings.Name
}

func isFailed(task jobs.RunTask) bool {
	return task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
		(task.State.LifeCycleState == jobs.RunLifeCycleStateTerminated &&
			task.State.ResultState == jobs.RunResultStateFailed)
}

func isSuccess(task jobs.RunTask) bool {
	return task.State.LifeCycleState == jobs.RunLifeCycleStateTerminated &&
		task.State.ResultState == jobs.RunResultStateSuccess
}

func (r *jobRunner) logFailedTasks(ctx context.Context, runId int64) {
	w := r.bundle.WorkspaceClient()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runId,
	})
	if err != nil {
		log.Errorf(ctx, "failed to log job run. Error: %s", err)
		return
	}
	if run.State.ResultState == jobs.RunResultStateSuccess {
		return
	}
	for _, task := range run.Tasks {
		if isSuccess(task) {
			log.Infof(ctx, "task %s completed successfully", green(task.TaskKey))
		} else if isFailed(task) {
			taskInfo, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{
				RunId: task.RunId,
			})
			if err != nil {
				log.Errorf(ctx, "task %s failed. Unable to fetch error trace: %s", red(task.TaskKey), err)
				continue
			}
			if progressLogger, ok := cmdio.FromContext(ctx); ok {
				progressLogger.Log(progress.NewTaskErrorEvent(task.TaskKey, taskInfo.Error, taskInfo.ErrorTrace))
			}
			log.Errorf(ctx, "Task %s failed!\nError:\n%s\nTrace:\n%s",
				red(task.TaskKey), taskInfo.Error, taskInfo.ErrorTrace)
		} else {
			log.Infof(ctx, "task %s is in state %s",
				yellow(task.TaskKey), task.State.LifeCycleState)
		}
	}
}

func pullRunIdCallback(runId *int64) func(info *jobs.Run) {
	return func(i *jobs.Run) {
		if *runId == 0 {
			*runId = i.RunId
		}
	}
}

func logDebugCallback(ctx context.Context, runId *int64) func(info *jobs.Run) {
	var prevState *jobs.RunState
	return func(i *jobs.Run) {
		state := i.State
		if state == nil {
			return
		}

		// Log the job run URL as soon as it is available.
		if prevState == nil {
			log.Infof(ctx, "Run available at %s", i.RunPageUrl)
		}
		if prevState == nil || prevState.LifeCycleState != state.LifeCycleState {
			log.Infof(ctx, "Run status: %s", i.State.LifeCycleState)
			prevState = state
		}
	}
}

func logProgressCallback(ctx context.Context, progressLogger *cmdio.Logger) func(info *jobs.Run) {
	var prevState *jobs.RunState
	return func(i *jobs.Run) {
		state := i.State
		if state == nil {
			return
		}

		if prevState == nil {
			openRunUrl(i.RunPageUrl)
			progressLogger.Log(progress.NewJobRunUrlEvent(i.RunPageUrl))
		}

		if prevState != nil && prevState.LifeCycleState == state.LifeCycleState &&
			prevState.ResultState == state.ResultState {
			return
		} else {
			prevState = state
		}

		event := &progress.JobProgressEvent{
			Timestamp: time.Now(),
			JobId:     i.JobId,
			RunId:     i.RunId,
			RunName:   i.RunName,
			State:     *i.State,
		}

		// log progress events to stderr
		progressLogger.Log(event)

		// log progress events in using the default logger
		log.Infof(ctx, event.String())
	}
}

func handleNotebookResultsCallback(ctx context.Context, progressLogger *cmdio.Logger, workspaceClient *databricks.WorkspaceClient) func(info *jobs.Run) {
	loggedLineCount := 0

	return func(i *jobs.Run) {
		details, err := workspaceClient.Jobs.GetRun(ctx, jobs.GetRunRequest{
			RunId: i.RunId,
		})

		if err != nil {
			return
		}

		// Loop over details.Tasks and export each task
		for _, task := range details.Tasks {
			var exportReq workspace.ExportRequest
			exportReq.Path = fmt.Sprintf("/Workspace/__databricks_jobs_tmp/job-%d-run-%d/notebook", details.JobId, task.RunId)
			exportReq.Format = "JUPYTER"
			response, err := workspaceClient.Workspace.Export(ctx, exportReq)

			if err != nil {
				return
			}

			notebookContent, err := base64.StdEncoding.DecodeString(response.Content)

			if err != nil {
				// Ignore for now
			} else {
				writeNotebookContentToFile(string(notebookContent))
				renderedNotebook := renderNotebook()
				filteredNotebook := filterNootebookLines(renderedNotebook)
				filteredLines := strings.Split(filteredNotebook, "\n")

				// Only print lines that haven't been logged before
				if loggedLineCount < len(filteredLines) && filteredNotebook != "" {
					newLines := filteredLines[loggedLineCount:]

					// log progress events in using the default logger
					progressLogger.Writer.Write([]byte(strings.Join(newLines, "\n")))
					progressLogger.Writer.Write([]byte("\n"))

					loggedLineCount = len(filteredLines)
				}
			}
		}
	}
}

func (r *jobRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	runId := new(int64)

	err = r.convertPythonParams(opts)
	if err != nil {
		return nil, err
	}

	// construct request payload from cmd line flags args
	req, err := opts.Job.toPayload(r.job, jobID)
	if err != nil {
		return nil, err
	}

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))

	w := r.bundle.WorkspaceClient()

	// gets the run id from inside Jobs.RunNowAndWait
	pullRunId := pullRunIdCallback(runId)

	// callback to log status updates to the universal log destination.
	// Called on every poll request
	logDebug := logDebugCallback(ctx, runId)

	// callback to log progress events. Called on every poll request
	progressLogger, ok := cmdio.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no progress logger found")
	}
	logProgress := logProgressCallback(ctx, progressLogger)
	handleNotebookResults := handleNotebookResultsCallback(ctx, progressLogger, w)

	waiter, err := w.Jobs.RunNow(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("cannot start job")
	}

	if opts.NoWait {
		details, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{
			RunId: waiter.RunId,
		})
		progressLogger.Log(progress.NewJobRunUrlEvent(details.RunPageUrl))
		return nil, err
	}

	run, err := waiter.OnProgress(func(r *jobs.Run) {
		pullRunId(r)
		logDebug(r)
		logProgress(r)
		if opts.LogResults {
			handleNotebookResults(r)
		}
	}).GetWithTimeout(jobRunTimeout)
	if err != nil && runId != nil {
		r.logFailedTasks(ctx, *runId)
	}
	if err != nil {
		return nil, err
	}
	if run.State.LifeCycleState == jobs.RunLifeCycleStateSkipped {
		log.Infof(ctx, "Run was skipped!")
		return nil, fmt.Errorf("run skipped: %s", run.State.StateMessage)
	}

	switch run.State.ResultState {
	// The run was canceled at user request.
	case jobs.RunResultStateCanceled:
		log.Infof(ctx, "Run was cancelled!")
		return nil, fmt.Errorf("run canceled: %s", run.State.StateMessage)

	// The task completed with an error.
	case jobs.RunResultStateFailed:
		log.Infof(ctx, "Run has failed!")
		return nil, fmt.Errorf("run failed: %s", run.State.StateMessage)

	// The task completed successfully.
	case jobs.RunResultStateSuccess:
		log.Infof(ctx, "Run has completed successfully!")
		return output.GetJobOutput(ctx, r.bundle.WorkspaceClient(), *runId)

	// The run was stopped after reaching the timeout.
	case jobs.RunResultStateTimedout:
		log.Infof(ctx, "Run has timed out!")
		return nil, fmt.Errorf("run timed out: %s", run.State.StateMessage)
	}

	return nil, err
}

func (r *jobRunner) convertPythonParams(opts *Options) error {
	if r.bundle.Config.Experimental != nil && !r.bundle.Config.Experimental.PythonWheelWrapper {
		return nil
	}

	needConvert := false
	for _, task := range r.job.Tasks {
		if task.PythonWheelTask != nil {
			needConvert = true
			break
		}
	}

	if !needConvert {
		return nil
	}

	if len(opts.Job.pythonParams) == 0 {
		return nil
	}

	if opts.Job.notebookParams == nil {
		opts.Job.notebookParams = make(map[string]string)
	}

	if len(opts.Job.pythonParams) > 0 {
		if _, ok := opts.Job.notebookParams["__python_params"]; ok {
			return fmt.Errorf("can't use __python_params as notebook param, the name is reserved for internal use")
		}
		p, err := json.Marshal(opts.Job.pythonParams)
		if err != nil {
			return err
		}
		opts.Job.notebookParams["__python_params"] = string(p)
	}

	return nil
}

func (r *jobRunner) Cancel(ctx context.Context) error {
	w := r.bundle.WorkspaceClient()
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	runs, err := w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{
		ActiveOnly: true,
		JobId:      jobID,
	})

	if err != nil {
		return err
	}

	if len(runs) == 0 {
		return nil
	}

	errGroup, errCtx := errgroup.WithContext(ctx)
	for _, run := range runs {
		runId := run.RunId
		errGroup.Go(func() error {
			wait, err := w.Jobs.CancelRun(errCtx, jobs.CancelRun{
				RunId: runId,
			})
			if err != nil {
				return err
			}
			// Waits for the Terminated or Skipped state
			_, err = wait.GetWithTimeout(jobRunTimeout)
			return err
		})
	}

	return errGroup.Wait()
}

func (r *jobRunner) ParseArgs(args []string, opts *Options) error {
	return r.posArgsHandler().ParseArgs(args, opts)
}

func (r *jobRunner) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return r.posArgsHandler().CompleteArgs(args, toComplete)
}

func writeNotebookContentToFile(content string) {
	// Write the content of the notebook to a file
	// so that nbpreview can read it
	file, err := os.Create("/tmp/dab_notebook_output.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println(err)
	}
}

// Function to render the notebook using nbpreview
func renderNotebook() string {
	cmd := exec.Command("nbpreview", "--decorated", "/tmp/dab_notebook_output.txt")

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
	}

	outputStr := string(output)
	return outputStr
}

// Hack to open the run URL in a browser tab in the background using AppleScript.
// We have to do this because the notebook on the run URL page doesn't exist until
// the page is loaded. This should be done in the webapp.
func openRunUrl(url string) {
	osascript := fmt.Sprintf("tell application \"Google Chrome\" to tell window 1 to make new tab with properties {URL:\"%s\"}", url)
	cmd := exec.Command("osascript", "-e", osascript)

	_, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
	}
}

// Function to filter out code cells from the nbpreview output
func filterNootebookLines(input string) string {
	lines := strings.Split(input, "\n")

	// Prefixes to filter out
	prefixes := []string{"     ╭", "     │", "     ╰", "[0]:", "      🌐", "      file:///var"}

	startsWithPrefix := func(line string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				return true
			}
		}
		return false
	}

	// Filter out lines that start with the prefixes or are empty
	var filteredLines []string
	for _, line := range lines {
		if !startsWithPrefix(line) && strings.TrimSpace(line) != "" {
			trimmedLine := strings.TrimSpace(line)
			filteredLines = append(filteredLines, trimmedLine)
		}
	}

	// Join the filtered lines back into a single string
	return strings.Join(filteredLines, "\n")
}
