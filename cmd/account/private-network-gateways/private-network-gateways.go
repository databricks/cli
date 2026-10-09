// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package private_network_gateways

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/experimental/api"
	"github.com/databricks/databricks-sdk-go/service/networking"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "private-network-gateways",
		Short: `These APIs manage private network gateways under network connectivity configurations.`,
		Long: `These APIs manage private network gateways under network connectivity
  configurations.`,
		GroupID: "provisioning",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreatePrivateNetworkGateway())
	cmd.AddCommand(newDeletePrivateNetworkGateway())
	cmd.AddCommand(newGetPrivateNetworkGateway())
	cmd.AddCommand(newGetPrivateNetworkGatewayOperation())
	cmd.AddCommand(newListPrivateNetworkGateways())
	cmd.AddCommand(newUpdatePrivateNetworkGateway())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-private-network-gateway command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createPrivateNetworkGatewayOverrides []func(
	*cobra.Command,
	*networking.CreatePrivateNetworkGatewayRequest,
)

func newCreatePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}

	var createPrivateNetworkGatewayReq networking.CreatePrivateNetworkGatewayRequest
	createPrivateNetworkGatewayReq.PrivateNetworkGateway = networking.PrivateNetworkGateway{}
	var createPrivateNetworkGatewayJson flags.JsonFlag

	var createPrivateNetworkGatewaySkipWait bool
	var createPrivateNetworkGatewayTimeout time.Duration

	cmd.Flags().BoolVar(&createPrivateNetworkGatewaySkipWait, "no-wait", createPrivateNetworkGatewaySkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createPrivateNetworkGatewayTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createPrivateNetworkGatewayJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createPrivateNetworkGatewayReq.RequestId, "request-id", createPrivateNetworkGatewayReq.RequestId, `A unique identifier for this request.`)
	// TODO: complex arg: aws_cloud_connection
	// TODO: complex arg: azure_cloud_connection
	cmd.Flags().IntVar(&createPrivateNetworkGatewayReq.PrivateNetworkGateway.BandwidthTierGigabitsPerSecond, "bandwidth-tier-gigabits-per-second", createPrivateNetworkGatewayReq.PrivateNetworkGateway.BandwidthTierGigabitsPerSecond, `The provisioned bandwidth tier for an Azure gateway, in gigabits per second.`)
	// TODO: array: destinations
	cmd.Flags().StringVar(&createPrivateNetworkGatewayReq.PrivateNetworkGateway.Name, "name", createPrivateNetworkGatewayReq.PrivateNetworkGateway.Name, `The canonical resource name of the gateway, in the form accounts/{account_id}/network-connectivity-configs/{ncc_id}/private-network-gateways/{gateway_id}.`)
	// TODO: array: private_dns_resolvers

	cmd.Use = "create-private-network-gateway PARENT DISPLAY_NAME TRAFFIC_MODE"
	cmd.Short = `Create a private network gateway.`
	cmd.Long = `Create a private network gateway.

  Creates a private network gateway.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-private-network-gateway-operation command.

  Arguments:
    PARENT: The network connectivity configuration that will contain the gateway.
    DISPLAY_NAME: The human-readable name of the gateway.
    TRAFFIC_MODE: The traffic routed through this gateway.
      Supported values: [ALL_TRAFFIC, SPECIFIC_DESTINATIONS]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'display_name', 'traffic_mode' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createPrivateNetworkGatewayJson.Unmarshal(&createPrivateNetworkGatewayReq.PrivateNetworkGateway)
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
		createPrivateNetworkGatewayReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createPrivateNetworkGatewayReq.PrivateNetworkGateway.DisplayName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createPrivateNetworkGatewayReq.PrivateNetworkGateway.TrafficMode)
			if err != nil {
				return fmt.Errorf("invalid TRAFFIC_MODE: %s", args[2])
			}

		}

		// Determine which mode to execute based on flags.
		switch {
		case createPrivateNetworkGatewaySkipWait:
			wait, err := a.PrivateNetworkGateways.CreatePrivateNetworkGateway(ctx, createPrivateNetworkGatewayReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := a.PrivateNetworkGateways.GetPrivateNetworkGatewayOperation(ctx, networking.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := a.PrivateNetworkGateways.CreatePrivateNetworkGateway(ctx, createPrivateNetworkGatewayReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-private-network-gateway to complete...")

			// Wait for completion.
			opts := api.WithTimeout(createPrivateNetworkGatewayTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			sp.Close()
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createPrivateNetworkGatewayOverrides {
		fn(cmd, &createPrivateNetworkGatewayReq)
	}

	return cmd
}

// start delete-private-network-gateway command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deletePrivateNetworkGatewayOverrides []func(
	*cobra.Command,
	*networking.DeletePrivateNetworkGatewayRequest,
)

func newDeletePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}

	var deletePrivateNetworkGatewayReq networking.DeletePrivateNetworkGatewayRequest

	cmd.Use = "delete-private-network-gateway NAME"
	cmd.Short = `Delete a private network gateway.`
	cmd.Long = `Delete a private network gateway.

  Permanently deletes a private network gateway.

  Arguments:
    NAME: The canonical resource name of the gateway.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deletePrivateNetworkGatewayReq.Name = args[0]

		err = a.PrivateNetworkGateways.DeletePrivateNetworkGateway(ctx, deletePrivateNetworkGatewayReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deletePrivateNetworkGatewayOverrides {
		fn(cmd, &deletePrivateNetworkGatewayReq)
	}

	return cmd
}

