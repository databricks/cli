// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package endpoints

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
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
		Use:     "endpoints",
		Short:   `These APIs manage endpoint configurations for this account.`,
		Long:    `These APIs manage endpoint configurations for this account.`,
		GroupID: "provisioning",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newListEndpoints())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createEndpointOverrides []func(
	*cobra.Command,
	*networking.CreateEndpointRequest,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var createEndpointReq networking.CreateEndpointRequest
	createEndpointReq.Endpoint = &networking.Endpoint{}

	var createEndpointJson flags.JsonFlag

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_vpc_endpoint_info
	// TODO: complex arg: azure_private_endpoint_info
	// TODO: complex arg: gcp_psc_endpoint_info

	cmd.Use = "create-endpoint PARENT DISPLAY_NAME REGION"
	cmd.Short = `Create a network endpoint.`
	cmd.Long = `Create a network endpoint.

  Creates a new network connectivity endpoint that enables private connectivity
  between your network resources and Databricks services.

  After creation, the endpoint is initially in the PENDING state. The Databricks
  endpoint service automatically reviews and approves the endpoint within a few
  minutes. Use the GET method to retrieve the latest endpoint state.

  An endpoint can be used only after it reaches the APPROVED state.

  Arguments:
    PARENT: The parent resource name of the account under which the endpoint is
      created. Format: accounts/{account_id}.
    DISPLAY_NAME: The human-readable display name of this endpoint. The input should conform
      to RFC-1034, which restricts to letters, numbers, and hyphens, with the
      first character a letter, the last a letter or a number, and a 63
      character maximum.
    REGION: The cloud provider region where this endpoint is located.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'display_name', 'region' in your JSON input")
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
			diags := createEndpointJson.Unmarshal(&createEndpointReq.Endpoint)
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
		createEndpointReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Endpoint.DisplayName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Endpoint.Region = new(args[2])
		}

		sdkReq, err := networking.CreateEndpointRequestToSDK(&createEndpointReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.CreateEndpoint(ctx, *sdkReq)
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
	for _, fn := range createEndpointOverrides {
		fn(cmd, &createEndpointReq)
	}

	return cmd
}

// start delete-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteEndpointOverrides []func(
	*cobra.Command,
	*sdknetworking.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteEndpointReq sdknetworking.DeleteEndpointRequest

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `Delete a network endpoint.`
	cmd.Long = `Delete a network endpoint.

  Deletes a network endpoint. This will remove the endpoint configuration from
  Databricks. Depending on the endpoint type and use case, you may also need to
  delete corresponding network resources in your cloud provider account.`

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

		deleteEndpointReq.Name = new(args[0])

		sdkReq := &deleteEndpointReq
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
	for _, fn := range deleteEndpointOverrides {
		fn(cmd, &deleteEndpointReq)
	}

	return cmd
}

// start get-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEndpointOverrides []func(
	*cobra.Command,
	*sdknetworking.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var getEndpointReq sdknetworking.GetEndpointRequest

	cmd.Use = "get-endpoint NAME"
	cmd.Short = `Get a network endpoint.`
	cmd.Long = `Get a network endpoint.

  Gets details of a specific network endpoint.`

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

		getEndpointReq.Name = new(args[0])

		sdkReq := &getEndpointReq
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
	for _, fn := range getEndpointOverrides {
		fn(cmd, &getEndpointReq)
	}

	return cmd
}

// start list-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEndpointsOverrides []func(
	*cobra.Command,
	*networking.ListEndpointsRequest,
)

func newListEndpoints() *cobra.Command {
	cmd := &cobra.Command{}
	var listEndpointsReq networking.ListEndpointsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEndpointsLimit int

	flags.IntPtrVar(cmd.Flags(), &listEndpointsReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEndpointsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listEndpointsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `List network endpoints.`
	cmd.Long = `List network endpoints.

  Lists all network connectivity endpoints for the account.

  Arguments:
    PARENT: The parent resource name of the account to list endpoints for. Format:
      accounts/{account_id}.`

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

		listEndpointsReq.Parent = new(args[0])

		sdkReq, err := networking.ListEndpointsRequestToSDK(&listEndpointsReq)
		if err != nil {
			return err
		}
		if listEndpointsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEndpointsLimit)
		}
		if listEndpointsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEndpointsLimit)
		}
		response := a.Networking.ListEndpointsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, networking.EndpointFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEndpointsOverrides {
		fn(cmd, &listEndpointsReq)
	}

	return cmd
}

// end service Endpoints
