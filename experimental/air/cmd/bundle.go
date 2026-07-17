package aircmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// A bundle is the persistent Jobs resource `air run` deploys: one job per
// experiment, holding an ai_runtime_task. Runs are executions of a bundle. The
// `bundle` noun manages those durable jobs (list / get / delete), distinct from
// `run`, which manages individual executions.

// bundleScanConcurrency bounds the parallel runs/list calls when checking each
// bundle for active runs.
const bundleScanConcurrency = 16

// airBundle is one AIR job, projected for display.
type airBundle struct {
	JobID   string `json:"job_id"`
	Name    string `json:"name"`
	User    string `json:"user"`
	Created *int64 `json:"created_time_ms,omitempty"`
}

// isAirJob reports whether a job's first task is an ai_runtime_task (or a
// foreach sweep of one), or a legacy gen_ai_compute_task with a training script.
// jobs/list analogue of isAirRun.
func isAirJob(j jobs.BaseJob) bool {
	if j.Settings == nil || len(j.Settings.Tasks) == 0 {
		return false
	}
	t := j.Settings.Tasks[0]
	if t.ForEachTask != nil {
		t = t.ForEachTask.Task
	}
	return t.AiRuntimeTask != nil ||
		(t.GenAiComputeTask != nil && t.GenAiComputeTask.TrainingScriptPath != "")
}

// listAirBundles enumerates AIR jobs via jobs/list, newest-first. userFilter, when
// set, keeps only that creator's bundles. truncated reports the maxListScan cap
// was hit.
func listAirBundles(ctx context.Context, w *databricks.WorkspaceClient, userFilter string) (bundles []airBundle, truncated bool, err error) {
	iter := w.Jobs.List(ctx, jobs.ListJobsRequest{ExpandTasks: true, Limit: jobsPageLimit})
	inspected := 0
	for iter.HasNext(ctx) && inspected < maxListScan {
		job, err := iter.Next(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("failed to list jobs: %w", err)
		}
		inspected++
		if !isAirJob(job) {
			continue
		}
		if userFilter != "" && job.CreatorUserName != userFilter {
			continue
		}
		b := airBundle{
			JobID: strconv.FormatInt(job.JobId, 10),
			Name:  job.Settings.Name,
			User:  job.CreatorUserName,
		}
		if job.CreatedTime > 0 {
			created := job.CreatedTime
			b.Created = &created
		}
		bundles = append(bundles, b)
	}
	return bundles, inspected >= maxListScan && iter.HasNext(ctx), nil
}

// bundleHasActiveRuns reports whether a bundle (job) has any run that is not yet
// terminal.
func bundleHasActiveRuns(ctx context.Context, w *databricks.WorkspaceClient, jobID int64) (bool, error) {
	iter := w.Jobs.ListRuns(ctx, jobs.ListRunsRequest{JobId: jobID, ActiveOnly: true, Limit: 1})
	if !iter.HasNext(ctx) {
		return false, nil
	}
	if _, err := iter.Next(ctx); err != nil {
		return false, fmt.Errorf("failed to list runs for job %d: %w", jobID, err)
	}
	return true, nil
}

// --- list bundles ---

type listBundlesData struct {
	Bundles []airBundle `json:"bundles"`
}

func newListBundlesCommand() *cobra.Command {
	var allUsers bool

	cmd := &cobra.Command{
		Use:   "bundles",
		Args:  root.NoArgs,
		Short: "List the persistent AIR jobs (bundles) `air run` has deployed",
	}
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show bundles from all users")
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		userFilter, err := selfUserFilter(ctx, w, allUsers)
		if err != nil {
			return err
		}

		bundles, truncated, err := listAirBundles(ctx, w, userFilter)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
		}
		if truncated {
			cmdio.LogString(ctx, fmt.Sprintf("air list bundles: stopped after inspecting %d jobs; results may be incomplete", maxListScan))
		}

		if root.OutputType(cmd) == flags.OutputJSON {
			return renderEnvelope(ctx, listBundlesData{Bundles: bundles})
		}
		renderBundlesText(cmd, bundles)
		return nil
	}
	return cmd
}

