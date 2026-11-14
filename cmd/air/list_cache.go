package aircmd

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// The AiTrainingService index path caches hydrated terminal runs on disk:
// terminal runs are immutable, so once we've paid for runs/get + get-output +
// MLflow we persist the finished row and skip those round-trips next time. The
// TTL matches AICM's ~60-day retention, after which the run drops out of the
// index anyway.
const (
	listCacheComponent = "air-list-runs-v3"
	listCacheTTL       = 60 * 24 * time.Hour
)

// listCacheKey fingerprints a cached run. Host isolates workspaces (a Jobs run
// id is unique only within one), matching how libs/cache namespaces entries.
type listCacheKey struct {
	Host  string `json:"host"`
	RunID int64  `json:"run_id"`
}

// cachedRuns is the persisted value for a run: one cachedRun per AI runtime
// task (see buildListRows).
type cachedRuns struct {
	Rows []cachedRun `json:"rows"`
}

// cachedRun is one persisted list row: every listRow field (including the
// table-only columns, which listRow tags json:"-" and so wouldn't survive a
// direct marshal), the filter inputs, and the submit time.
type cachedRun struct {
	RunID           string       `json:"run_id"`
	TaskKey         string       `json:"task_key"`
	RunName         string       `json:"run_name"`
	User            string       `json:"user"`
	Status          string       `json:"status"`
	StartedAt       *string      `json:"started_at"`
	IsSweep         bool         `json:"is_sweep"`
	ExperimentName  *string      `json:"experiment_name"`
	DurationSeconds *int64       `json:"duration_seconds"`
	AcceleratorType *string      `json:"accelerator_type"`
	NumAccelerators *int         `json:"num_accelerators"`
	DashboardURL    string       `json:"dashboard_url"`
	Experiment      string       `json:"experiment"`
	Duration        string       `json:"duration"`
	MLflowURL       string       `json:"mlflow_url"`
	MLflowLabel     string       `json:"mlflow_label"`
	RunURL          string       `json:"run_url"`
	ExperimentURL   string       `json:"experiment_url"`
	Accelerators    string       `json:"accelerators"`
	Fields          filterFields `json:"filter_fields"`
	SubmitTimeMs    int64        `json:"submit_time_ms"`

	// Tasks are the run's task runs (see retryTasks), which pick its MLflow link.
	Tasks []jobs.RunTask `json:"tasks"`
}

func (c cachedRun) toRow() listRow {
	return listRow{
		RunID: c.RunID, TaskKey: c.TaskKey, RunName: c.RunName, User: c.User, Status: c.Status,
		StartedAt: c.StartedAt, IsSweep: c.IsSweep, ExperimentName: c.ExperimentName,
		DurationSeconds: c.DurationSeconds, AcceleratorType: c.AcceleratorType, NumAccelerators: c.NumAccelerators, DashboardURL: c.DashboardURL, Experiment: c.Experiment,
		Duration: c.Duration, MLflowURL: c.MLflowURL, MLflowLabel: c.MLflowLabel,
		RunURL: c.RunURL, ExperimentURL: c.ExperimentURL, Accelerators: c.Accelerators,
	}
}

func cachedRunFromRow(r listRow, tasks []jobs.RunTask, fields filterFields, submitTimeMs int64) cachedRun {
	return cachedRun{
		RunID: r.RunID, TaskKey: r.TaskKey, RunName: r.RunName, User: r.User, Status: r.Status,
		StartedAt: r.StartedAt, IsSweep: r.IsSweep, ExperimentName: r.ExperimentName,
		DurationSeconds: r.DurationSeconds, AcceleratorType: r.AcceleratorType, NumAccelerators: r.NumAccelerators, DashboardURL: r.DashboardURL, Experiment: r.Experiment,
		Duration: r.Duration, MLflowURL: r.MLflowURL, MLflowLabel: r.MLflowLabel,
		RunURL: r.RunURL, ExperimentURL: r.ExperimentURL, Accelerators: r.Accelerators,
		Tasks: tasks, Fields: fields, SubmitTimeMs: submitTimeMs,
	}
}

// newListCache builds the cache for the index path. It fails open, so a nil
// return (or any cache error) just means every run is hydrated from the API.
func newListCache(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, listCacheComponent, listCacheTTL, nil)
}

// cachedRows returns the cached rows of a run, one per AI runtime task. An
// entry with a row that lacks task runs is treated as a miss.
func cachedRows(ctx context.Context, c *cache.Cache, host string, runID int64) ([]cachedRun, bool) {
	entry, ok := cache.Get[cachedRuns](ctx, c, listCacheKey{Host: host, RunID: runID})
	if !ok || len(entry.Rows) == 0 {
		return nil, false
	}
	for _, row := range entry.Rows {
		if len(row.Tasks) == 0 {
			return nil, false
		}
	}
	return entry.Rows, true
}

// putRows caches a terminal run's rows.
func putRows(ctx context.Context, c *cache.Cache, host string, runID int64, rows []cachedRun) {
	cache.Put(ctx, c, listCacheKey{Host: host, RunID: runID}, cachedRuns{Rows: rows})
}

// updateCachedRow replaces the display row of the run's task while preserving
// cache metadata. A run or task that is not cached is left alone.
func updateCachedRow(ctx context.Context, c *cache.Cache, host string, runID int64, taskKey string, row listRow) {
	key := listCacheKey{Host: host, RunID: runID}
	entry, ok := cache.Get[cachedRuns](ctx, c, key)
	if !ok {
		return
	}
	for i, cached := range entry.Rows {
		if cached.TaskKey == taskKey {
			entry.Rows[i] = cachedRunFromRow(row, cached.Tasks, cached.Fields, cached.SubmitTimeMs)
			cache.Put(ctx, c, key, entry)
			return
		}
	}
}
