package aircmd

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// trainingETA is a best-effort estimate of the wall-clock time remaining for a
// running training job, derived from MLflow progress signals.
//
// It is only produced when MLflow carries both a progress metric and a known
// total. That combination is logged by HuggingFace Trainer's MLflow integration
// (a metric per training log carrying the global step and fractional epoch, plus
// `max_steps` / `num_train_epochs` as params); AIR itself logs only system
// metrics, which are a wall-clock heartbeat, not training progress. For a run
// without a known total there is no reliable denominator, so no estimate is
// shown rather than a misleading one.
type trainingETA struct {
	// RemainingSeconds is the projected time to completion.
	RemainingSeconds int64
	// Progress is a short human breadcrumb, e.g. "step 4120/10000" or "epoch 1.8/3".
	Progress string
}

// etaProgressMetric is the MLflow metric HuggingFace Trainer logs on every
// training log call. Each point carries the fractional epoch as its value and
// the global step as its MLflow step, so a single metric history drives both the
// step-based and epoch-based estimates.
const etaProgressMetric = "epoch"

// etaWindowPoints bounds how many trailing history points feed the rate, so the
// estimate reflects recent throughput rather than a warmup-skewed whole-run
// average.
const etaWindowPoints = 10

// detailed renders the ETA for the single-run view: "~48m 20s · step 4120/10000".
func (e *trainingETA) detailed() string {
	return fmt.Sprintf("~%s · %s", formatDuration(e.RemainingSeconds), e.Progress)
}

// compact renders the ETA for the list table's ETA column: "~48m 20s".
func (e *trainingETA) compact() string {
	return "~" + formatDuration(e.RemainingSeconds)
}

// estimateTrainingETA fetches a running run's MLflow params and progress-metric
// history and projects the remaining time. It returns nil whenever an estimate
// can't be made (no known total, too little history, an API error): the ETA is a
// convenience, so any failure is logged and treated as "no estimate" rather than
// failing the command.
func estimateTrainingETA(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) *trainingETA {
	if mlflowRunID == "" {
		return nil
	}

	resp, err := w.Experiments.GetRun(ctx, ml.GetRunRequest{RunId: mlflowRunID})
	if err != nil {
		log.Debugf(ctx, "air: could not fetch MLflow run for ETA: %v", err)
		return nil
	}
	if resp.Run == nil || resp.Run.Data == nil {
		return nil
	}

	// The total is only present for Trainer runs; skip the metric-history fetch
	// entirely when it is absent, so a run we can't estimate costs one call, not two.
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
		log.Debugf(ctx, "air: could not fetch %q metric history for ETA: %v", etaProgressMetric, err)
		return nil
	}
	return computeETA(params, history)
}

// computeETA is the pure projection: given a run's params and the ascending
// history of the progress metric, it returns the remaining-time estimate, or nil
// when it can't be computed. Split from estimateTrainingETA so it can be tested
// without an API client.
//
// When max_steps is set (> 0) HuggingFace Trainer trains by step and overrides
// num_train_epochs, so the step total wins when both are present.
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

// stepETA projects remaining time from the global step (the metric's MLflow
// step) against a known max_steps.
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
		Progress:         fmt.Sprintf("step %d/%d", current, maxSteps),
	}
}

// epochETA projects remaining time from the fractional epoch (the metric's
// value) against a known num_train_epochs.
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
		Progress:         fmt.Sprintf("epoch %.1f/%s", current, trimFloat(totalEpochs)),
	}
}

// paramMap indexes MLflow params by key.
func paramMap(params []ml.Param) map[string]string {
	m := make(map[string]string, len(params))
	for _, p := range params {
		m[p.Key] = p.Value
	}
	return m
}

// positiveInt parses s as an integer and reports it only when it is > 0. Trainer
// logs max_steps=-1 when training by epochs, which this rejects.
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

// trimFloat renders a float without trailing zeros: 3.0 -> "3", 2.5 -> "2.5".
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
