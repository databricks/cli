// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vpc_endpoints

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "vpc-endpoints",
	Short: `These APIs manage VPC endpoint configurations for this account.`,
	Long:  `These APIs manage VPC endpoint configurations for this account.`,
}

// start create command

var createReq provisioning.CreateVpcEndpointRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.AwsVpcEndpointId, "aws-vpc-endpoint-id", createReq.AwsVpcEndpointId, `The ID of the VPC endpoint object in AWS.`)
	// TODO: complex arg: gcp_vpc_endpoint_info
	createCmd.Flags().StringVar(&createReq.Region, "region", createReq.Region, `The AWS region in which this VPC endpoint object exists.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create VPC endpoint configuration.`,
	Long: `Create VPC endpoint configuration.
  
  Creates a VPC endpoint configuration, which represents a [VPC endpoint] object
  in AWS used to communicate privately with Databricks over [AWS PrivateLink].
  
  After you create the VPC endpoint configuration, the Databricks [endpoint
  service] automatically accepts the VPC endpoint.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html
  [endpoint service]: https://docs.aws.amazon.com/vpc/latest/privatelink/privatelink-share-your-services.html`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.VpcEndpointName = args[0]

		response, err := a.VpcEndpoints.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq provisioning.DeleteVpcEndpointRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete VPC_ENDPOINT_ID",
	Short: `Delete VPC endpoint configuration.`,
	Long: `Delete VPC endpoint configuration.
  
  Deletes a VPC endpoint configuration, which represents an [AWS VPC endpoint]
  that can communicate privately with Databricks over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [AWS VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.VpcEndpoints.VpcEndpointVpcEndpointNameToVpcEndpointIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Databricks VPC endpoint ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks vpc endpoint id")
		}
		deleteReq.VpcEndpointId = args[0]

		err = a.VpcEndpoints.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq provisioning.GetVpcEndpointRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get VPC_ENDPOINT_ID",
	Short: `Get a VPC endpoint configuration.`,
	Long: `Get a VPC endpoint configuration.
  
  Gets a VPC endpoint configuration, which represents a [VPC endpoint] object in
  AWS used to communicate privately with Databricks over [AWS PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.VpcEndpoints.VpcEndpointVpcEndpointNameToVpcEndpointIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Databricks VPC endpoint ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks vpc endpoint id")
		}
		getReq.VpcEndpointId = args[0]

		response, err := a.VpcEndpoints.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all VPC endpoint configurations.`,
	Long: `Get all VPC endpoint configurations.
  
  Gets a list of all VPC endpoints for an account, specified by ID.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.VpcEndpoints.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service VpcEndpoints
