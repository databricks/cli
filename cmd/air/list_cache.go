package aircmd

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/cache"
)

// The AiTrainingService index path caches hydrated terminal runs on disk:
// terminal runs are immutable, so once we've paid for runs/get + get-output +
// MLflow we persist the finished row and skip those round-trips next time. The
// TTL matches AICM's ~60-day retention, after which the run drops out of the
// index anyway.
const (
	listCacheComponent = "air-list-runs"
	listCacheTTL       = 60 * 24 * time.Hour
)

// listCacheKey fingerprints a cached run. Host isolates workspaces (a Jobs run
// id is unique only within one), matching how libs/cache namespaces entries.
type listCacheKey struct {
	Host  string `json:"host"`
	RunID int64  `json:"run_id"`
}

// cachedRun is the persisted value: every listRow field (including the
// table-only columns, which listRow tags json:"-" and so wouldn't survive a
// direct marshal), the filter inputs, and the submit time.
type cachedRun struct {
	RunID         string       `json:"run_id"`
	RunName       string       `json:"run_name"`
	User          string       `json:"user"`
	Status        string       `json:"status"`
	StartedAt     *string      `json:"started_at"`
	IsSweep       bool         `json:"is_sweep"`
	Experiment    string       `json:"experiment"`
	Duration      string       `json:"duration"`
	MLflowURL     string       `json:"mlflow_url"`
	MLflowLabel   string       `json:"mlflow_label"`
	RunURL        string       `json:"run_url"`
	ExperimentURL string       `json:"experiment_url"`
	Accelerators  string       `json:"accelerators"`
	TaskRunID     int64        `json:"task_run_id"`
	Fields        filterFields `json:"filter_fields"`
	SubmitTimeMs  int64        `json:"submit_time_ms"`
}

func (c cachedRun) toRow() listRow {
	return listRow{
		RunID: c.RunID, RunName: c.RunName, User: c.User, Status: c.Status,
		StartedAt: c.StartedAt, IsSweep: c.IsSweep, Experiment: c.Experiment,
		Duration: c.Duration, MLflowURL: c.MLflowURL, MLflowLabel: c.MLflowLabel,
		RunURL: c.RunURL, ExperimentURL: c.ExperimentURL, Accelerators: c.Accelerators,
	}
}

func cachedRunFromRow(r listRow, taskRunID int64, fields filterFields, submitTimeMs int64) cachedRun {
	return cachedRun{
		RunID: r.RunID, RunName: r.RunName, User: r.User, Status: r.Status,
		StartedAt: r.StartedAt, IsSweep: r.IsSweep, Experiment: r.Experiment,
		Duration: r.Duration, MLflowURL: r.MLflowURL, MLflowLabel: r.MLflowLabel,
		RunURL: r.RunURL, ExperimentURL: r.ExperimentURL, Accelerators: r.Accelerators,
		TaskRunID: taskRunID, Fields: fields, SubmitTimeMs: submitTimeMs,
	}
}

// newListCache builds the cache for the index path. It fails open, so a nil
// return (or any cache error) just means every run is hydrated from the API.
func newListCache(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, listCacheComponent, listCacheTTL, nil)
}

// cachedRow returns the cached row, filter fields, and task run id for a run.
// Entries written before task ids were cached are treated as misses.
func cachedRow(ctx context.Context, c *cache.Cache, host string, runID int64) (listRow, filterFields, int64, bool) {
	entry, ok := cache.Get[cachedRun](ctx, c, listCacheKey{Host: host, RunID: runID})
	if !ok || entry.TaskRunID == 0 {
		return listRow{}, filterFields{}, 0, false
	}
	return entry.toRow(), entry.Fields, entry.TaskRunID, true
}

// putRow caches a terminal run's base row and filter fields under its submit time.
func putRow(ctx context.Context, c *cache.Cache, host string, runID, taskRunID, submitTimeMs int64, row listRow, fields filterFields) {
	cache.Put(ctx, c, listCacheKey{Host: host, RunID: runID}, cachedRunFromRow(row, taskRunID, fields, submitTimeMs))
}

// updateCachedRow replaces the display row while preserving cache metadata.
func updateCachedRow(ctx context.Context, c *cache.Cache, host string, runID int64, row listRow) {
	key := listCacheKey{Host: host, RunID: runID}
	entry, ok := cache.Get[cachedRun](ctx, c, key)
	if !ok {
		return
	}
	cache.Put(ctx, c, key, cachedRunFromRow(row, entry.TaskRunID, entry.Fields, entry.SubmitTimeMs))
}
