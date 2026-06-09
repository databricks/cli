// Copied to cmd/pipelines/version.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package version

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/spf13/cobra"
)

// updateCheckTemplate renders an update check in text mode. JSON output is
// rendered directly from the versioncheck.Result struct by cmdio.
const updateCheckTemplate = `Databricks CLI v{{.CurrentVersion}}
{{if .DevelopmentBuild -}}
This is a development build; skipping the update check.
{{- else if .CheckFailed -}}
Could not reach GitHub to check for a newer version. See https://github.com/databricks/cli/releases for the latest release.
{{- else if .UpdateAvailable -}}
{{yellow "A new version is available"}}: {{.LatestVersion}} (you have {{.CurrentVersion}}).
{{if .UpgradeCommand -}}
To upgrade, run:
  {{.UpgradeCommand}}
{{- else -}}
Download the latest release: https://github.com/databricks/cli/releases
{{- end}}
{{- else -}}
{{green "You're on the latest version."}}
{{- end}}
`

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Args:  root.NoArgs,
		Short: "Retrieve information about the current version of this CLI",
		Annotations: map[string]string{
			"template": "Databricks CLI v{{.Version}}\n",
		},
	}

	var check bool
	cmd.Flags().BoolVar(&check, "check", false, "Check whether a newer version of the CLI is available")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if check {
			return runUpdateCheck(cmd)
		}
		return cmdio.Render(cmd.Context(), build.GetInfo())
	}

	cmd.AddCommand(newCheckCommand())

	return cmd
}

func newCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Args:  root.NoArgs,
		Short: "Check whether a newer version of the CLI is available",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runUpdateCheck(cmd)
	}
	return cmd
}

func runUpdateCheck(cmd *cobra.Command) error {
	ctx := cmd.Context()
	result, err := versioncheck.Check(ctx)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	return cmdio.RenderWithTemplate(ctx, result, "", updateCheckTemplate)
}
