// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package private_network_gateways

import (
	"errors"
	"fmt"
	"time"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
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
	*map[string]any,
)

func newCreatePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}
	var createPrivateNetworkGatewayReq map[string]any
	createPrivateNetworkGatewayReq = map[string]any{}
	var createPrivateNetworkGatewayParentRaw string
	var createPrivateNetworkGatewayRequestIdRaw string
	var createPrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw int
	var createPrivateNetworkGatewayDisplayNameRaw string
	var createPrivateNetworkGatewayErrorMessageRaw string
	var createPrivateNetworkGatewayNameRaw string
	var createPrivateNetworkGatewayStateRaw string
	var createPrivateNetworkGatewayTrafficModeRaw string

	var createPrivateNetworkGatewayJson flags.JsonFlag

	var createPrivateNetworkGatewaySkipWait bool
	var createPrivateNetworkGatewayTimeout time.Duration

	cmd.Flags().BoolVar(&createPrivateNetworkGatewaySkipWait, "no-wait", createPrivateNetworkGatewaySkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createPrivateNetworkGatewayTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createPrivateNetworkGatewayJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createPrivateNetworkGatewayRequestIdRaw, "request-id", createPrivateNetworkGatewayRequestIdRaw, `A unique identifier for this request.`)
	// TODO: complex arg: aws_cloud_connection
	// TODO: complex arg: azure_cloud_connection
	cmd.Flags().IntVar(&createPrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw, "bandwidth-tier-gigabits-per-second", createPrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw, `The provisioned bandwidth tier for an Azure gateway, in gigabits per second.`)
	// TODO: array: destinations
	cmd.Flags().StringVar(&createPrivateNetworkGatewayNameRaw, "name", createPrivateNetworkGatewayNameRaw, `The canonical resource name of the gateway, in the form accounts/{account_id}/network-connectivity-configs/{ncc_id}/private-network-gateways/{gateway_id}.`)
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
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createPrivateNetworkGatewayJson.Unmarshal(&createPrivateNetworkGatewayReq)
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
		createPrivateNetworkGatewayParentRaw = args[0]
		if !cmd.Flags().Changed("json") {
			createPrivateNetworkGatewayDisplayNameRaw = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createPrivateNetworkGatewayTrafficModeRaw = args[2]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("request-id") || createPrivateNetworkGatewayRequestIdRaw != "" {
			rawQuery["request_id"] = createPrivateNetworkGatewayRequestIdRaw
		}
		if cmd.Flags().Changed("bandwidth-tier-gigabits-per-second") || createPrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw != 0 {
			createPrivateNetworkGatewayReq["bandwidth_tier_gigabits_per_second"] = createPrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw
		}
		if cmd.Flags().Changed("display-name") || createPrivateNetworkGatewayDisplayNameRaw != "" {
			createPrivateNetworkGatewayReq["display_name"] = createPrivateNetworkGatewayDisplayNameRaw
		}
		if cmd.Flags().Changed("error-message") || createPrivateNetworkGatewayErrorMessageRaw != "" {
			createPrivateNetworkGatewayReq["error_message"] = createPrivateNetworkGatewayErrorMessageRaw
		}
		if cmd.Flags().Changed("name") || createPrivateNetworkGatewayNameRaw != "" {
			createPrivateNetworkGatewayReq["name"] = createPrivateNetworkGatewayNameRaw
		}
		if cmd.Flags().Changed("state") || createPrivateNetworkGatewayStateRaw != "" {
			createPrivateNetworkGatewayReq["state"] = createPrivateNetworkGatewayStateRaw
		}
		if cmd.Flags().Changed("traffic-mode") || createPrivateNetworkGatewayTrafficModeRaw != "" {
			createPrivateNetworkGatewayReq["traffic_mode"] = createPrivateNetworkGatewayTrafficModeRaw
		}
		rawPath := fmt.Sprintf("/api/networking/v1/%s/private-network-gateways", fmt.Sprint(createPrivateNetworkGatewayParentRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, nil, rawQuery, createPrivateNetworkGatewayReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*sdknetworking.DeleteEndpointRequest,
)

func newDeletePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}
	var deletePrivateNetworkGatewayReq sdknetworking.DeleteEndpointRequest

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
		_ = a

		deletePrivateNetworkGatewayReq.Name = new(args[0])

		sdkReq := &deletePrivateNetworkGatewayReq
		err = a.Networking.DeleteEndpoint(ctx, *sdkReq)
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
	*sdknetworking.GetEndpointRequest,
)

func newGetPrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}
	var getPrivateNetworkGatewayReq sdknetworking.GetEndpointRequest

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
		_ = a

		getPrivateNetworkGatewayReq.Name = new(args[0])

		sdkReq := &getPrivateNetworkGatewayReq
		response, err := a.Networking.GetEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdknetworking.GetEndpointRequest,
)

