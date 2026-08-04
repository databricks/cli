package environments

import (
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	libslocalenv "github.com/databricks/cli/libs/localenv"
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
	}
	// The target is selected via flags; reject stray positional args rather than
	// silently ignoring them.
	cmd.Args = cobra.NoArgs
	cmd.PreRunE = root.MustWorkspaceClient
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

// runPipeline builds and runs the setup-local Pipeline.
func runPipeline(cmd *cobra.Command) error {
	ctx := cmd.Context()

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
	b := root.TryConfigureBundle(cmd)
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
