package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/tmpdms"
	"github.com/spf13/cobra"
)

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources deployed by this bundle",
		Long: `Summarize resources deployed by this bundle with their workspace URLs.
Useful after deployment to see what was created and where to find it.`,
		Args: root.NoArgs,
	}

	var forcePull bool
	var includeLocations bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	cmd.Flags().MarkHidden("include-locations")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			ReadState:        true,
			AlwaysPull:       forcePull,
			IncludeLocations: includeLocations,
			InitIDs:          true,
		})
		if err != nil {
			return err
		}
		err = showSummary(cmd, b)
		if err != nil {
			return err
		}

		if logdiag.HasError(cmd.Context()) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}

func showSummary(cmd *cobra.Command, b *bundle.Bundle) error {
	ctx := cmd.Context()

	// When the deployment metadata service is in use, surface the bundle's
	// deployment_id and current version_id so callers (e.g. the workspace UI)
	// can link to the deployment metadata. Both are empty otherwise.
	versionID, err := currentDeploymentVersion(ctx, b)
	if err != nil {
		return err
	}

	switch root.OutputType(cmd) {
	case flags.OutputText:
		return render.RenderSummary(ctx, cmd.OutOrStdout(), b, b.DeploymentID, versionID)
	case flags.OutputJSON:
		return renderSummaryJSON(cmd, b, versionID)
	}
	return nil
}

// currentDeploymentVersion returns the latest version_id recorded for the
// bundle's deployment, or "" when the deployment metadata service is not in use.
func currentDeploymentVersion(ctx context.Context, b *bundle.Bundle) (string, error) {
	if b.DeploymentID == "" {
		return "", nil
	}
	svc, err := tmpdms.NewDeploymentMetadataAPI(b.WorkspaceClient(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to create metadata service client: %w", err)
	}
	dep, err := svc.GetDeployment(ctx, tmpdms.GetDeploymentRequest{DeploymentID: b.DeploymentID})
	if err != nil {
		return "", fmt.Errorf("failed to get deployment: %w", err)
	}
	return dep.LastVersionID, nil
}

// renderSummaryJSON marshals the bundle configuration and, when the deployment
// metadata service is in use, adds a top-level "deployment" object carrying the
// deployment_id and version_id.
func renderSummaryJSON(cmd *cobra.Command, b *bundle.Bundle, versionID string) error {
	value := b.Config.Value().AsAny()
	if b.DeploymentID != "" {
		// The config root is always a mapping for a loaded bundle.
		value.(map[string]any)["deployment"] = map[string]string{
			"deployment_id": b.DeploymentID,
			"version_id":    versionID,
		}
	}
	buf, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = out.Write(buf)
	_, _ = out.Write([]byte{'\n'})
	return nil
}
