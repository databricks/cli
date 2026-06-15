package aircmd

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// maxListScan bounds how many runs `air list` inspects while looking for AIR
// runs that match the filters. The runs/list API returns runs of every kind for
// every user, so a workspace with a large run history could otherwise make the
// command page indefinitely. This mirrors the Python CLI's pagination cap.
const maxListScan = 2000

// mlflowFetchConcurrency bounds the parallel runs/get-output lookups used to
// build MLflow links for the table.
const mlflowFetchConcurrency = 8

// listData is the payload printed by `air list`.
type listData struct {
	Rows []listRow `json:"runs"`
}

// listRow is one run in the list. The json-tagged fields form the
// machine-readable output; fields tagged `json:"-"` are shown only in the
// human-readable table.
type listRow struct {
	RunID     string  `json:"run_id"`
	RunName   string  `json:"run_name"`
	User      string  `json:"user"`
	Status    string  `json:"status"`
	StartedAt *string `json:"started_at"`
	IsSweep   bool    `json:"is_sweep"`

	// Experiment, Duration, MLflowURL and Accelerators are table-only columns,
	// omitted from JSON to match `air list --json`.
	Experiment   string `json:"-"`
	Duration     string `json:"-"`
	MLflowURL    string `json:"-"`
	Accelerators string `json:"-"`
}

// listedRun pairs a rendered row with the run it came from, so MLflow links can
// be fetched for the row after the run has been filtered in.
type listedRun struct {
	row listRow
	run *jobs.Run
}

// listQuery holds the resolved inputs to listAirRuns.
type listQuery struct {
	activeOnly bool
	userFilter string
	filters    listFilters
	limit      int
}

func newListCommand() *cobra.Command {
	var (
		limit    int
		active   bool
		allUsers bool
		filters  []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Args:  root.NoArgs,
		Short: "List recent runs",
		Annotations: map[string]string{
			"template": listTemplate,
		},
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&active, "active", false, "Show only active runs")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show runs from all users")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter runs, e.g. experiment=foo* (repeatable)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if limit <= 0 {
			return fmt.Errorf("invalid --limit %d: must be a positive integer", limit)
		}

		f, err := parseListFilters(filters)
		if err != nil {
			return err
		}

		// An explicit user= filter wins; otherwise default to the current user
		// unless --all-users is set. The runs/list API has no user field, so the
		// creator is matched while scanning.
		userFilter := f.User
		if userFilter == "" && !allUsers {
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return fmt.Errorf("failed to resolve current user: %w", err)
			}
			userFilter = me.UserName
		}

		entries, err := listAirRuns(ctx, w, listQuery{
			activeOnly: active,
			userFilter: userFilter,
			filters:    f,
			limit:      limit,
		})
		if err != nil {
			return err
		}

		// MLflow links appear only in the text table, so the extra lookups are
		// skipped for JSON output (which omits them, matching `air list --json`).
		if root.OutputType(cmd) == flags.OutputText {
			setMLflowLinks(ctx, w, entries)
		}

		rows := make([]listRow, len(entries))
		for i, e := range entries {
			rows[i] = e.row
		}
		return renderEnvelope(ctx, listData{Rows: rows})
	}

	return cmd
}

// listAirRuns pages through runs/list, keeping the AIR runs that match the user
// and filters, up to the requested limit. It stops once it has enough matches
// or has scanned maxListScan runs, whichever comes first.
func listAirRuns(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) ([]listedRun, error) {
	// Limit is intentionally left unset: it caps total runs returned by the API
	// (of any kind, any user), not AIR matches, so the result is filtered and
	// truncated client-side instead.
	it := w.Jobs.ListRuns(ctx, jobs.ListRunsRequest{
		ActiveOnly:  q.activeOnly,
		ExpandTasks: true,
		RunType:     jobs.RunTypeSubmitRun,
	})

	var entries []listedRun
	scanned := 0
	for it.HasNext(ctx) && scanned < maxListScan {
		base, err := it.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list runs: %w", err)
		}
		scanned++

		if !isAirRun(&base) {
			continue
		}
		if q.userFilter != "" && base.CreatorUserName != q.userFilter {
			continue
		}
		run := baseRunToRun(&base)
		if !q.filters.matches(run) {
			continue
		}

		entries = append(entries, listedRun{row: buildListRow(run), run: run})
		if len(entries) >= q.limit {
			break
		}
	}

	if scanned >= maxListScan {
		log.Warnf(ctx, "air list: stopped after scanning %d runs; results may be incomplete", maxListScan)
	}

	return entries, nil
}

// setMLflowLinks fills in each row's MLflow link in parallel. It is best-effort:
// a row whose link can't be built keeps its placeholder, matching mlflowURL's
// own non-fatal behavior.
func setMLflowLinks(ctx context.Context, w *databricks.WorkspaceClient, entries []listedRun) {
	var g errgroup.Group
	g.SetLimit(mlflowFetchConcurrency)
	for i := range entries {
		g.Go(func() error {
			if ids := mlflowIDs(ctx, w, entries[i].run); ids != nil {
				entries[i].row.MLflowURL = mlflowLogsURL(w.Config.Host, ids)
			}
			return nil
		})
	}
	// mlflowIDs never returns an error (it logs and yields nil), so Wait can't fail.
	_ = g.Wait()
}

// baseRunToRun adapts a BaseRun (returned by runs/list) to a Run so the shared
// format helpers, which take *jobs.Run, can be reused without change. Only the
// fields those helpers read are copied.
func baseRunToRun(b *jobs.BaseRun) *jobs.Run {
	return &jobs.Run{
		RunId:           b.RunId,
		RunName:         b.RunName,
		StartTime:       b.StartTime,
		EndTime:         b.EndTime,
		RunDuration:     b.RunDuration,
		State:           b.State,
		CreatorUserName: b.CreatorUserName,
		RunPageUrl:      b.RunPageUrl,
		Tasks:           b.Tasks,
	}
}

// genAITask returns the GenAI compute task for a run task, unwrapping a foreach
// sweep when present, or nil if the task isn't an AI runtime task.
func genAITask(task jobs.RunTask) *jobs.GenAiComputeTask {
	if task.GenAiComputeTask != nil {
		return task.GenAiComputeTask
	}
	if task.ForEachTask != nil {
		return task.ForEachTask.Task.GenAiComputeTask
	}
	return nil
}

// isAirRun reports whether a run is an AI runtime training workload. Such runs
// carry a GenAI compute task with a training script, either directly or nested
// inside a foreach sweep.
func isAirRun(run *jobs.BaseRun) bool {
	if len(run.Tasks) == 0 {
		return false
	}
	t := genAITask(run.Tasks[0])
	return t != nil && t.TrainingScriptPath != ""
}

// tasksAreSweep reports whether a run's first task fans out into iterations.
func tasksAreSweep(tasks []jobs.RunTask) bool {
	return len(tasks) > 0 && tasks[0].ForEachTask != nil
}

// runCompute returns the GPU type and count from a run's first GenAI task, or
// ("", 0) when the run has no GPU compute attached. It reads the same field as
// the accelerators helper, so the accelerator filter and the displayed
// Accelerators column stay consistent.
func runCompute(run *jobs.Run) (string, int) {
	if len(run.Tasks) == 0 {
		return "", 0
	}
	t := run.Tasks[0].GenAiComputeTask
	if t == nil || t.Compute == nil {
		return "", 0
	}
	return t.Compute.GpuType, t.Compute.NumGpus
}
