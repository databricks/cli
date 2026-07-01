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
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// maxListScan bounds how many runs `air list` inspects while looking for AIR runs
// that match the filters. runs/list returns runs of every kind, so this caps the
// work on a workspace with a large run history.
const maxListScan = 2000

// jobsPageLimit is the per-request page size for runs/list; enrichConcurrency
// bounds the parallel MLflow lookups.
const (
	jobsPageLimit     = 25
	enrichConcurrency = 8
)

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

// listedRun pairs a row with its task run id, so the MLflow link can be fetched
// after the run has been filtered in.
type listedRun struct {
	row       listRow
	taskRunID int64
}

// listQuery holds the resolved inputs to listAirRuns.
type listQuery struct {
	activeOnly  bool
	allUsers    bool
	userFilter  string
	filters     listFilters
	limit       int
	fetchMLflow bool
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
		Short: "List your recent runs (active and completed) for the current profile",
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
		// unless --all-users is set. runs/list has no creator param, so the
		// creator is matched while scanning.
		userFilter := f.User
		if userFilter == "" && !allUsers {
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return fmt.Errorf("failed to resolve current user: %w", err)
			}
			userFilter = me.UserName
		}

		rows, err := listAirRuns(ctx, w, listQuery{
			activeOnly:  active,
			allUsers:    allUsers,
			userFilter:  userFilter,
			filters:     f,
			limit:       limit,
			fetchMLflow: root.OutputType(cmd) == flags.OutputText,
		})
		if err != nil {
			return err
		}

		// JSON keeps the air envelope; text renders the table (interactive and
		// navigable in a terminal, printed once when piped).
		if root.OutputType(cmd) == flags.OutputJSON {
			return renderEnvelope(ctx, listData{Rows: rows})
		}
		return renderListText(cmd, rows)
	}

	return cmd
}

// listAirRuns pages through Jobs runs/list, keeps the AIR runs that match the
// user and filters, and stops once it has enough matches or has scanned
// maxListScan runs. Detail comes straight from runs/list (expand_tasks), so no
// per-run calls are needed.
func listAirRuns(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) ([]listRow, error) {
	query := map[string]any{
		"run_type":     "SUBMIT_RUN",
		"expand_tasks": true,
		"limit":        jobsPageLimit,
	}
	if q.activeOnly {
		query["active_only"] = true
	}

	var entries []listedRun
	scanned := 0
	done := false
	for !done && scanned < maxListScan {
		resp, err := fetchJobRunsPage(ctx, w, query)
		if err != nil {
			return nil, err
		}

		for i := range resp.Runs {
			run := &resp.Runs[i]
			scanned++

			if !isAirRun(run) {
				continue
			}
			if q.userFilter != "" && run.CreatorUserName != q.userFilter {
				continue
			}
			if !q.filters.matches(run) {
				continue
			}

			entries = append(entries, listedRun{row: buildListRow(run), taskRunID: taskRunID(run)})
			if len(entries) >= q.limit {
				done = true
				break
			}
		}

		if done || resp.NextPageToken == "" {
			break
		}
		query["page_token"] = resp.NextPageToken
	}

	if !done && scanned >= maxListScan {
		log.Warnf(ctx, "air list: stopped after scanning %d runs; results may be incomplete", maxListScan)
	}

	// MLflow links appear only in the text table, so the per-run get-output
	// lookups are skipped for JSON output (which omits the column anyway).
	if q.fetchMLflow {
		setMLflowLinks(ctx, w, entries)
	}

	rows := make([]listRow, len(entries))
	for i, e := range entries {
		rows[i] = e.row
	}
	return rows, nil
}

// setMLflowLinks fills in each row's MLflow link in parallel, best-effort: a row
// whose IDs can't be resolved keeps its "-" placeholder.
func setMLflowLinks(ctx context.Context, w *databricks.WorkspaceClient, entries []listedRun) {
	var g errgroup.Group
	g.SetLimit(enrichConcurrency)
	for i := range entries {
		g.Go(func() error {
			if ids := mlflowIDsForTask(ctx, w, entries[i].taskRunID); ids != nil {
				entries[i].row.MLflowURL = mlflowLogsURL(w.Config.Host, ids)
			}
			return nil
		})
	}
	// mlflowIDsForTask never returns an error (it logs and yields nil), so Wait can't fail.
	_ = g.Wait()
}
