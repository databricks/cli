package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var name string
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Sandbox environment",
		Long: `Create a new Sandbox environment.

Creates a new personal development environment backed by a microVM.
Blocks until the sandbox is running and prints the sandbox ID.

Examples:
  databricks sandbox create
  databricks sandbox create --name my-project
  databricks sandbox create --json`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateName(name); err != nil {
				return err
			}

			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			wantJSON := jsonOutput(cmd, outputJSON)

			// In JSON mode, suppress the spinner+ok lines so the only thing
			// on stdout/stderr that a scripted caller has to parse is the
			// JSON body itself.
			var result *createResponse
			if wantJSON {
				result, err = api.create(ctx, name)
				if err != nil {
					return fmt.Errorf("failed to create sandbox: %w", err)
				}
			} else {
				s := spin(ctx, "Provisioning your sandbox…")
				defer s.Close()
				result, err = api.create(ctx, name)
				if err != nil {
					s.fail("Failed to create sandbox")
					return fmt.Errorf("failed to create sandbox: %w", err)
				}
				s.ok("Sandbox " + cmdio.Bold(ctx, result.SandboxID) + " is " + status(ctx, result.Status))
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			_ = setGatewayHost(ctx, profile, result.GatewayHost)
			_ = upsertSandbox(ctx, profile, result.SandboxID, name)

			// Only clobber an existing default if it's actually gone
			// (404). Transient errors (5xx, network blip, rate limit)
			// must not silently overwrite the user's chosen default.
			currentDefault := getDefault(ctx, profile)
			shouldSetDefault := currentDefault == ""
			if !shouldSetDefault {
				if _, err := api.get(ctx, currentDefault); errors.Is(err, apierr.ErrNotFound) {
					shouldSetDefault = true
				}
			}
			if shouldSetDefault {
				if err := setDefault(ctx, profile, result.SandboxID); err != nil {
					warn(ctx, fmt.Sprintf("Could not save default: %v", err))
				} else if !wantJSON {
					field(ctx, cmd.ErrOrStderr(), "default", result.SandboxID)
				}
			}

			if wantJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			blank(cmd.ErrOrStderr())
			fmt.Fprintln(cmd.OutOrStdout(), result.SandboxID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display label for the sandbox (max 256 bytes)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
