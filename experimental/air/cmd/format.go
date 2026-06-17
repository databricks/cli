package aircmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// gpuDisplayNames maps the GPU identifiers returned by the backend to the short
// names we show to users. Unknown identifiers are shown unchanged.
var gpuDisplayNames = map[string]string{
	"h100_80gb":  "H100",
	"a10":        "A10",
	"GPU_1xA10":  "A10",
	"GPU_8xH100": "H100",
	"GPU_1xH100": "H100",
}

// runStatus returns the single status word to show for a run. The backend
// reports two values: a lifecycle state (e.g. PENDING, RUNNING) and, once the
// run has finished, a result state (e.g. SUCCESS, FAILED). The result state is
// the more meaningful one, so we prefer it when it is set.
func runStatus(state *jobs.RunState) string {
	if state == nil {
		return "UNKNOWN"
	}
	if state.ResultState != "" {
		return string(state.ResultState)
	}
	if state.LifeCycleState != "" {
		return string(state.LifeCycleState)
	}
	return "UNKNOWN"
}

// reportedTiming returns the run's start and end times (epoch milliseconds),
// preferring the last task's window over the run-level times so a retried run
// reports its latest attempt. Mirrors Python's _reported_attempt_timing
// (cli_display.py:78-87).
func reportedTiming(run *jobs.Run) (startMillis, endMillis int64) {
	startMillis, endMillis = run.StartTime, run.EndTime
	if n := len(run.Tasks); n > 0 {
		last := run.Tasks[n-1]
		if last.StartTime > 0 {
			startMillis = last.StartTime
		}
		if last.EndTime > 0 {
			endMillis = last.EndTime
		}
	}
	return startMillis, endMillis
}

// startedAt returns the run's start time as a Python-isoformat string ("+00:00",
// not "Z"; microseconds only when non-zero, cli_entrypoint.py:1899), or nil if it
// hasn't started.
func startedAt(run *jobs.Run) *string {
	startMillis, _ := reportedTiming(run)
	if startMillis == 0 {
		return nil
	}
	t := time.UnixMilli(startMillis).UTC()
	layout := "2006-01-02T15:04:05-07:00"
	if t.Nanosecond() != 0 {
		layout = "2006-01-02T15:04:05.000000-07:00"
	}
	s := t.Format(layout)
	return &s
}

// durationSeconds returns how long the run has taken, in whole seconds, or nil
// if it has not started. For a finished run this is the elapsed time of the
// reported attempt; for a still-running run it is the time since it started.
func durationSeconds(run *jobs.Run) *int64 {
	startMillis, endMillis := reportedTiming(run)
	if startMillis == 0 {
		return nil
	}
	if endMillis == 0 {
		// Still running: measure against the current time.
		endMillis = time.Now().UnixMilli()
	}
	d := roundMillisToSeconds(endMillis - startMillis)
	return &d
}

// roundMillisToSeconds rounds milliseconds to whole seconds, half to even, to
// match Python's round() (cli_entrypoint.py:1903).
func roundMillisToSeconds(ms int64) int64 {
	return int64(math.RoundToEven(float64(ms) / 1000))
}

// dashboardURL builds {host}/jobs/runs/{id}?o={workspace_id}, matching Python
// (cli_entrypoint.py:1911). The ?o= workspace id deep-links to the right
// workspace on multi-workspace accounts.
func dashboardURL(host string, runID, workspaceID int64) string {
	return fmt.Sprintf("%s/jobs/runs/%d?o=%d", strings.TrimRight(host, "/"), runID, workspaceID)
}

// formatDuration turns a number of seconds into a compact human string such as
// "1h 2m 3s". Trailing zero units are dropped, but a lone "0s" is kept so the
// result is never empty.
func formatDuration(totalSeconds int64) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	return strings.Join(parts, " ")
}

// latestAttemptNumber returns the retry count of the run's most recent task.
// Tasks start at attempt 0, so a value of 0 means the run has not been retried.
func latestAttemptNumber(run *jobs.Run) int {
	if len(run.Tasks) == 0 {
		return 0
	}
	return run.Tasks[len(run.Tasks)-1].AttemptNumber
}

// experimentName returns the MLflow experiment name for the run, or nil if there
// isn't one. Experiment names are often stored under a user's home folder (e.g.
// "/Users/me@example.com/my-experiment"); we strip that prefix so users see just
// the experiment name they chose.
func experimentName(run *jobs.Run) *string {
	if len(run.Tasks) == 0 {
		return nil
	}
	task := run.Tasks[0].GenAiComputeTask
	if task == nil || task.MlflowExperimentName == "" {
		return nil
	}
	name := stripExperimentUserPrefix(task.MlflowExperimentName)
	return &name
}

// stripExperimentUserPrefix removes a leading "/Users/<user>/" from an
// experiment name, leaving the remainder. Names without that prefix are returned
// unchanged.
func stripExperimentUserPrefix(name string) string {
	if !strings.HasPrefix(name, "/Users/") {
		return name
	}
	// Split into ["", "Users", "<user>", "<rest>"]; keep "<rest>".
	parts := strings.SplitN(name, "/", 4)
	if len(parts) == 4 {
		return parts[3]
	}
	return name
}

// accelerators returns a short description of the GPUs the run uses, such as
// "8x H100", or an empty string if the run has no GPU compute attached.
func accelerators(run *jobs.Run) string {
	if len(run.Tasks) == 0 {
		return ""
	}
	task := run.Tasks[0].GenAiComputeTask
	if task == nil || task.Compute == nil || task.Compute.NumGpus == 0 {
		return ""
	}
	return fmt.Sprintf("%dx %s", task.Compute.NumGpus, gpuDisplayName(task.Compute.GpuType))
}

// gpuDisplayName returns the friendly name for a GPU identifier, falling back to
// the identifier itself when it is not one we recognize.
func gpuDisplayName(gpuType string) string {
	if name, ok := gpuDisplayNames[gpuType]; ok {
		return name
	}
	return gpuType
}
