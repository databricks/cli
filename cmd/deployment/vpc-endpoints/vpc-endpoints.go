package vpc_endpoints

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "vpc-endpoints",
	Short: `These APIs manage VPC endpoint configurations for this account.`,
}

var createReq deployment.CreateVpcEndpointRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.AwsVpcEndpointId, "aws-vpc-endpoint-id", "", `The ID of the VPC endpoint object in AWS.`)
	createCmd.Flags().StringVar(&createReq.Region, "region", "", `The AWS region in which this VPC endpoint object exists.`)
	createCmd.Flags().StringVar(&createReq.VpcEndpointName, "vpc-endpoint-name", "", `The human-readable name of the storage configuration.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create VPC endpoint configuration.`,
	Long: `Create VPC endpoint configuration.
  
  Creates a VPC endpoint configuration, which represents a [VPC endpoint] object
  in AWS used to communicate privately with Databricks over [AWS PrivateLink].
  
  **Important**: When you register a VPC endpoint to the Databricks workspace
  VPC endpoint service for any workspace, **in this release Databricks enables
  front-end (web application and REST API) access from the source network of the
  VPC endpoint to all workspaces in that AWS region in your Databricks account
  if the workspaces have any PrivateLink connections in their workspace
  configuration**. If you have questions about this behavior, contact your
  Databricks representative.
  
  Within AWS, your VPC endpoint stays in pendingAcceptance state until you
  register it in a VPC endpoint configuration through the Account API. After you
  register the VPC endpoint configuration, the Databricks [endpoint service]
  automatically accepts the VPC endpoint and it eventually transitions to the
  available state.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html
  [endpoint service]: https://docs.aws.amazon.com/vpc/latest/privatelink/privatelink-share-your-services.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.VpcEndpoints.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq deployment.DeleteVpcEndpointRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.VpcEndpointId, "vpc-endpoint-id", "", `Databricks VPC endpoint ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete VPC endpoint configuration.`,
	Long: `Delete VPC endpoint configuration.
  
  Deletes a VPC endpoint configuration, which represents an [AWS VPC endpoint]
  that can communicate privately with Databricks over [AWS PrivateLink].
  
  Upon deleting a VPC endpoint configuration, the VPC endpoint in AWS changes
  its state from accepted to rejected, which means that it is no longer
  usable from your VPC.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [AWS VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.VpcEndpoints.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetVpcEndpointRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.VpcEndpointId, "vpc-endpoint-id", "", `Databricks VPC endpoint ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a VPC endpoint configuration.`,
	Long: `Get a VPC endpoint configuration.
  
  Gets a VPC endpoint configuration, which represents a [VPC endpoint] object in
  AWS used to communicate privately with Databricks over [AWS PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [VPC endpoint]: https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.VpcEndpoints.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

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
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.VpcEndpoints.List(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
