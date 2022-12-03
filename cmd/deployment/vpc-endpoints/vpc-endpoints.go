package vpc_endpoints

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "vpc-endpoints",
	Short: `These APIs manage VPC endpoint configurations for this account.`, // TODO: fix FirstSentence logic and append dot to summary
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
	Short: `Create VPC endpoint configuration Creates a VPC endpoint configuration, which represents a [VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html) object in AWS used to communicate privately with Databricks over [AWS PrivateLink](https://aws.amazon.com/privatelink).`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
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
	Short: `Delete VPC endpoint configuration Deletes a VPC endpoint configuration, which represents an [AWS VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html) that can communicate privately with Databricks over [AWS PrivateLink](https://aws.amazon.com/privatelink).`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
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
	Short: `Get a VPC endpoint configuration Gets a VPC endpoint configuration, which represents a [VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html) object in AWS used to communicate privately with Databricks over [AWS PrivateLink](https://aws.amazon.com/privatelink).`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
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
	Short: `Get all VPC endpoint configurations Gets a list of all VPC endpoints for an account, specified by ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
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