// --- get bundle ---

type getBundleData struct {
	JobID        string    `json:"job_id"`
	Name         string    `json:"name"`
	User         string    `json:"user"`
	DashboardURL string    `json:"dashboard_url"`
	Runs         []listRow `json:"runs"`
}

func newGetBundleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle NAME",
		Args:  root.ExactArgs(1),
		Short: "Show a bundle (persistent AIR job) and its recent runs",
	}
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		bundle, err := resolveBundle(ctx, w, args[0])
		if err != nil {
			return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false, err)
		}
		jobID, _ := strconv.ParseInt(bundle.JobID, 10, 64)

		rows, err := runsForBundle(ctx, w, jobID)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
		}

		data := getBundleData{
			JobID:        bundle.JobID,
			Name:         bundle.Name,
			User:         bundle.User,
			DashboardURL: strings.TrimRight(w.Config.Host, "/") + "/jobs/" + bundle.JobID,
			Runs:         rows,
		}
		if root.OutputType(cmd) == flags.OutputJSON {
			return renderEnvelope(ctx, data)
		}
		renderBundleDetail(cmd, data)
		return nil
	}
	return cmd
}

// --- delete bundle ---

type deleteBundleData struct {
	Deleted []string        `json:"deleted"`
	All     bool            `json:"all,omitempty"`
	Failed  []cancelFailure `json:"failed,omitempty"`
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource (bundle)",
	}
	cmd.AddCommand(newDeleteBundleCommand())
	return cmd
}

func newDeleteBundleCommand() *cobra.Command {
	var (
		all bool
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "bundle [NAME...]",
		Short: "Delete persistent AIR jobs (bundles) `air run` deployed",
		Long: `Delete one or more bundles (persistent AIR jobs) by name, or all of your
bundles with --all. A bundle with running runs is confirmed before deletion.`,
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Delete all of your bundles")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		switch {
		case all && len(args) > 0:
			return &root.InvalidArgsError{Command: cmd, Message: "cannot combine NAME arguments with --all"}
		case !all && len(args) == 0:
			return &root.InvalidArgsError{Command: cmd, Message: "provide at least one bundle NAME, or use --all"}
		}
		return nil
	}
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		jsonOut := root.OutputType(cmd) == flags.OutputJSON

		targets, err := deleteTargets(ctx, w, args, all)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
		}
		if len(targets) == 0 {
			if jsonOut {
				return renderEnvelope(ctx, deleteBundleData{Deleted: []string{}, All: all})
			}
			cmdio.LogString(ctx, "No bundles found.")
			return nil
		}

		// A bundle with running runs needs explicit confirmation; --all always
		// confirms the whole set. Non-interactive callers pass -y.
		active, err := bundlesWithActiveRuns(ctx, w, targets)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
		}
		if !yes && (all || len(active) > 0) {
			confirmed, err := confirmDelete(ctx, targets, active, all)
			if err != nil {
				return err
			}
			if !confirmed {
				cmdio.LogString(ctx, "Deletion aborted.")
				return root.ErrAlreadyPrinted
			}
		}

		data := deleteBundleData{Deleted: []string{}, All: all}
		for _, b := range targets {
			jobID, _ := strconv.ParseInt(b.JobID, 10, 64)
			if err := w.Jobs.DeleteByJobId(ctx, jobID); err != nil {
				data.Failed = append(data.Failed, cancelFailure{RunID: b.Name, Error: err.Error()})
				if !jsonOut {
					cmdio.LogString(ctx, fmt.Sprintf("Failed to delete bundle %s: %s", b.Name, err))
				}
				continue
			}
			data.Deleted = append(data.Deleted, b.Name)
			if !jsonOut {
				cmdio.LogString(ctx, "Deleted bundle "+b.Name)
			}
		}

		if jsonOut {
			if err := renderEnvelope(ctx, data); err != nil {
				return err
			}
			if len(data.Failed) > 0 {
				return root.ErrAlreadyPrinted
			}
			return nil
		}
		if len(data.Failed) > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("%d bundle(s) failed to delete.", len(data.Failed)))
			return root.ErrAlreadyPrinted
		}
		return nil
	}
	return cmd
}