// start get-private-network-gateway command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPrivateNetworkGatewayOverrides []func(
	*cobra.Command,
	*networking.GetPrivateNetworkGatewayRequest,
)

func newGetPrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}

	var getPrivateNetworkGatewayReq networking.GetPrivateNetworkGatewayRequest

	cmd.Use = "get-private-network-gateway NAME"
	cmd.Short = `Get a private network gateway.`
	cmd.Long = `Get a private network gateway.

  Gets a private network gateway.

  Arguments:
    NAME: The canonical resource name of the gateway.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getPrivateNetworkGatewayReq.Name = args[0]

		response, err := a.PrivateNetworkGateways.GetPrivateNetworkGateway(ctx, getPrivateNetworkGatewayReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPrivateNetworkGatewayOverrides {
		fn(cmd, &getPrivateNetworkGatewayReq)
	}

	return cmd
}

// start get-private-network-gateway-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPrivateNetworkGatewayOperationOverrides []func(
	*cobra.Command,
	*networking.GetOperationRequest,
)

func newGetPrivateNetworkGatewayOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getPrivateNetworkGatewayOperationReq networking.GetOperationRequest

	cmd.Use = "get-private-network-gateway-operation NAME"
	cmd.Short = `Get a private network gateway operation.`
	cmd.Long = `Get a private network gateway operation.

  Gets the status of a private network gateway create operation.

  Arguments:
    NAME: The name of the operation resource.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getPrivateNetworkGatewayOperationReq.Name = args[0]

		response, err := a.PrivateNetworkGateways.GetPrivateNetworkGatewayOperation(ctx, getPrivateNetworkGatewayOperationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPrivateNetworkGatewayOperationOverrides {
		fn(cmd, &getPrivateNetworkGatewayOperationReq)
	}

	return cmd
}

// start list-private-network-gateways command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPrivateNetworkGatewaysOverrides []func(
	*cobra.Command,
	*networking.ListPrivateNetworkGatewaysRequest,
)

