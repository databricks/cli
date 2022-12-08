package networks

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "networks",
	Short: `These APIs manage network configurations for customer-managed VPCs (optional).`,
}

var createReq deployment.CreateNetworkRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: gcp_network_info
	createCmd.Flags().StringVar(&createReq.NetworkName, "network-name", "", `The human-readable name of the network configuration.`)
	// TODO: array: security_group_ids
	// TODO: array: subnet_ids
	// TODO: complex arg: vpc_endpoints
	createCmd.Flags().StringVar(&createReq.VpcId, "vpc-id", "", `The ID of the VPC associated with this network.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create network configuration.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Networks.Create(ctx, createReq)
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

var deleteReq deployment.DeleteNetworkRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.NetworkId, "network-id", "", `Databricks Account API network configuration ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete network configuration.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.Networks.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetNetworkRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.NetworkId, "network-id", "", `Databricks Account API network configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a network configuration.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Networks.Get(ctx, getReq)
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
	Short: `Get all network configurations.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Networks.List(ctx)
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
