package environments

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/spf13/cobra"
)

// envConstraintSource is the environment variable that overrides the constraint
// source with a full base URL (used e.g. by tests pointing at a local server).
// When unset, the base URL is derived from the hosting repo via
// libslocalenv.RepoConstraintBaseURL (which reads its own repo env var).
const envConstraintSource = "DATABRICKS_LOCALENV_CONSTRAINT_SOURCE"

func newSyncCommand() *cobra.Command {
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
		// help and completion until it is unveiled (mirrors the pre-rename state
		// on main, where the local-env group carried this flag).
		Hidden: true,
	}
	// The target is selected via flags; reject stray positional args rather than
	// silently ignoring them.
	cmd.Args = cobra.NoArgs
	cmd.PreRunE = root.MustWorkspaceClient
	addTargetFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPipeline(cmd)
	}
	return cmd
}

// addTargetFlags adds the shared target and mode flags to a command.
func addTargetFlags(cmd *cobra.Command) {
	cmd.Flags().String("cluster", "", "cluster ID to use as the compute target")
	cmd.Flags().String("serverless", "", "serverless version to use as the compute target (e.g. v4)")
	cmd.Flags().String("job", "", "job ID to use as the compute target")
	cmd.Flags().Bool("constraints-only", false, "apply the Python version and constraints without adding the databricks-connect dependency")
	cmd.Flags().Bool("check", false, "compute the plan without writing files or provisioning")
	cmd.Flags().String("constraint-source", "", "URL for the constraint source (overrides "+envConstraintSource+")")
	// Hide constraint-source from casual --help output; it is a power-user escape hatch.
	_ = cmd.Flags().MarkHidden("constraint-source")
	cmd.MarkFlagsMutuallyExclusive("cluster", "serverless", "job")
}

// runPipeline builds and runs the local-env Pipeline.
func runPipeline(cmd *cobra.Command) error {
	ctx := cmd.Context()

	cluster, _ := cmd.Flags().GetString("cluster")
	serverless, _ := cmd.Flags().GetString("serverless")
	job, _ := cmd.Flags().GetString("job")
	constraintsOnly, _ := cmd.Flags().GetBool("constraints-only")
	check, _ := cmd.Flags().GetBool("check")
	constraintSource, _ := cmd.Flags().GetString("constraint-source")

	targetFlags := libslocalenv.TargetFlags{
		Cluster:    cluster,
		Serverless: serverless,
		Job:        job,
	}
	// ValidateTargetFlags is kept despite MarkFlagsMutuallyExclusive above:
	// it also validates the library path (no Cobra equivalent) and guards
	// non-Cobra call paths such as tests that invoke runPipeline directly.
	if err := libslocalenv.ValidateTargetFlags(targetFlags); err != nil {
		return err
	}

	mode := libslocalenv.ModeDefault
	if constraintsOnly {
		mode = libslocalenv.ModeConstraintsOnly
	}

	constraintBaseURL := resolveConstraintBaseURL(ctx, constraintSource)

	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheDir, "databricks", "localenv")

	// The bundle is only a fallback: ResolveTarget consults it solely when no
	// explicit --cluster/--serverless/--job flag is set. Skip the bundle load
	// entirely when a flag is present — it would otherwise re-run TryConfigureBundle
	// (a second full load) and re-print any bundle load-time diagnostics for nothing.
	var bt libslocalenv.BundleTarget
	if cluster == "" && serverless == "" && job == "" {
		bt = bundleTarget(cmd)
	}

	w := cmdctx.WorkspaceClient(ctx)
	p := &libslocalenv.Pipeline{
		Mode:              mode,
		Check:             check,
		ProjectDir:        projectDir,
		ConstraintBaseURL: constraintBaseURL,
		CacheDir:          cacheDir,
		Flags:             targetFlags,
		Compute:           sdkCompute{w: w},
		Bundle:            bt,
		PM:                libslocalenv.NewUvManager(),
	}

	res, pipelineErr := p.Run(ctx)
	return renderResult(ctx, cmd, res, pipelineErr)
}

// resolveConstraintBaseURL returns the constraint base URL using ordered precedence:
// an explicit --constraint-source flag, then a full-URL override from
// DATABRICKS_LOCALENV_CONSTRAINT_SOURCE, then the URL derived from the hosting repo
// (libslocalenv.RepoConstraintBaseURL). All three may be unset, in which case it
// returns "" and the pipeline reports the missing source at the fetch phase.
func resolveConstraintBaseURL(ctx context.Context, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v, ok := env.Lookup(ctx, envConstraintSource); ok && v != "" {
		return v
	}
	return libslocalenv.RepoConstraintBaseURL(ctx)
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
