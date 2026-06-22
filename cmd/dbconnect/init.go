package dbconnect

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	libsdbconnect "github.com/databricks/cli/libs/dbconnect"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

const (
	// defaultConstraintBaseURL is the default URL for the constraint source.
	defaultConstraintBaseURL = "https://raw.githubusercontent.com/pietern/databricks-environments/main"

	// envConstraintSource is the environment variable for overriding the constraint source URL.
	envConstraintSource = "DATABRICKS_DBCONNECT_CONSTRAINT_SOURCE"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a fresh pyproject.toml and provision a matched .venv",
	}
	cmd.PreRunE = root.MustWorkspaceClient
	addTargetFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPipeline(cmd, libsdbconnect.ModeInit)
	}
	return cmd
}

// addTargetFlags adds the shared target flags to a command.
func addTargetFlags(cmd *cobra.Command) {
	cmd.Flags().String("cluster", "", "cluster ID to use as the compute target")
	cmd.Flags().String("serverless", "", "serverless version to use as the compute target (e.g. v4)")
	cmd.Flags().String("job", "", "job ID to use as the compute target")
	cmd.Flags().Bool("check", false, "compute the plan without writing files or provisioning")
	cmd.Flags().String("constraint-source", "", "URL for the constraint source (overrides "+envConstraintSource+")")
	// Hide constraint-source from casual --help output; it is a power-user escape hatch.
	_ = cmd.Flags().MarkHidden("constraint-source")
	cmd.MarkFlagsMutuallyExclusive("cluster", "serverless", "job")
}

// runPipeline builds and runs the dbconnect Pipeline for the given mode.
func runPipeline(cmd *cobra.Command, mode libsdbconnect.Mode) error {
	ctx := cmd.Context()

	cluster, _ := cmd.Flags().GetString("cluster")
	serverless, _ := cmd.Flags().GetString("serverless")
	job, _ := cmd.Flags().GetString("job")
	check, _ := cmd.Flags().GetBool("check")
	constraintSource, _ := cmd.Flags().GetString("constraint-source")

	targetFlags := libsdbconnect.TargetFlags{
		Cluster:    cluster,
		Serverless: serverless,
		Job:        job,
	}
	if err := libsdbconnect.ValidateTargetFlags(targetFlags); err != nil {
		return err
	}

	// Resolve constraint base URL: flag → env var → default constant.
	constraintBaseURL := resolveConstraintBaseURL(ctx, constraintSource)

	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheDir, "databricks", "dbconnect")

	bt := bundleTarget(cmd)

	w := cmdctx.WorkspaceClient(ctx)
	p := &libsdbconnect.Pipeline{
		Mode:              mode,
		Check:             check,
		ProjectDir:        projectDir,
		ConstraintBaseURL: constraintBaseURL,
		CacheDir:          cacheDir,
		Flags:             targetFlags,
		Compute:           sdkCompute{w: w},
		Bundle:            bt,
		PM:                libsdbconnect.NewUvManager(),
	}

	res, pipelineErr := p.Run(ctx)
	return renderResult(cmd, ctx, res, pipelineErr)
}

// resolveConstraintBaseURL returns the constraint base URL using ordered precedence:
// flag → env var → default constant.
func resolveConstraintBaseURL(ctx context.Context, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v, ok := env.Lookup(ctx, envConstraintSource); ok {
		return v
	}
	return defaultConstraintBaseURL
}

// bundleTarget reads the active bundle (if any) and maps its compute configuration
// to a libsdbconnect.BundleTarget.
//
// Only the top-level bundle.cluster_id field is consulted here; serverless is not
// recorded in the bundle config, so Selected=true is set only when a cluster ID is
// present. If the bundle is absent or has no cluster_id, Selected=false is returned
// so the pipeline falls through to requiring an explicit flag.
//
// TODO: extend once bundle config exposes a serverless field at the bundle level.
func bundleTarget(cmd *cobra.Command) libsdbconnect.BundleTarget {
	b := root.TryConfigureBundle(cmd)
	if b == nil {
		return libsdbconnect.BundleTarget{Selected: false}
	}
	clusterID := b.Config.Bundle.ClusterId
	if clusterID == "" {
		return libsdbconnect.BundleTarget{Selected: false}
	}
	return libsdbconnect.BundleTarget{
		ClusterID: clusterID,
		Selected:  true,
	}
}
