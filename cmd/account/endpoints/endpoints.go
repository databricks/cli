// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package endpoints

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/networking"
	"github.com/spf13/cobra"
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
	createEndpointReq.Endpoint = networking.Endpoint{}
	var createEndpointJson flags.JsonFlag

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: azure_private_endpoint_info

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
    PARENT:
    DISPLAY_NAME: The human-readable display name of this endpoint. The input should conform
      to RFC-1034, which restricts to letters, numbers, and hyphens, with the
      first character a letter, the last a letter or a number, and a 63
      character maximum.
    REGION: The cloud provider region where this endpoint is located.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT as positional arguments. Provide 'display_name', 'region' in your JSON input")
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
		createEndpointReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Endpoint.DisplayName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Endpoint.Region = args[2]
		}

		response, err := a.Endpoints.CreateEndpoint(ctx, createEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*networking.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteEndpointReq networking.DeleteEndpointRequest

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `Delete a network endpoint.`
	cmd.Long = `Delete a network endpoint.

  Deletes a network endpoint. This will remove the endpoint configuration from
  Databricks. Depending on the endpoint type and use case, you may also need to
  delete corresponding network resources in your cloud provider account.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteEndpointReq.Name = args[0]

		err = a.Endpoints.DeleteEndpoint(ctx, deleteEndpointReq)
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
	*networking.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var getEndpointReq networking.GetEndpointRequest

	cmd.Use = "get-endpoint NAME"
	cmd.Short = `Get a network endpoint.`
	cmd.Long = `Get a network endpoint.

  Gets details of a specific network endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getEndpointReq.Name = args[0]

		response, err := a.Endpoints.GetEndpoint(ctx, getEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	cmd.Flags().IntVar(&listEndpointsReq.PageSize, "page-size", listEndpointsReq.PageSize, ``)
	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, ``)

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `List network endpoints.`
	cmd.Long = `List network endpoints.

  Lists all network connectivity endpoints for the account.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		listEndpointsReq.Parent = args[0]

		response := a.Endpoints.ListEndpoints(ctx, listEndpointsReq)
		return cmdio.RenderIterator(ctx, response)
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
