package ai

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// versionInfo combines the local CLI version with the result of a live workspace
// probe, so a successful render proves profile resolution, authentication, and output rendering all work end-to-end.
type versionInfo struct {
	CLIVersion   string `json:"cli_version"`
	Connectivity string `json:"connectivity"`
	Host         string `json:"host,omitempty"`
	User         string `json:"user,omitempty"`
	// Error carries the probe failure message when Connectivity is "error".
	Error string `json:"error,omitempty"`
}

const versionTemplate = `Databricks AI runtime CLI v{{.CLIVersion}}
{{- if eq .Connectivity "ok"}}
Host: {{.Host}}
User: {{.User}}
{{- else}}
Workspace: unreachable
{{- end}}
`

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Args:  root.NoArgs,
		Short: "Show the AI runtime CLI version and verify workspace connectivity",
		Long: `Show the AI runtime CLI version and verify workspace connectivity.

This is the proof-of-life command for the AI runtime CLI: it prints the local
CLI version and makes an authenticated call to the resolved workspace, so it
exercises profile selection, authentication, and output rendering in one shot.

The local version always prints. If the workspace probe fails (e.g. missing or
expired credentials) the command reports the failure and exits non-zero.`,
		Annotations: map[string]string{
			"template": versionTemplate,
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		info := versionInfo{
			CLIVersion:   build.GetInfo().Version,
			Connectivity: "ok",
		}

		// Proof-of-life probe: resolve the profile, authenticate, and make one authenticated round-trip. We resolve the client inside RunE
		// so the local version still prints when the workspace is unreachable.
		probeErr := root.MustWorkspaceClient(cmd, args)
		if probeErr == nil {
			w := cmdctx.WorkspaceClient(ctx)
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				probeErr = err
			} else {
				info.Host = w.Config.Host
				info.User = me.UserName
			}
		}
		if probeErr != nil {
			info.Connectivity = "error"
			info.Error = probeErr.Error()
		}

		if err := cmdio.Render(ctx, info); err != nil {
			return err
		}

		// A failed probe is a real failure for a proof-of-life command: return it
		// so the command exits non-zero and the actionable auth error is surfaced by the CLI's error handler.
		return probeErr
	}

	return cmd
}
