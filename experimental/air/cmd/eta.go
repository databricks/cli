package aircmd

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// trainingETA is a best-effort estimate of how far a running training job has
// progressed and how much wall-clock time remains.
//
// The interface is a plain MLflow logging contract, not a specific framework: a
// run becomes estimable once it has logged, to its MLflow run,
//   - a total — the `max_steps` param (> 0), or else the `num_train_epochs` param
//     (> 0); and
//   - progress over time — an `epoch` metric logged repeatedly, each point carrying
//     the current global step as its MLflow `step` and the fractional epoch as its
//     value.
//
// HuggingFace Trainer (report_to=["mlflow"]) emits exactly this automatically, but
// nothing here is HF-specific: any logger — Composer, a manual training loop —
// gets a Progress readout by logging the same keys. AIR itself logs only system
// metrics, which are a heartbeat rather than training progress, so a run that logs
// no total shows nothing rather than a misleading estimate.
type trainingETA struct {
	// RemainingSeconds is the projected time to completion.
	RemainingSeconds int64
	// PercentComplete is progress through the run, 0-99 (never 100 while running).
	PercentComplete int
}

// etaProgressMetric is the MLflow metric that drives the estimate. Each point
// carries the fractional epoch as its value and the global step as its MLflow
// step, so a single metric history serves both the step- and epoch-based paths.
const etaProgressMetric = "epoch"

// etaWindowPoints bounds how many trailing history points feed the rate, so the
// estimate tracks recent throughput rather than a warmup-skewed whole-run average.
const etaWindowPoints = 10

// minStepsForETA withholds an estimate until enough steps have elapsed that
// throughput has settled; the first few steps (compile, dataloader warmup) are
// not representative.
const minStepsForETA = 10

// detailed renders the Progress cell for the single-run view: "45% · ~2h 15m left".
func (e *trainingETA) detailed() string {
	return fmt.Sprintf("%d%% · ~%s left", e.PercentComplete, coarseDuration(e.RemainingSeconds))
}

// compact renders the Progress cell for the list table: "45% · ~2h 15m".
func (e *trainingETA) compact() string {
	return fmt.Sprintf("%d%% · ~%s", e.PercentComplete, coarseDuration(e.RemainingSeconds))
}

// coarseDuration renders seconds rounded to the nearest minute, dropping seconds:
// 80 -> "1m", 8100 -> "2h 15m", under ~30s -> "<1m". A remaining-time estimate is
// not precise to the second, so showing seconds would imply false precision.
func coarseDuration(totalSeconds int64) string {
	mins := int64(math.Round(float64(totalSeconds) / 60.0))
	if mins <= 0 {
		return "<1m"
	}
	hours, minutes := mins/60, mins%60
	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	return strings.Join(parts, " ")
}

// estimateTrainingETA fetches a running run's MLflow params and progress-metric
// history and projects progress + remaining time. It returns nil whenever an
// estimate can't be made (no known total, too little history, still warming up, an
// API error): Progress is a convenience, so any failure is logged and treated as
// "no estimate" rather than failing the command.
func estimateTrainingETA(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) *trainingETA {
	if mlflowRunID == "" {
		return nil
	}

	resp, err := w.Experiments.GetRun(ctx, ml.GetRunRequest{RunId: mlflowRunID})
	if err != nil {
		log.Debugf(ctx, "air: could not fetch MLflow run for progress estimate: %v", err)
		return nil
	}
	if resp.Run == nil || resp.Run.Data == nil {
		return nil
	}

	// The total is only present once the contract is met; skip the metric-history
	// fetch entirely when it is absent, so a run we can't estimate costs one call.
	params := paramMap(resp.Run.Data.Params)
	if _, hasSteps := positiveInt(params["max_steps"]); !hasSteps {
		if _, hasEpochs := positiveFloat(params["num_train_epochs"]); !hasEpochs {
			return nil
		}
	}

	history, err := w.Experiments.GetHistoryAll(ctx, ml.GetHistoryRequest{
		RunId:     mlflowRunID,
		MetricKey: etaProgressMetric,
	})
	if err != nil {
		log.Debugf(ctx, "air: could not fetch %q metric history for progress estimate: %v", etaProgressMetric, err)
		return nil
	}
	return computeETA(params, history)
}

