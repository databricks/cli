package environments

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newSetupLocalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   libslocalenv.CommandVerb,
		Short: "Provision a local Python environment matched to a Databricks compute target",
		Long: `Provision (or update) a local Python environment matched to a Databricks compute target.

Resolves the target to an environment key, fetches the pinned Python version,
databricks-connect version, and dependency constraints published for that key,
then provisions a matched .venv with uv. A project with no pyproject.toml is
initialized from scratch; an existing pyproject.toml is merged in place (its
env-owned sections are refreshed, user-owned content is preserved).`,
		// Hidden until the environment constraints repository is publicly
		// available: the command is runnable for dogfooding but stays out of
		// help and completion until it is unveiled.
		Hidden: true,
	}
	// The target is selected via flags; reject stray positional args rather than
	// silently ignoring them.
	cmd.Args = cobra.NoArgs
	// This command resolves its own compute target and only consults the bundle as
	// an optional source of bundle.cluster_id (see bundleTarget). Skip bundle-based
	// auth configuration in the shared PreRunE so a malformed databricks.yml (e.g.
	// two targets marked default) can't fail the command before it runs; the fallback
	// bundle read in bundleTarget swallows such errors and falls through to E_NO_TARGET.
	// As a consequence auth resolves from profile/env only: the bundle's
	// workspace.host/profile no longer feed the workspace client for this command.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}
	addComputeFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPipeline(cmd)
	}
	return cmd
}

// addComputeFlags adds the shared compute and mode flags to a command.
func addComputeFlags(cmd *cobra.Command) {
	cmd.Flags().String("cluster-id", "", "cluster ID to use as the compute target")
	cmd.Flags().String("cluster-name", "", "cluster name to use as the compute target (resolved to an ID via the Clusters API)")
	cmd.Flags().String("serverless-version", "", "serverless version to use as the compute target (e.g. 5)")
	cmd.Flags().String("job-task", "", "job task to use as the compute target, as <job-id>.<task-key> (the task key is required)")
	cmd.Flags().Bool("constraints-only", false, "apply the Python version and constraints without adding the databricks-connect dependency")
	cmd.Flags().Bool("dry-run", false, "compute the plan without writing files or provisioning")
	// The mutual exclusivity of the target flags is enforced in the pipeline's
	// preflight (as E_USAGE) rather than via cmd.MarkFlagsMutuallyExclusive, so
	// the conflict is reported through the phase/JSON contract the --output json
	// consumer relies on, instead of a bare pre-RunE Cobra error.
}

// watchInterruptSignals cancels ctx on the first SIGINT (Ctrl-C) or SIGTERM (how
// a supervisor, CI timeout, or VS Code stops the child), which propagates to the
// uv subprocesses the pipeline spawns so they are reaped instead of orphaned
// mid-provision. The CLI root installs no signal handler of its own.
//
// The returned stop function uninstalls the handler and joins the goroutine; the
// caller must defer it.
//
// The handler must give the *second* signal back to the OS. signal.Notify (like
// signal.NotifyContext, which wraps it) disables the default disposition for
// SIGINT/SIGTERM for as long as the channel stays registered, so without the
// signal.Stop below a second Ctrl-C is merely buffered and dropped: the user
// would have no way to abort during the process group's SIGKILL grace window.
// Stopping the relay as soon as the first signal lands restores SIG_DFL, so a
// second signal terminates the CLI immediately. That matters more here than in
// most commands because WithProcessGroup moves uv out of the foreground process
// group, so the tty no longer delivers Ctrl-C to it directly — this handler is
// the only delivery path.
func watchInterruptSignals(ctx context.Context, cancel context.CancelFunc) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Selecting on ctx.Done() too lets the goroutine exit on the normal (no
		// signal) path rather than blocking on sigCh for the rest of the process:
		// signal.Stop unregisters the channel but never closes it.
		select {
		case <-sigCh:
			signal.Stop(sigCh)
			cancel()
		case <-ctx.Done():
		}
	}()

	return func() {
		signal.Stop(sigCh)
		// Wake the goroutine in case neither sigCh nor ctx.Done has fired.
		cancel()
		<-done
	}
}

