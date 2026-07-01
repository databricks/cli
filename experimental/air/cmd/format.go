package aircmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"go.yaml.in/yaml/v3"
)

// na is the placeholder shown for an empty text-table cell, matching the Python CLI.
const na = "N/A"

// orNA returns s, or "N/A" when s is empty, for text-table cells.
func orNA(s string) string {
	if s == "" {
		return na
	}
	return s
}

// osc8Link wraps label in an OSC 8 terminal hyperlink to url.
// See https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
func osc8Link(label, url string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

// hyperlink renders label as a terminal hyperlink to url when out is a rich
// terminal, otherwise it returns label unchanged. This mirrors the Python CLI's
// Rich link markup, which drops the URL on non-terminals (so piped or captured
// output stays plain text).
func hyperlink(ctx context.Context, out io.Writer, label, url string) string {
	if url == "" || !cmdio.SupportsColor(ctx, out) {
		return label
	}
	return osc8Link(label, url)
}

// reformatYAMLForDisplay re-renders a training-config YAML so multi-line strings
// (notably the `command:` field) appear as `|` block literals instead of the
// quoted "\n"-escaped single line they are stored as, which is unreadable. It
// mirrors Python's _reformat_yaml_for_display (cli_display.py); we skip the
// Rich syntax-highlighted panel and only fix the whitespace. On any parse or
// re-encode failure it returns the original content unchanged.
func reformatYAMLForDisplay(content []byte) string {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return string(content)
	}
	forceLiteralBlockStrings(&node)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return string(content)
	}
	enc.Close()
	return buf.String()
}

// forceLiteralBlockStrings walks a YAML node tree and marks every multi-line
// string scalar for `|` block-literal rendering. The encoder automatically
// falls back to a quoted style when a value can't be represented as a block
// literal (e.g. lines with trailing whitespace), so no explicit guard is needed.
func forceLiteralBlockStrings(node *yaml.Node) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" && strings.Contains(node.Value, "\n") {
		node.Style = yaml.LiteralStyle
	}
	for _, child := range node.Content {
		forceLiteralBlockStrings(child)
	}
}

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
	return statusWord(string(state.LifeCycleState), string(state.ResultState))
}

// statusWord picks the status word to show from a run's lifecycle and result
// states: the result state is the more meaningful one, so it wins when set.
func statusWord(lifeCycle, result string) string {
	if result != "" {
		return result
	}
	if lifeCycle != "" {
		return lifeCycle
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
	s := isoFormat(time.UnixMilli(startMillis))
	return &s
}

// isoFormat renders a time as a Python-style isoformat string in UTC ("+00:00",
// not "Z"; microseconds only when the sub-second part is non-zero), matching
// cli_entrypoint.py:1899.
func isoFormat(t time.Time) string {
	t = t.UTC()
	layout := "2006-01-02T15:04:05-07:00"
	if t.Nanosecond() != 0 {
		layout = "2006-01-02T15:04:05.000000-07:00"
	}
	return t.Format(layout)
}

// submittedDisplay formats the run's start time for the text table as
// "2006-01-02 15:04 UTC", or "N/A" if it hasn't started. Mirrors Python's
// _format_timestamp (cli_display.py); we render in UTC for stable output rather
// than the local zone Python uses.
func submittedDisplay(run *jobs.Run) string {
	startMillis, _ := reportedTiming(run)
	if startMillis == 0 {
		return na
	}
	return time.UnixMilli(startMillis).UTC().Format("2006-01-02 15:04 MST")
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
	if task == nil || task.Compute == nil {
		return ""
	}
	return acceleratorLabel(task.Compute.GpuType, task.Compute.NumGpus)
}

// acceleratorLabel renders a GPU count and type as "8x H100", "" for none, or
// the count alone ("8x") when the type is unrecognized.
func acceleratorLabel(gpuType string, count int) string {
	if count == 0 {
		return ""
	}
	if name := gpuDisplayName(gpuType); name != "" {
		return fmt.Sprintf("%dx %s", count, name)
	}
	return fmt.Sprintf("%dx", count)
}

// gpuDisplayName returns the friendly name for a GPU identifier, falling back to
// the identifier itself when it is not one we recognize.
func gpuDisplayName(gpuType string) string {
	if name, ok := gpuDisplayNames[gpuType]; ok {
		return name
	}
	return gpuType
}

// environment returns the run's runtime image (the training environment), or an
// empty string if the run has no GenAI-compute task.
func environment(run *jobs.Run) string {
	if len(run.Tasks) == 0 {
		return ""
	}
	task := run.Tasks[0].GenAiComputeTask
	if task == nil {
		return ""
	}
	return task.DlRuntimeImage
}

// maxRetries returns the configured retry limit for the run's latest task as a
// display string: "unlimited" for the backend's -1, otherwise the count.
func maxRetries(run *jobs.Run) string {
	if len(run.Tasks) == 0 {
		return "0"
	}
	n := run.Tasks[len(run.Tasks)-1].MaxRetries
	if n < 0 {
		return "unlimited"
	}
	return strconv.Itoa(n)
}
