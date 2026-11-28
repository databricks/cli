package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
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
			var result *sdk.Sandbox
			if wantJSON {
				result, err = createAndWait(ctx, api, name, nil)
				if err != nil {
					return fmt.Errorf("failed to create sandbox: %w", err)
				}
			} else {
				s := spin(ctx, "Provisioning your sandbox…")
				defer s.Close()
				result, err = createAndWait(ctx, api, name, s)
				if err != nil {
					s.fail("Failed to create sandbox")
					return fmt.Errorf("failed to create sandbox: %w", err)
				}
				s.ok("Sandbox " + cmdio.Bold(ctx, sandboxResourceID(result)) + " is " + status(ctx, sandboxStatus(result)))
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

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
				if err := setDefault(ctx, profile, sandboxResourceID(result)); err != nil {
					warn(ctx, fmt.Sprintf("Could not save default: %v", err))
				} else if !wantJSON {
					field(ctx, cmd.ErrOrStderr(), "default", sandboxResourceID(result))
				}
			}

			if wantJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(createJSONOutput{SandboxID: sandboxResourceID(result), Status: sandboxStatus(result)})
			}

			blank(cmd.ErrOrStderr())
			fmt.Fprintln(cmd.OutOrStdout(), sandboxResourceID(result))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display label for the sandbox (max 256 bytes)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

func createAndWait(ctx context.Context, api *sandboxAPI, name string, s *spinner) (*sdk.Sandbox, error) {
	result, err := api.create(ctx, name)
	if err != nil {
		return nil, err
	}
	// Provisioning may fail or be cancelled after creation was accepted. Keep the
	// ID discoverable locally without making an unready sandbox the default.
	profile := api.c.Config.Profile
	if profile == "" {
		profile = api.c.Config.Host
	}
	_ = upsertSandbox(ctx, profile, sandboxResourceID(result), name)

	if sandboxStatus(result) != "Running" {
		final, err := waitForState(ctx, api, s, sandboxResourceID(result), "Running", "Provisioning", "Stopped", "Terminated", "Failed")
		if err != nil {
			return nil, fmt.Errorf("sandbox %s was created but is not ready: %w", sandboxResourceID(result), err)
		}
		result = final
	}
	return result, nil
}
