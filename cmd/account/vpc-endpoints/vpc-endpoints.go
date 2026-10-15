// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vpc_endpoints

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
		Use:     "vpc-endpoints",
		Short:   `These APIs manage VPC endpoint configurations for this account.`,
		Long:    `These APIs manage VPC endpoint configurations for this account.`,
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
	*networking.CreateVpcEndpointRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq networking.CreateVpcEndpointRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.AwsVpcEndpointId, "aws-vpc-endpoint-id", `The ID of the VPC endpoint object in AWS.`)
	// TODO: complex arg: gcp_vpc_endpoint_info
	flags.StringPtrVar(cmd.Flags(), &createReq.Region, "region", `The region in which this VPC endpoint object exists.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.VpcEndpointName, "vpc-endpoint-name", `The human-readable name of the storage configuration.`)

	cmd.Use = "create"
	cmd.Short = `Create a VPC endpoint configuration.`
	cmd.Long = `Create a VPC endpoint configuration.

  Creates a VPC endpoint configuration, which represents a [VPC endpoint] object
  in AWS used to communicate privately with Databricks over [AWS PrivateLink].

  After you create the VPC endpoint configuration, the Databricks [endpoint
  service] automatically accepts the VPC endpoint.

  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].

  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html
  [endpoint service]: https://docs.aws.amazon.com/vpc/latest/privatelink/privatelink-share-your-services.html`

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

		sdkReq, err := networking.CreateVpcEndpointRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.CreateVpcEndpointPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.VpcEndpointFromSDK(response)
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
	*sdknetworking.DeleteVpcEndpointRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdknetworking.DeleteVpcEndpointRequest

	cmd.Use = "delete VPC_ENDPOINT_ID"
	cmd.Short = `Delete a VPC endpoint configuration.`
	cmd.Long = `Delete a VPC endpoint configuration.

  Deletes a Databricks VPC endpoint configuration. You cannot delete a VPC
  endpoint configuration that is associated with any workspace.`

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

		deleteReq.VpcEndpointId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.Networking.DeleteVpcEndpointPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.VpcEndpointFromSDK(response)
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
	*sdknetworking.GetVpcEndpointRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdknetworking.GetVpcEndpointRequest

	cmd.Use = "get VPC_ENDPOINT_ID"
	cmd.Short = `Get a VPC endpoint configuration.`
	cmd.Long = `Get a VPC endpoint configuration.

  Gets a VPC endpoint configuration, which represents a [VPC endpoint] object in
  AWS used to communicate privately with Databricks over [AWS PrivateLink].

  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html

  Arguments:
    VPC_ENDPOINT_ID: Databricks VPC endpoint ID.`

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

		getReq.VpcEndpointId = new(args[0])

		sdkReq := &getReq
		response, err := a.Networking.GetVpcEndpointPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.VpcEndpointFromSDK(response)
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
	cmd.Short = `List VPC endpoint configurations.`
	cmd.Long = `List VPC endpoint configurations.

  Lists Databricks VPC endpoint configurations for an account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdknetworking.ListVpcEndpointRequest{}
		response, err := a.Networking.ListVpcEndpointPublic(ctx, *sdkReq)
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

// end service VpcEndpoints
