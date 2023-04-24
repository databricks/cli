// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspaces

import (
	"fmt"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspaces",
	Short: `These APIs manage workspaces for this account.`,
	Long: `These APIs manage workspaces for this account. A Databricks workspace is an
  environment for accessing all of your Databricks assets. The workspace
  organizes objects (notebooks, libraries, and experiments) into folders, and
  provides access to data and computational resources such as clusters and jobs.
  
  These endpoints are available if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,
}

// start create command

var createReq provisioning.CreateWorkspaceRequest
var createJson jsonflag.JsonFlag
var createNoWait bool
var createTimeout time.Duration

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createNoWait, "no-wait", createNoWait, `do not wait to reach RUNNING state`)
	createCmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.AwsRegion, "aws-region", createReq.AwsRegion, `The AWS region of the workspace's data plane.`)
	createCmd.Flags().StringVar(&createReq.Cloud, "cloud", createReq.Cloud, `The cloud provider which the workspace uses.`)
	// TODO: complex arg: cloud_resource_container
	createCmd.Flags().StringVar(&createReq.CredentialsId, "credentials-id", createReq.CredentialsId, `ID of the workspace's credential configuration object.`)
	createCmd.Flags().StringVar(&createReq.DeploymentName, "deployment-name", createReq.DeploymentName, `The deployment name defines part of the subdomain for the workspace.`)
	createCmd.Flags().StringVar(&createReq.Location, "location", createReq.Location, `The Google Cloud region of the workspace data plane in your Google account.`)
	createCmd.Flags().StringVar(&createReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", createReq.ManagedServicesCustomerManagedKeyId, `The ID of the workspace's managed services encryption key configuration object.`)
	createCmd.Flags().StringVar(&createReq.NetworkId, "network-id", createReq.NetworkId, ``)
	createCmd.Flags().Var(&createReq.PricingTier, "pricing-tier", `The pricing tier of the workspace.`)
	createCmd.Flags().StringVar(&createReq.PrivateAccessSettingsId, "private-access-settings-id", createReq.PrivateAccessSettingsId, `ID of the workspace's private access settings object.`)
	createCmd.Flags().StringVar(&createReq.StorageConfigurationId, "storage-configuration-id", createReq.StorageConfigurationId, `The ID of the workspace's storage configuration object.`)
	createCmd.Flags().StringVar(&createReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", createReq.StorageCustomerManagedKeyId, `The ID of the workspace's storage encryption key configuration object.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new workspace.`,
	Long: `Create a new workspace.
  
  Creates a new workspace.
  
  **Important**: This operation is asynchronous. A response with HTTP status
  code 200 means the request has been accepted and is in progress, but does not
  mean that the workspace deployed successfully and is running. The initial
  workspace status is typically PROVISIONING. Use the workspace ID
  (workspace_id) field in the response to identify the new workspace and make
  repeated GET requests with the workspace ID and check its status. The
  workspace becomes available when the status changes to RUNNING.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.WorkspaceName = args[0]

		if !createNoWait {
			spinner := ui.StartSpinner()
			info, err := a.Workspaces.CreateAndWait(ctx, createReq,
				retries.Timeout[provisioning.Workspace](createTimeout),
				func(i *retries.Info[provisioning.Workspace]) {
					if i.Info == nil {
						return
					}
					statusMessage := i.Info.WorkspaceStatusMessage
					spinner.Suffix = " " + statusMessage
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, info)
		}
		response, err := a.Workspaces.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq provisioning.DeleteWorkspaceRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete WORKSPACE_ID",
	Short: `Delete a workspace.`,
	Long: `Delete a workspace.
  
  Terminates and deletes a Databricks workspace. From an API perspective,
  deletion is immediate. However, it might take a few minutes for all workspaces
  resources to be deleted, depending on the size and number of workspace
  resources.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Workspaces.WorkspaceWorkspaceNameToWorkspaceIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Workspace ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have workspace id")
		}
		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		err = a.Workspaces.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq provisioning.GetWorkspaceRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get WORKSPACE_ID",
	Short: `Get a workspace.`,
	Long: `Get a workspace.
  
  Gets information including status for a Databricks workspace, specified by ID.
  In the response, the workspace_status field indicates the current status.
  After initial workspace creation (which is asynchronous), make repeated GET
  requests with the workspace ID and check its status. The workspace becomes
  available when the status changes to RUNNING.
  
  For information about how to create a new workspace with this API **including
  error handling**, see [Create a new workspace using the Account API].
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.
  
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Workspaces.WorkspaceWorkspaceNameToWorkspaceIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Workspace ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have workspace id")
		}
		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		response, err := a.Workspaces.Get(ctx, getReq)
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
	Short: `Get all workspaces.`,
	Long: `Get all workspaces.
  
  Gets a list of all workspaces associated with an account, specified by ID.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.Workspaces.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq provisioning.UpdateWorkspaceRequest

var updateNoWait bool
var updateTimeout time.Duration

func init() {
	Cmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&updateNoWait, "no-wait", updateNoWait, `do not wait to reach RUNNING state`)
	updateCmd.Flags().DurationVar(&updateTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.AwsRegion, "aws-region", updateReq.AwsRegion, `The AWS region of the workspace's data plane (for example, us-west-2).`)
	updateCmd.Flags().StringVar(&updateReq.CredentialsId, "credentials-id", updateReq.CredentialsId, `ID of the workspace's credential configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", updateReq.ManagedServicesCustomerManagedKeyId, `The ID of the workspace's managed services encryption key configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.NetworkId, "network-id", updateReq.NetworkId, `The ID of the workspace's network configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.StorageConfigurationId, "storage-configuration-id", updateReq.StorageConfigurationId, `The ID of the workspace's storage configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", updateReq.StorageCustomerManagedKeyId, `The ID of the key configuration object for workspace storage.`)

}

var updateCmd = &cobra.Command{
	Use:   "update WORKSPACE_ID",
	Short: `Update workspace configuration.`,
	Long: `Update workspace configuration.
  
  Updates a workspace configuration for either a running workspace or a failed
  workspace. The elements that can be updated varies between these two use
  cases.
  
  ### Update a failed workspace You can update a Databricks workspace
  configuration for failed workspace deployment for some fields, but not all
  fields. For a failed workspace, this request supports updates to the following
  fields only: - Credential configuration ID - Storage configuration ID -
  Network configuration ID. Used only to add or change a network configuration
  for a customer-managed VPC. For a failed workspace only, you can convert a
  workspace with Databricks-managed VPC to use a customer-managed VPC by adding
  this ID. You cannot downgrade a workspace with a customer-managed VPC to be a
  Databricks-managed VPC. You can update the network configuration for a failed
  or running workspace to add PrivateLink support, though you must also add a
  private access settings object. - Key configuration ID for managed services
  (control plane storage, such as notebook source and Databricks SQL queries).
  Used only if you use customer-managed keys for managed services. - Key
  configuration ID for workspace storage (root S3 bucket and, optionally, EBS
  volumes). Used only if you use customer-managed keys for workspace storage.
  **Important**: If the workspace was ever in the running state, even if briefly
  before becoming a failed workspace, you cannot add a new key configuration ID
  for workspace storage. - Private access settings ID to add PrivateLink
  support. You can add or update the private access settings ID to upgrade a
  workspace to add support for front-end, back-end, or both types of
  connectivity. You cannot remove (downgrade) any existing front-end or back-end
  PrivateLink support on a workspace.
  
  After calling the PATCH operation to update the workspace configuration,
  make repeated GET requests with the workspace ID and check the workspace
  status. The workspace is successful if the status changes to RUNNING.
  
  For information about how to create a new workspace with this API **including
  error handling**, see [Create a new workspace using the Account API].
  
  ### Update a running workspace You can update a Databricks workspace
  configuration for running workspaces for some fields, but not all fields. For
  a running workspace, this request supports updating the following fields only:
  - Credential configuration ID
  
  - Network configuration ID. Used only if you already use a customer-managed
  VPC. You cannot convert a running workspace from a Databricks-managed VPC to a
  customer-managed VPC. You can use a network configuration update in this API
  for a failed or running workspace to add support for PrivateLink, although you
  also need to add a private access settings object.
  
  - Key configuration ID for managed services (control plane storage, such as
  notebook source and Databricks SQL queries). Databricks does not directly
  encrypt the data with the customer-managed key (CMK). Databricks uses both the
  CMK and the Databricks managed key (DMK) that is unique to your workspace to
  encrypt the Data Encryption Key (DEK). Databricks uses the DEK to encrypt your
  workspace's managed services persisted data. If the workspace does not already
  have a CMK for managed services, adding this ID enables managed services
  encryption for new or updated data. Existing managed services data that
  existed before adding the key remains not encrypted with the DEK until it is
  modified. If the workspace already has customer-managed keys for managed
  services, this request rotates (changes) the CMK keys and the DEK is
  re-encrypted with the DMK and the new CMK. - Key configuration ID for
  workspace storage (root S3 bucket and, optionally, EBS volumes). You can set
  this only if the workspace does not already have a customer-managed key
  configuration for workspace storage. - Private access settings ID to add
  PrivateLink support. You can add or update the private access settings ID to
  upgrade a workspace to add support for front-end, back-end, or both types of
  connectivity. You cannot remove (downgrade) any existing front-end or back-end
  PrivateLink support on a workspace.
  
  **Important**: To update a running workspace, your workspace must have no
  running compute resources that run in your workspace's VPC in the Classic data
  plane. For example, stop all all-purpose clusters, job clusters, pools with
  running clusters, and Classic SQL warehouses. If you do not terminate all
  cluster instances in the workspace before calling this API, the request will
  fail.
  
  ### Wait until changes take effect. After calling the PATCH operation to
  update the workspace configuration, make repeated GET requests with the
  workspace ID and check the workspace status and the status of the fields. *
  For workspaces with a Databricks-managed VPC, the workspace status becomes
  PROVISIONING temporarily (typically under 20 minutes). If the workspace
  update is successful, the workspace status changes to RUNNING. Note that you
  can also check the workspace status in the [Account Console]. However, you
  cannot use or create clusters for another 20 minutes after that status change.
  This results in a total of up to 40 minutes in which you cannot create
  clusters. If you create or use clusters before this time interval elapses,
  clusters do not launch successfully, fail, or could cause other unexpected
  behavior.
  
  * For workspaces with a customer-managed VPC, the workspace status stays at
  status RUNNING and the VPC change happens immediately. A change to the
  storage customer-managed key configuration ID might take a few minutes to
  update, so continue to check the workspace until you observe that it has been
  updated. If the update fails, the workspace might revert silently to its
  original configuration. After the workspace has been updated, you cannot use
  or create clusters for another 20 minutes. If you create or use clusters
  before this time interval elapses, clusters do not launch successfully, fail,
  or could cause other unexpected behavior.
  
  If you update the _storage_ customer-managed key configurations, it takes 20
  minutes for the changes to fully take effect. During the 20 minute wait, it is
  important that you stop all REST API calls to the DBFS API. If you are
  modifying _only the managed services key configuration_, you can omit the 20
  minute wait.
  
  **Important**: Customer-managed keys and customer-managed VPCs are supported
  by only some deployment types and subscription types. If you have questions
  about availability, contact your Databricks representative.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.
  
  [Account Console]: https://docs.databricks.com/administration-guide/account-settings-e2/account-console-e2.html
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Workspaces.WorkspaceWorkspaceNameToWorkspaceIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Workspace ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have workspace id")
		}
		_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}

		if !updateNoWait {
			spinner := ui.StartSpinner()
			info, err := a.Workspaces.UpdateAndWait(ctx, updateReq,
				retries.Timeout[provisioning.Workspace](updateTimeout),
				func(i *retries.Info[provisioning.Workspace]) {
					if i.Info == nil {
						return
					}
					statusMessage := i.Info.WorkspaceStatusMessage
					spinner.Suffix = " " + statusMessage
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, info)
		}
		err = a.Workspaces.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Workspaces