func newListPrivateNetworkGateways() *cobra.Command {
	cmd := &cobra.Command{}

	var listPrivateNetworkGatewaysReq networking.ListPrivateNetworkGatewaysRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listPrivateNetworkGatewaysLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listPrivateNetworkGatewaysLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listPrivateNetworkGatewaysReq.PageToken, "page-token", listPrivateNetworkGatewaysReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-private-network-gateways PARENT"
	cmd.Short = `List private network gateways.`
	cmd.Long = `List private network gateways.

  Lists private network gateways under a network connectivity configuration.

  Arguments:
    PARENT: The network connectivity configuration containing the gateways.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		listPrivateNetworkGatewaysReq.Parent = args[0]

		response := a.PrivateNetworkGateways.ListPrivateNetworkGateways(ctx, listPrivateNetworkGatewaysReq)
		if listPrivateNetworkGatewaysLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listPrivateNetworkGatewaysLimit)
		}
		if listPrivateNetworkGatewaysLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listPrivateNetworkGatewaysLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPrivateNetworkGatewaysOverrides {
		fn(cmd, &listPrivateNetworkGatewaysReq)
	}

	return cmd
}

// start update-private-network-gateway command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePrivateNetworkGatewayOverrides []func(
	*cobra.Command,
	*networking.UpdatePrivateNetworkGatewayRequest,
)

func newUpdatePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePrivateNetworkGatewayReq networking.UpdatePrivateNetworkGatewayRequest
	updatePrivateNetworkGatewayReq.PrivateNetworkGateway = networking.PrivateNetworkGateway{}
	var updatePrivateNetworkGatewayJson flags.JsonFlag

	cmd.Flags().Var(&updatePrivateNetworkGatewayJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_cloud_connection
	// TODO: complex arg: azure_cloud_connection
	cmd.Flags().IntVar(&updatePrivateNetworkGatewayReq.PrivateNetworkGateway.BandwidthTierGigabitsPerSecond, "bandwidth-tier-gigabits-per-second", updatePrivateNetworkGatewayReq.PrivateNetworkGateway.BandwidthTierGigabitsPerSecond, `The provisioned bandwidth tier for an Azure gateway, in gigabits per second.`)
	// TODO: array: destinations
	cmd.Flags().StringVar(&updatePrivateNetworkGatewayReq.PrivateNetworkGateway.Name, "name", updatePrivateNetworkGatewayReq.PrivateNetworkGateway.Name, `The canonical resource name of the gateway, in the form accounts/{account_id}/network-connectivity-configs/{ncc_id}/private-network-gateways/{gateway_id}.`)
	// TODO: array: private_dns_resolvers

	cmd.Use = "update-private-network-gateway NAME UPDATE_MASK DISPLAY_NAME TRAFFIC_MODE"
	cmd.Short = `Update a private network gateway.`
	cmd.Long = `Update a private network gateway.

  Updates a private network gateway.

  Arguments:
    NAME: The canonical resource name of the gateway, in the form
      accounts/{account_id}/network-connectivity-configs/{ncc_id}/private-network-gateways/{gateway_id}.
    UPDATE_MASK: The fields to update.
    DISPLAY_NAME: The human-readable name of the gateway.
    TRAFFIC_MODE: The traffic routed through this gateway.
      Supported values: [ALL_TRAFFIC, SPECIFIC_DESTINATIONS]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'traffic_mode' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePrivateNetworkGatewayJson.Unmarshal(&updatePrivateNetworkGatewayReq.PrivateNetworkGateway)
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
		updatePrivateNetworkGatewayReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updatePrivateNetworkGatewayReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updatePrivateNetworkGatewayReq.PrivateNetworkGateway.DisplayName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updatePrivateNetworkGatewayReq.PrivateNetworkGateway.TrafficMode)
			if err != nil {
				return fmt.Errorf("invalid TRAFFIC_MODE: %s", args[3])
			}

		}

		response, err := a.PrivateNetworkGateways.UpdatePrivateNetworkGateway(ctx, updatePrivateNetworkGatewayReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePrivateNetworkGatewayOverrides {
		fn(cmd, &updatePrivateNetworkGatewayReq)
	}

	return cmd
}

// end service PrivateNetworkGateways