// runPipeline builds and runs the setup-local Pipeline.
func runPipeline(cmd *cobra.Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	defer watchInterruptSignals(ctx, cancel)()

	cluster, _ := cmd.Flags().GetString("cluster-id")
	clusterName, _ := cmd.Flags().GetString("cluster-name")
	serverless, _ := cmd.Flags().GetString("serverless-version")
	jobTask, _ := cmd.Flags().GetString("job-task")
	constraintsOnly, _ := cmd.Flags().GetBool("constraints-only")
	check, _ := cmd.Flags().GetBool("dry-run")

	computeFlags := libslocalenv.ComputeFlags{
		Cluster:     cluster,
		ClusterName: clusterName,
		Serverless:  serverless,
		JobTask:     jobTask,
	}
	// Flag validation (including mutual exclusivity) happens in the pipeline's
	// preflight, so a conflict is reported as E_USAGE through the phase/JSON
	// contract rather than as a bare error here.

	mode := libslocalenv.ModeDefault
	if constraintsOnly {
		mode = libslocalenv.ModeConstraintsOnly
	}

	constraintBaseURL := libslocalenv.ConstraintBaseURL(ctx)

	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheDir, "databricks", "localenv")

	// The bundle is only a fallback: ResolveCompute consults it solely when no
	// explicit --cluster-id/--cluster-name/--serverless-version/--job-task flag is set. Skip the bundle load
	// entirely when a flag is present — it would otherwise re-run TryConfigureBundle
	// (a second full load) and re-print any bundle load-time diagnostics for nothing.
	var bt libslocalenv.BundleTarget
	if cluster == "" && clusterName == "" && serverless == "" && jobTask == "" {
		bt = bundleTarget(cmd)
	}

	w := cmdctx.WorkspaceClient(ctx)
	p := &libslocalenv.Pipeline{
		Mode:              mode,
		Check:             check,
		ProjectDir:        projectDir,
		ConstraintBaseURL: constraintBaseURL,
		CacheDir:          cacheDir,
		Flags:             computeFlags,
		Compute:           sdkCompute{w: w},
		Bundle:            bt,
		PM:                libslocalenv.NewUvManager(),
	}

	res, pipelineErr := p.Run(ctx)
	return renderResult(ctx, cmd, res, pipelineErr)
}

// bundleTarget reads the active bundle (if any) and maps its compute configuration
// to a libslocalenv.BundleTarget.
//
// Only the top-level bundle.cluster_id field is consulted here; serverless is not
// recorded in the bundle config, so Selected=true is set only when a cluster ID is
// present. If the bundle is absent or has no cluster_id, Selected=false is returned
// so the pipeline falls through to requiring an explicit flag.
//
// TODO: extend once bundle config exposes a serverless field at the bundle level.
func bundleTarget(cmd *cobra.Command) libslocalenv.BundleTarget {
	// Load the bundle in an isolated diagnostics context: the bundle is only an
	// optional source of cluster_id here, so a malformed databricks.yml must not
	// surface as a fatal command error. Any load error is logged for debugging and
	// treated as "no bundle target", so the pipeline falls through to E_NO_TARGET
	// (which tells the user to pass an explicit --cluster-id/--serverless-version/etc).
	orig := cmd.Context()
	ctx := logdiag.IsolatedContext(orig)
	// Collect (buffer) diagnostics instead of rendering them: an isolated context
	// still prints each diagnostic to stderr unless collection is on, and we want a
	// bundle load error to be silent (debug-logged) on this optional fallback path.
	logdiag.SetCollect(ctx, true)
	cmd.SetContext(ctx)
	defer cmd.SetContext(orig)
	b := root.TryConfigureBundle(cmd)
	if logdiag.HasError(ctx) {
		log.Debugf(ctx, "ignoring bundle for cluster_id fallback: %s", logdiag.GetFirstErrorSummary(ctx))
		return libslocalenv.BundleTarget{Selected: false}
	}
	if b == nil {
		return libslocalenv.BundleTarget{Selected: false}
	}
	clusterID := b.Config.Bundle.ClusterId
	if clusterID == "" {
		return libslocalenv.BundleTarget{Selected: false}
	}
	return libslocalenv.BundleTarget{
		ClusterID: clusterID,
		Selected:  true,
	}
}
