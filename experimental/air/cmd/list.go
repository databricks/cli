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
)

// maxListPages caps how many pages `air list runs` fetches (a safety bound);
// listPageBatch is the per-request page size.
const (
	maxListPages  = 50
	listPageBatch = 100
)

// listData is the payload printed by `air list runs`.
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
	// omitted from JSON to match `air list runs --json`.
	Experiment   string `json:"-"`
	Duration     string `json:"-"`
	MLflowURL    string `json:"-"`
	Accelerators string `json:"-"`
}

// listQuery holds the resolved inputs to listAirRuns.
type listQuery struct {
	activeOnly bool
	allUsers   bool
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
		Short: "List your recent runs (active and completed) for the current profile",
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of runs to show (default: all)")
	cmd.Flags().BoolVar(&active, "active", false, "Show only active runs")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show runs from all users")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter runs, e.g. experiment=foo* (repeatable)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		// limit 0 (unset) means "all"; an explicit value must be positive.
		if cmd.Flags().Changed("limit") && limit <= 0 {
			return fmt.Errorf("invalid --limit %d: must be a positive integer", limit)
		}

		f, err := parseListFilters(filters)
		if err != nil {
			return err
		}

		// An explicit user= filter wins; otherwise default to the current user
		// unless --all-users is set. The resolved name is sent to the server as
		// creator_name, which scopes the listing.
		userFilter := f.User
		if userFilter == "" && !allUsers {
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return fmt.Errorf("failed to resolve current user: %w", err)
			}
			userFilter = me.UserName
		}

		rows, err := listAirRuns(ctx, w, listQuery{
			activeOnly: active,
			allUsers:   allUsers,
			userFilter: userFilter,
			filters:    f,
			limit:      limit,
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

// listAirRuns pages through ListTrainingWorkflows, applies the client-side
// --filter keys, and accumulates matches up to q.limit (0 = all). It stops at the
// limit, when the server runs out of pages, or at maxListPages.
func listAirRuns(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) ([]listRow, error) {
	// Fetch in bounded batches and follow the page token, so "all" doesn't ask
	// the server for an unbounded page.
	pageSize := q.limit
	if pageSize <= 0 || pageSize > listPageBatch {
		pageSize = listPageBatch
	}
	query := map[string]any{
		"active_only":       q.activeOnly,
		"include_all_users": q.allUsers,
		"page_size":         pageSize,
	}
	if q.userFilter != "" {
		query["creator_name"] = q.userFilter
	}

	var rows []listRow
	for range maxListPages {
		resp, err := listTrainingWorkflows(ctx, w, query)
		if err != nil {
			return nil, err
		}

		for i := range resp.TrainingWorkflows {
			wf := &resp.TrainingWorkflows[i]
			if !q.filters.matches(wf) {
				continue
			}
			rows = append(rows, buildListRow(wf, w.Config.Host))
			if q.limit > 0 && len(rows) >= q.limit {
				return rows, nil
			}
		}

		if resp.NextPageToken == "" {
			return rows, nil
		}
		query["page_token"] = resp.NextPageToken
	}

	log.Warnf(ctx, "air list: stopped after %d pages; results may be incomplete", maxListPages)
	return rows, nil
}
