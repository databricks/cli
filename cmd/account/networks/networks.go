// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package networks

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "networks",
	Short: `These APIs manage network configurations for customer-managed VPCs (optional).`,
	Long: `These APIs manage network configurations for customer-managed VPCs (optional).
  Its ID is used when creating a new workspace if you use customer-managed VPCs.`,
}

// start create command

var createReq provisioning.CreateNetworkRequest
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: gcp_network_info
	// TODO: array: security_group_ids
	// TODO: array: subnet_ids
	// TODO: complex arg: vpc_endpoints
	createCmd.Flags().StringVar(&createReq.VpcId, "vpc-id", createReq.VpcId, `The ID of the VPC associated with this network.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create network configuration.`,
	Long: `Create network configuration.
  
  Creates a Databricks network configuration that represents an VPC and its
  resources. The VPC will be used for new Databricks clusters. This requires a
  pre-existing VPC and subnets.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshal(&createReq)
		if err != nil {
			return err
		}
		createReq.NetworkName = args[0]

		response, err := a.Networks.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq provisioning.DeleteNetworkRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NETWORK_ID",
	Short: `Delete a network configuration.`,
	Long: `Delete a network configuration.
  
  Deletes a Databricks network configuration, which represents a cloud VPC and
  its resources. You cannot delete a network that is associated with a
  workspace.
  
  This operation is available only if your account is on the E2 version of the
  platform.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Networks.NetworkNetworkNameToNetworkIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API network configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api network configuration id")
		}
		deleteReq.NetworkId = args[0]

		err = a.Networks.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq provisioning.GetNetworkRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NETWORK_ID",
	Short: `Get a network configuration.`,
	Long: `Get a network configuration.
  
  Gets a Databricks network configuration, which represents a cloud VPC and its
  resources.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Networks.NetworkNetworkNameToNetworkIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API network configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api network configuration id")
		}
		getReq.NetworkId = args[0]

		response, err := a.Networks.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.Networks.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Networks
