package root

import (
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/liteswap"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// MustWorkspaceClientForTemplate is MustWorkspaceClient for commands that render a template. When
// template.HelpersProfileEnvVar names a profile, it also loads that profile the same way as
// --profile and makes the template read the workspace through it. It fails if the profile can't
// be loaded or doesn't have the selected profile's workspace ID, so that a template never reads
// one workspace and writes to another.
func MustWorkspaceClientForTemplate(cmd *cobra.Command, args []string) error {
	if err := MustWorkspaceClient(cmd, args); err != nil {
		return err
	}
	ctx := cmd.Context()
	name := env.Get(ctx, template.HelpersProfileEnvVar)
	if name == "" {
		return nil
	}

	cfg := &config.Config{Profile: name}
	liteswap.Apply(ctx, cfg)
	applyProfileAuthPrecedence(ctx, cfg, true)
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return fmt.Errorf("failed to load profile %q from %s: %w", name, template.HelpersProfileEnvVar, err)
	}

	helpersID := w.Config.WorkspaceID
	selectedID := cmdctx.WorkspaceClient(ctx).Config.WorkspaceID
	if helpersID == "" || helpersID == profile.WorkspaceIDNone || helpersID != selectedID {
		return fmt.Errorf("profile %q from %s must have the workspace ID of the selected profile, got %q and %q", name, template.HelpersProfileEnvVar, helpersID, selectedID)
	}

	cmd.SetContext(template.WithHelpersClient(ctx, w))
	return nil
}
