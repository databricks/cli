package aircmd

import (
	"context"
	"slices"
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
	listCacheComponent = "air-list-runs-v2"
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
	RunID           string       `json:"run_id"`
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
		RunID: c.RunID, RunName: c.RunName, User: c.User, Status: c.Status,
		StartedAt: c.StartedAt, IsSweep: c.IsSweep, ExperimentName: c.ExperimentName,
		DurationSeconds: c.DurationSeconds, AcceleratorType: c.AcceleratorType, NumAccelerators: c.NumAccelerators, DashboardURL: c.DashboardURL, Experiment: c.Experiment,
		Duration: c.Duration, MLflowURL: c.MLflowURL, MLflowLabel: c.MLflowLabel,
		RunURL: c.RunURL, ExperimentURL: c.ExperimentURL, Accelerators: c.Accelerators,
	}
}

func cachedRunFromRow(r listRow, tasks []jobs.RunTask, fields filterFields, submitTimeMs int64) cachedRun {
	return cachedRun{
		RunID: r.RunID, RunName: r.RunName, User: r.User, Status: r.Status,
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

// cachedRow returns the cached row, filter fields, and task runs for a run.
// Entries written before task runs were cached are treated as misses.
func cachedRow(ctx context.Context, c *cache.Cache, host string, runID int64) (listRow, filterFields, []jobs.RunTask, bool) {
	entry, ok := cache.Get[cachedRun](ctx, c, listCacheKey{Host: host, RunID: runID})
	if !ok || len(entry.Tasks) == 0 {
		return listRow{}, filterFields{}, nil, false
	}
	return entry.toRow(), entry.Fields, entry.Tasks, true
}

// putRow caches a terminal run's base row and filter fields under its submit time.
func putRow(ctx context.Context, c *cache.Cache, host string, runID int64, tasks []jobs.RunTask, submitTimeMs int64, row listRow, fields filterFields) {
	cache.Put(ctx, c, listCacheKey{Host: host, RunID: runID}, cachedRunFromRow(row, tasks, fields, submitTimeMs))
}

// The Jobs scan builds its rows from runs/list on every call, so it caches only
// the MLflow links of terminal runs, which cost a get-output and an MLflow
// request per row. It uses its own component so its entries never stand in for
// the index path's full rows.
const listScanMLflowCacheComponent = "air-list-scan-mlflow-v1"

// cachedMLflowLink is a terminal run's MLflow link fields, with the task runs
// they were resolved from.
type cachedMLflowLink struct {
	MLflowURL     string         `json:"mlflow_url"`
	MLflowLabel   string         `json:"mlflow_label"`
	ExperimentURL string         `json:"experiment_url"`
	Tasks         []jobs.RunTask `json:"tasks"`
}

func newScanMLflowCache(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, listScanMLflowCacheComponent, listCacheTTL, nil)
}

// applyCachedMLflowLinks fills terminal entries' MLflow fields from the cache.
// An entry is used only while the run's task runs and attempts still match the
// ones it was resolved from, so a run repaired after it was cached is refetched.
func applyCachedMLflowLinks(ctx context.Context, c *cache.Cache, host string, entries []listedRun) {
	for i := range entries {
		if !entries[i].terminal {
			continue
		}
		link, ok := cache.Get[cachedMLflowLink](ctx, c, listCacheKey{Host: host, RunID: entries[i].runID})
		if !ok || !sameTaskRuns(link.Tasks, entries[i].tasks) {
			continue
		}
		entries[i].row.MLflowURL = link.MLflowURL
		entries[i].row.MLflowLabel = link.MLflowLabel
		entries[i].row.ExperimentURL = link.ExperimentURL
	}
}

// putMLflowLinks caches terminal entries that have an MLflow link.
func putMLflowLinks(ctx context.Context, c *cache.Cache, host string, entries []listedRun) {
	for _, e := range entries {
		if !e.terminal || e.row.MLflowURL == "" || e.row.MLflowURL == "-" {
			continue
		}
		cache.Put(ctx, c, listCacheKey{Host: host, RunID: e.runID}, cachedMLflowLink{
			MLflowURL:     e.row.MLflowURL,
			MLflowLabel:   e.row.MLflowLabel,
			ExperimentURL: e.row.ExperimentURL,
			Tasks:         e.tasks,
		})
	}
}

// sameTaskRuns reports whether two retryTasks lists name the same task runs and
// attempts.
func sameTaskRuns(a, b []jobs.RunTask) bool {
	return slices.EqualFunc(a, b, func(x, y jobs.RunTask) bool {
		return x.RunId == y.RunId && x.TaskKey == y.TaskKey && x.AttemptNumber == y.AttemptNumber
	})
}

// updateCachedRow replaces the display row while preserving cache metadata.
func updateCachedRow(ctx context.Context, c *cache.Cache, host string, runID int64, row listRow) {
	key := listCacheKey{Host: host, RunID: runID}
	entry, ok := cache.Get[cachedRun](ctx, c, key)
	if !ok {
		return
	}
	cache.Put(ctx, c, key, cachedRunFromRow(row, entry.Tasks, entry.Fields, entry.SubmitTimeMs))
}
