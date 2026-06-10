// Copied to cmd/pipelines/version.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package version

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/spf13/cobra"
)

// updateCheckTemplate renders the version command's text output, including the
// outcome of the update check.
const updateCheckTemplate = `Databricks CLI v{{.CurrentVersion}}
{{if .DevelopmentBuild -}}
This is a development build; skipping the update check.
{{- else if .CheckFailed -}}
Could not reach GitHub to check for a newer version. See https://github.com/databricks/cli/releases for the latest release.
{{- else if .UpdateAvailable -}}
{{yellow "A new version is available"}}: {{.LatestVersion}}
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
		Short: "Show the CLI version and check whether a newer version is available",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// JSON output keeps the build-info shape scripts already parse and
		// stays offline. The update check applies to the human-facing text
		// mode only; the --version flag (handled by cobra) stays lightweight.
		if root.OutputType(cmd) == flags.OutputJSON {
			return cmdio.Render(ctx, build.GetInfo())
		}
		return cmdio.RenderWithTemplate(ctx, versioncheck.Check(ctx), "", updateCheckTemplate)
	}

	return cmd
}
