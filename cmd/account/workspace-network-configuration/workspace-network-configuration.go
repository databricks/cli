// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_network_configuration

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	networking "github.com/databricks/cli/libs/sdktypes/networking/v1"
	sdknetworking "github.com/databricks/sdk-go/networking/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-network-configuration",
		Short: `These APIs allow configuration of network settings for Databricks workspaces by selecting which network policy to associate with the workspace.`,
		Long: `These APIs allow configuration of network settings for Databricks workspaces
  by selecting which network policy to associate with the workspace. Each
  workspace is always associated with exactly one network policy that controls
  which network destinations can be accessed from the Databricks environment. By
  default, workspaces are associated with the 'default-policy' network policy.
  You cannot create or delete a workspace's network option, only update it to
  associate the workspace with a different policy`,
		GroupID: "settings",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGetWorkspaceNetworkOptionRpc())
	cmd.AddCommand(newUpdateWorkspaceNetworkOptionRpc())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-workspace-network-option-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceNetworkOptionRpcOverrides []func(
	*cobra.Command,
	*sdknetworking.GetWorkspaceNetworkOptionRequest,
)

func newGetWorkspaceNetworkOptionRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var getWorkspaceNetworkOptionRpcReq sdknetworking.GetWorkspaceNetworkOptionRequest

	cmd.Use = "get-workspace-network-option-rpc WORKSPACE_ID"
	cmd.Short = `Get workspace network option.`
	cmd.Long = `Get workspace network option.

  Gets the network option for a workspace. Every workspace has exactly one
  network policy binding, with 'default-policy' used if no explicit assignment
  exists.

  Arguments:
    WORKSPACE_ID: The workspace ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		var getWorkspaceNetworkOptionRpcWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &getWorkspaceNetworkOptionRpcWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		getWorkspaceNetworkOptionRpcReq.WorkspaceId = &getWorkspaceNetworkOptionRpcWorkspaceIdArg

		sdkReq := &getWorkspaceNetworkOptionRpcReq
		response, err := a.Networking.GetWorkspaceNetworkOptionRpc(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.WorkspaceNetworkOptionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceNetworkOptionRpcOverrides {
		fn(cmd, &getWorkspaceNetworkOptionRpcReq)
	}

	return cmd
}

// start update-workspace-network-option-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceNetworkOptionRpcOverrides []func(
	*cobra.Command,
	*networking.UpdateWorkspaceNetworkOptionRequest,
)

func newUpdateWorkspaceNetworkOptionRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWorkspaceNetworkOptionRpcReq networking.UpdateWorkspaceNetworkOptionRequest
	updateWorkspaceNetworkOptionRpcReq.WorkspaceNetworkOption = &networking.WorkspaceNetworkOption{}

	var updateWorkspaceNetworkOptionRpcJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceNetworkOptionRpcJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateWorkspaceNetworkOptionRpcReq.WorkspaceNetworkOption.NetworkPolicyId, "network-policy-id", `The network policy ID to apply to the workspace.`)
	flags.Int64PtrVar(cmd.Flags(), &updateWorkspaceNetworkOptionRpcReq.WorkspaceId, "workspace-id", `The workspace ID.`)

	cmd.Use = "update-workspace-network-option-rpc WORKSPACE_ID"
	cmd.Short = `Update workspace network option.`
	cmd.Long = `Update workspace network option.

  Updates the network option for a workspace. This operation associates the
  workspace with the specified network policy. To revert to the default policy,
  specify 'default-policy' as the network_policy_id.

  Arguments:
    WORKSPACE_ID: The workspace ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceNetworkOptionRpcJson.Unmarshal(&updateWorkspaceNetworkOptionRpcReq.WorkspaceNetworkOption)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		var updateWorkspaceNetworkOptionRpcWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &updateWorkspaceNetworkOptionRpcWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		updateWorkspaceNetworkOptionRpcReq.WorkspaceId = &updateWorkspaceNetworkOptionRpcWorkspaceIdArg

		sdkReq, err := networking.UpdateWorkspaceNetworkOptionRequestToSDK(&updateWorkspaceNetworkOptionRpcReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.UpdateWorkspaceNetworkOptionRpc(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.WorkspaceNetworkOptionFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceNetworkOptionRpcOverrides {
		fn(cmd, &updateWorkspaceNetworkOptionRpcReq)
	}

	return cmd
}

// end service WorkspaceNetworkConfiguration