// deleteTargets resolves the bundles a delete should act on: every AIR bundle
// for --all, otherwise each named bundle (erroring if any name is unknown).
func deleteTargets(ctx context.Context, w *databricks.WorkspaceClient, names []string, all bool) ([]airBundle, error) {
	me, err := currentUserName(ctx, w)
	if err != nil {
		return nil, err
	}
	bundles, _, err := listAirBundles(ctx, w, me)
	if err != nil {
		return nil, err
	}
	if all {
		return bundles, nil
	}
	byName := make(map[string][]airBundle, len(bundles))
	for _, b := range bundles {
		byName[b.Name] = append(byName[b.Name], b)
	}
	var targets []airBundle
	for _, name := range names {
		matches := byName[name]
		if len(matches) == 0 {
			return nil, fmt.Errorf("bundle %q not found", name)
		}
		targets = append(targets, matches...)
	}
	return targets, nil
}

// bundlesWithActiveRuns returns the names of bundles that have a non-terminal run.
func bundlesWithActiveRuns(ctx context.Context, w *databricks.WorkspaceClient, bundles []airBundle) ([]string, error) {
	active := make([]bool, len(bundles))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(bundleScanConcurrency)
	for i := range bundles {
		g.Go(func() error {
			jobID, _ := strconv.ParseInt(bundles[i].JobID, 10, 64)
			has, err := bundleHasActiveRuns(gctx, w, jobID)
			if err != nil {
				return err
			}
			active[i] = has
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	var names []string
	for i := range bundles {
		if active[i] {
			names = append(names, bundles[i].Name)
		}
	}
	return names, nil
}

// confirmDelete shows what will be deleted, flags any bundles with running runs,
// and prompts for confirmation.
func confirmDelete(ctx context.Context, targets []airBundle, active []string, all bool) (bool, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "About to delete %d bundle(s):\n", len(targets))
	activeSet := make(map[string]bool, len(active))
	for _, n := range active {
		activeSet[n] = true
	}
	for _, b := range targets {
		if activeSet[b.Name] {
			fmt.Fprintf(&sb, "  %s  (has running runs)\n", b.Name)
		} else {
			fmt.Fprintf(&sb, "  %s\n", b.Name)
		}
	}
	cmdio.LogString(ctx, strings.TrimRight(sb.String(), "\n"))

	prompt := "\nDelete these bundle(s)?"
	if len(active) > 0 {
		prompt = fmt.Sprintf("\n%d bundle(s) have running runs. Delete anyway?", len(active))
	}
	return cmdio.AskYesOrNo(ctx, prompt)
}

// resolveBundle finds a single AIR bundle by name for the current user.
func resolveBundle(ctx context.Context, w *databricks.WorkspaceClient, name string) (airBundle, error) {
	me, err := currentUserName(ctx, w)
	if err != nil {
		return airBundle{}, err
	}
	bundles, _, err := listAirBundles(ctx, w, me)
	if err != nil {
		return airBundle{}, err
	}
	for _, b := range bundles {
		if b.Name == name {
			return b, nil
		}
	}
	return airBundle{}, fmt.Errorf("bundle %q not found", name)
}

// runsForBundle lists a bundle's runs (a single page) as display rows.
func runsForBundle(ctx context.Context, w *databricks.WorkspaceClient, jobID int64) ([]listRow, error) {
	iter := w.Jobs.ListRuns(ctx, jobs.ListRunsRequest{JobId: jobID, ExpandTasks: true, Limit: jobsPageLimit})
	var rows []listRow
	for iter.HasNext(ctx) {
		base, err := iter.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list runs for job %d: %w", jobID, err)
		}
		rows = append(rows, buildListRow(baseRunToRun(base)))
	}
	return rows, nil
}
