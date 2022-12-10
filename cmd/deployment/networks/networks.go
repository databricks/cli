// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package networks

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "networks",
	Short: `These APIs manage network configurations for customer-managed VPCs (optional).`,
	Long: `These APIs manage network configurations for customer-managed VPCs (optional).
  A network configuration encapsulates the IDs for AWS VPCs, subnets, and
  security groups. Its ID is used when creating a new workspace if you use
  customer-managed VPCs.`,
}

// start create command

var createReq deployment.CreateNetworkRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: gcp_network_info
	createCmd.Flags().StringVar(&createReq.NetworkName, "network-name", createReq.NetworkName, `The human-readable name of the network configuration.`)
	// TODO: array: security_group_ids
	// TODO: array: subnet_ids
	// TODO: complex arg: vpc_endpoints
	createCmd.Flags().StringVar(&createReq.VpcId, "vpc-id", createReq.VpcId, `The ID of the VPC associated with this network.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create network configuration.`,
	Long: `Create network configuration.
  
  Creates a Databricks network configuration that represents an AWS VPC and its
  resources. The VPC will be used for new Databricks clusters. This requires a
  pre-existing VPC and subnets. For VPC requirements, see [Customer-managed
  VPC].
  
  **Important**: You can share one customer-managed VPC with multiple workspaces
  in a single account. Therefore, you can share one VPC across multiple Account
  API network configurations. However, you **cannot** reuse subnets or Security
  Groups between workspaces. Because a Databricks Account API network
  configuration encapsulates this information, you cannot reuse a Databricks
  Account API network configuration across workspaces. If you plan to share one
  VPC with multiple workspaces, make sure you size your VPC and subnets
  accordingly. For information about how to create a new workspace with this
  API, see [Create a new workspace using the Account API].
  
  This operation is available only if your account is on the E2 version of the
  platform.
  
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html
  [Customer-managed VPC]: http://docs.databricks.com/administration-guide/cloud-configurations/aws/customer-managed-vpc.html`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Networks.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq deployment.DeleteNetworkRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.NetworkId, "network-id", deleteReq.NetworkId, `Databricks Account API network configuration ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete network configuration.`,
	Long: `Delete network configuration.
  
  Deletes a Databricks network configuration, which represents an AWS VPC and
  its resources. You cannot delete a network that is associated with a
  workspace.
  
  This operation is available only if your account is on the E2 version of the
  platform.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = a.Networks.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq deployment.GetNetworkRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.NetworkId, "network-id", getReq.NetworkId, `Databricks Account API network configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a network configuration.`,
	Long: `Get a network configuration.
  
  Gets a Databricks network configuration, which represents an AWS VPC and its
  resources. This requires a pre-existing VPC and subnets. For VPC requirements,
  see [Customer-managed VPC].
  
  This operation is available only if your account is on the E2 version of the
  platform.
  
  [Customer-managed VPC]: http://docs.databricks.com/administration-guide/cloud-configurations/aws/customer-managed-vpc.html`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Networks.Get(ctx, getReq)
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
	Short: `Get all network configurations.`,
	Long: `Get all network configurations.
  
  Gets a list of all Databricks network configurations for an account, specified
  by ID.
  
  This operation is available only if your account is on the E2 version of the
  platform.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Networks.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Networks