func newGetPrivateNetworkGatewayOperation() *cobra.Command {
	cmd := &cobra.Command{}
	var getPrivateNetworkGatewayOperationReq sdknetworking.GetEndpointRequest

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
		_ = a

		getPrivateNetworkGatewayOperationReq.Name = new(args[0])

		sdkReq := &getPrivateNetworkGatewayOperationReq
		response, err := a.Networking.GetEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*map[string]any,
)

func newListPrivateNetworkGateways() *cobra.Command {
	cmd := &cobra.Command{}
	var listPrivateNetworkGatewaysReq map[string]any
	listPrivateNetworkGatewaysReq = map[string]any{}
	var listPrivateNetworkGatewaysPageTokenRaw string
	var listPrivateNetworkGatewaysParentRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listPrivateNetworkGatewaysLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listPrivateNetworkGatewaysLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listPrivateNetworkGatewaysPageTokenRaw, "page-token", listPrivateNetworkGatewaysPageTokenRaw, `Pagination token.`)
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
		_ = a

		listPrivateNetworkGatewaysParentRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-token") || listPrivateNetworkGatewaysPageTokenRaw != "" {
			rawQuery["page_token"] = listPrivateNetworkGatewaysPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/networking/v1/%s/private-network-gateways", fmt.Sprint(listPrivateNetworkGatewaysParentRaw))

		maps.Copy(rawQuery, listPrivateNetworkGatewaysReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listPrivateNetworkGatewaysLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listPrivateNetworkGatewaysLimit)
		}
		return cmdio.Render(ctx, response)
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
	*map[string]any,
)

func newUpdatePrivateNetworkGateway() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePrivateNetworkGatewayReq map[string]any
	updatePrivateNetworkGatewayReq = map[string]any{}
	var updatePrivateNetworkGatewayNameRaw string
	var updatePrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw int
	var updatePrivateNetworkGatewayDisplayNameRaw string
	var updatePrivateNetworkGatewayErrorMessageRaw string
	var updatePrivateNetworkGatewayStateRaw string
	var updatePrivateNetworkGatewayTrafficModeRaw string

	var updatePrivateNetworkGatewayJson flags.JsonFlag

	cmd.Flags().Var(&updatePrivateNetworkGatewayJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_cloud_connection
	// TODO: complex arg: azure_cloud_connection
	cmd.Flags().IntVar(&updatePrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw, "bandwidth-tier-gigabits-per-second", updatePrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw, `The provisioned bandwidth tier for an Azure gateway, in gigabits per second.`)
	// TODO: array: destinations
	cmd.Flags().StringVar(&updatePrivateNetworkGatewayNameRaw, "name", updatePrivateNetworkGatewayNameRaw, `The canonical resource name of the gateway, in the form accounts/{account_id}/network-connectivity-configs/{ncc_id}/private-network-gateways/{gateway_id}.`)
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
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updatePrivateNetworkGatewayJson.Unmarshal(&updatePrivateNetworkGatewayReq)
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
		updatePrivateNetworkGatewayNameRaw = args[0]
		if args[1] != "" {
			updatePrivateNetworkGatewayReq["update_mask"] = args[1]
		}
		if !cmd.Flags().Changed("json") {
			updatePrivateNetworkGatewayDisplayNameRaw = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updatePrivateNetworkGatewayTrafficModeRaw = args[3]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("bandwidth-tier-gigabits-per-second") || updatePrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw != 0 {
			updatePrivateNetworkGatewayReq["bandwidth_tier_gigabits_per_second"] = updatePrivateNetworkGatewayBandwidthTierGigabitsPerSecondRaw
		}
		if cmd.Flags().Changed("display-name") || updatePrivateNetworkGatewayDisplayNameRaw != "" {
			updatePrivateNetworkGatewayReq["display_name"] = updatePrivateNetworkGatewayDisplayNameRaw
		}
		if cmd.Flags().Changed("error-message") || updatePrivateNetworkGatewayErrorMessageRaw != "" {
			updatePrivateNetworkGatewayReq["error_message"] = updatePrivateNetworkGatewayErrorMessageRaw
		}
		if cmd.Flags().Changed("name") || updatePrivateNetworkGatewayNameRaw != "" {
			updatePrivateNetworkGatewayReq["name"] = updatePrivateNetworkGatewayNameRaw
		}
		if cmd.Flags().Changed("state") || updatePrivateNetworkGatewayStateRaw != "" {
			updatePrivateNetworkGatewayReq["state"] = updatePrivateNetworkGatewayStateRaw
		}
		if cmd.Flags().Changed("traffic-mode") || updatePrivateNetworkGatewayTrafficModeRaw != "" {
			updatePrivateNetworkGatewayReq["traffic_mode"] = updatePrivateNetworkGatewayTrafficModeRaw
		}
		rawPath := fmt.Sprintf("/api/networking/v1/%s", fmt.Sprint(updatePrivateNetworkGatewayNameRaw))

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updatePrivateNetworkGatewayReq, &response)
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
