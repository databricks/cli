// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package networks

import (
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
		Use:   "networks",
		Short: `These APIs manage network configurations for customer-managed VPCs (optional).`,
		Long: `These APIs manage network configurations for customer-managed VPCs (optional).
  Its ID is used when creating a new workspace if you use customer-managed VPCs.`,
		GroupID: "provisioning",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*networking.CreateNetworkRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq networking.CreateNetworkRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: gcp_network_info
	flags.StringPtrVar(cmd.Flags(), &createReq.NetworkName, "network-name", `The human-readable name of the network configuration.`)
	// TODO: array: security_group_ids
	// TODO: array: subnet_ids
	// TODO: complex arg: vpc_endpoints
	flags.StringPtrVar(cmd.Flags(), &createReq.VpcId, "vpc-id", `The ID of the VPC associated with this network configuration.`)

	cmd.Use = "create"
	cmd.Short = `Create network configuration.`
	cmd.Long = `Create network configuration.

  Creates a Databricks network configuration that represents an VPC and its
  resources. The VPC will be used for new Databricks clusters. This requires a
  pre-existing VPC and subnets.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		sdkReq, err := networking.CreateNetworkRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.CreateNetworkPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.NetworkFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdknetworking.DeleteNetworkRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdknetworking.DeleteNetworkRequest

	cmd.Use = "delete NETWORK_ID"
	cmd.Short = `Delete a network configuration.`
	cmd.Long = `Delete a network configuration.

  Deletes a Databricks network configuration, which represents a cloud VPC and
  its resources. You cannot delete a network that is associated with a
  workspace.

  This operation is available only if your account is on the E2 version of the
  platform.

  Arguments:
    NETWORK_ID: Databricks Account API network configuration ID.`

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

		deleteReq.NetworkId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.Networking.DeleteNetworkPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.NetworkFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sdknetworking.GetNetworkRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdknetworking.GetNetworkRequest

	cmd.Use = "get NETWORK_ID"
	cmd.Short = `Get a network configuration.`
	cmd.Long = `Get a network configuration.

  Gets a Databricks network configuration, which represents a cloud VPC and its
  resources.

  Arguments:
    NETWORK_ID: Databricks Account API network configuration ID.`

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

		getReq.NetworkId = new(args[0])

		sdkReq := &getReq
		response, err := a.Networking.GetNetworkPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.NetworkFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `List network configurations.`
	cmd.Long = `List network configurations.

  Lists Databricks network configurations for an account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdknetworking.ListNetworkRequest{}

		response, err := a.Networking.ListNetworkPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Networks