// computeETA is the pure projection: given a run's params and the ascending
// history of the progress metric, it returns the progress estimate, or nil when
// it can't be computed. Split from estimateTrainingETA so it can be tested without
// an API client.
//
// When max_steps is set (> 0) the run trains by step and it wins over
// num_train_epochs, matching HuggingFace Trainer's own precedence.
func computeETA(params map[string]string, history []ml.Metric) *trainingETA {
	if len(history) < 2 {
		return nil
	}

	// History should already be ordered by step; sort by timestamp defensively so
	// the rate is measured over increasing wall-clock time.
	points := append([]ml.Metric(nil), history...)
	slices.SortStableFunc(points, func(a, b ml.Metric) int { return cmp.Compare(a.Timestamp, b.Timestamp) })
	if len(points) > etaWindowPoints {
		points = points[len(points)-etaWindowPoints:]
	}
	first, last := points[0], points[len(points)-1]

	// Withhold an estimate during the first few steps while throughput settles.
	if last.Step < minStepsForETA {
		return nil
	}

	elapsedSec := float64(last.Timestamp-first.Timestamp) / 1000.0
	if elapsedSec <= 0 {
		return nil
	}

	if maxSteps, ok := positiveInt(params["max_steps"]); ok {
		return stepETA(first, last, elapsedSec, maxSteps)
	}
	if totalEpochs, ok := positiveFloat(params["num_train_epochs"]); ok {
		return epochETA(first, last, elapsedSec, totalEpochs)
	}
	return nil
}

// stepETA projects from the global step (the metric's MLflow step) against a known
// max_steps.
func stepETA(first, last ml.Metric, elapsedSec float64, maxSteps int64) *trainingETA {
	current := last.Step
	if current <= 0 || current >= maxSteps {
		return nil
	}
	done := last.Step - first.Step
	if done <= 0 {
		return nil
	}
	perSec := float64(done) / elapsedSec
	remaining := float64(maxSteps-current) / perSec
	return &trainingETA{
		RemainingSeconds: int64(math.Round(remaining)),
		PercentComplete:  percentComplete(float64(current), float64(maxSteps)),
	}
}

// epochETA projects from the fractional epoch (the metric's value) against a known
// num_train_epochs.
func epochETA(first, last ml.Metric, elapsedSec, totalEpochs float64) *trainingETA {
	current := last.Value
	if current <= 0 || current >= totalEpochs {
		return nil
	}
	done := last.Value - first.Value
	if done <= 0 {
		return nil
	}
	perSec := done / elapsedSec
	remaining := (totalEpochs - current) / perSec
	return &trainingETA{
		RemainingSeconds: int64(math.Round(remaining)),
		PercentComplete:  percentComplete(current, totalEpochs),
	}
}

// percentComplete is current/total as a whole percent, clamped to 0-99: a running
// run is never shown as 100% complete.
func percentComplete(current, total float64) int {
	p := int(math.Round(current / total * 100))
	if p < 0 {
		return 0
	}
	if p > 99 {
		return 99
	}
	return p
}

// paramMap indexes MLflow params by key.
func paramMap(params []ml.Param) map[string]string {
	m := make(map[string]string, len(params))
	for _, p := range params {
		m[p.Key] = p.Value
	}
	return m
}

// positiveInt parses s as an integer and reports it only when it is > 0. HF
// Trainer logs max_steps=-1 when training by epochs, which this rejects.
func positiveInt(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// positiveFloat parses s as a float and reports it only when it is > 0.
func positiveFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return f, true
}
