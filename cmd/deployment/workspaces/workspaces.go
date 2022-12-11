// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspaces

import (
	"fmt"
	"time"

	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/deployment"
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

var createReq deployment.CreateWorkspaceRequest
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
	// TODO: complex arg: cloud_resource_bucket
	createCmd.Flags().StringVar(&createReq.CredentialsId, "credentials-id", createReq.CredentialsId, `ID of the workspace's credential configuration object.`)
	createCmd.Flags().StringVar(&createReq.DeploymentName, "deployment-name", createReq.DeploymentName, `The deployment name defines part of the subdomain for the workspace.`)
	createCmd.Flags().StringVar(&createReq.Location, "location", createReq.Location, `The Google Cloud region of the workspace data plane in your Google account.`)
	createCmd.Flags().StringVar(&createReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", createReq.ManagedServicesCustomerManagedKeyId, `The ID of the workspace's managed services encryption key configuration object.`)
	// TODO: complex arg: network
	createCmd.Flags().StringVar(&createReq.NetworkId, "network-id", createReq.NetworkId, `The ID of the workspace's network configuration object.`)
	createCmd.Flags().Var(&createReq.PricingTier, "pricing-tier", `The pricing tier of the workspace.`)
	createCmd.Flags().StringVar(&createReq.PrivateAccessSettingsId, "private-access-settings-id", createReq.PrivateAccessSettingsId, `ID of the workspace's private access settings object.`)
	createCmd.Flags().StringVar(&createReq.StorageConfigurationId, "storage-configuration-id", createReq.StorageConfigurationId, `The ID of the workspace's storage configuration object.`)
	createCmd.Flags().StringVar(&createReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", createReq.StorageCustomerManagedKeyId, `The ID of the workspace's storage encryption key configuration object.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new workspace.`,
	Long: `Create a new workspace.
  
  Creates a new workspace using a credential configuration and a storage
  configuration, an optional network configuration (if using a customer-managed
  VPC), an optional managed services key configuration (if using
  customer-managed keys for managed services), and an optional storage key
  configuration (if using customer-managed keys for storage). The key
  configurations used for managed services and storage encryption can be the
  same or different.
  
  **Important**: This operation is asynchronous. A response with HTTP status
  code 200 means the request has been accepted and is in progress, but does not
  mean that the workspace deployed successfully and is running. The initial
  workspace status is typically PROVISIONING. Use the workspace ID
  (workspace_id) field in the response to identify the new workspace and make
  repeated GET requests with the workspace ID and check its status. The
  workspace becomes available when the status changes to RUNNING.
  
  You can share one customer-managed VPC with multiple workspaces in a single
  account. It is not required to create a new VPC for each workspace. However,
  you **cannot** reuse subnets or Security Groups between workspaces. If you
  plan to share one VPC with multiple workspaces, make sure you size your VPC
  and subnets accordingly. Because a Databricks Account API network
  configuration encapsulates this information, you cannot reuse a Databricks
  Account API network configuration across workspaces.\nFor information about
  how to create a new workspace with this API **including error handling**, see
  [Create a new workspace using the Account API].
  
  **Important**: Customer-managed VPCs, PrivateLink, and customer-managed keys
  are supported on a limited set of deployment and subscription types. If you
  have questions about availability, contact your Databricks
  representative.\n\nThis operation is available only if your account is on the
  E2 version of the platform or on a select custom plan that allows multiple
  workspaces per account.
  
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if !createNoWait {
			spinner := ui.StartSpinner()
			info, err := a.Workspaces.CreateAndWait(ctx, createReq,
				retries.Timeout[deployment.Workspace](createTimeout),
				func(i *retries.Info[deployment.Workspace]) {
					statusMessage := i.Info.WorkspaceStatusMessage
					spinner.Suffix = " " + statusMessage
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		response, err := a.Workspaces.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq deployment.DeleteWorkspaceRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete WORKSPACE_ID",
	Short: `Delete workspace.`,
	Long: `Delete workspace.
  
  Terminates and deletes a Databricks workspace. From an API perspective,
  deletion is immediate. However, it might take a few minutes for all workspaces
  resources to be deleted, depending on the size and number of workspace
  resources.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &deleteReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = a.Workspaces.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq deployment.GetWorkspaceRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get WORKSPACE_ID",
	Short: `Get workspace.`,
	Long: `Get workspace.
  
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
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &getReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Workspaces.Get(ctx, getReq)
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
	Short: `Get all workspaces.`,
	Long: `Get all workspaces.
  
  Gets a list of all workspaces associated with an account, specified by ID.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Workspaces.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq deployment.UpdateWorkspaceRequest

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
  Network configuration ID. Used only if you use customer-managed VPC. - Key
  configuration ID for managed services (control plane storage, such as notebook
  source and Databricks SQL queries). Used only if you use customer-managed keys
  for managed services. - Key configuration ID for workspace storage (root S3
  bucket and, optionally, EBS volumes). Used only if you use customer-managed
  keys for workspace storage. **Important**: If the workspace was ever in the
  running state, even if briefly before becoming a failed workspace, you cannot
  add a new key configuration ID for workspace storage.
  
  After calling the PATCH operation to update the workspace configuration,
  make repeated GET requests with the workspace ID and check the workspace
  status. The workspace is successful if the status changes to RUNNING.
  
  For information about how to create a new workspace with this API **including
  error handling**, see [Create a new workspace using the Account API].
  
  ### Update a running workspace You can update a Databricks workspace
  configuration for running workspaces for some fields, but not all fields. For
  a running workspace, this request supports updating the following fields only:
  - Credential configuration ID
  
  - Network configuration ID. Used only if you already use use customer-managed
  VPC. This change is supported only if you specified a network configuration ID
  in your original workspace creation. In other words, you cannot switch from a
  Databricks-managed VPC to a customer-managed VPC. **Note**: You cannot use a
  network configuration update in this API to add support for PrivateLink (in
  Public Preview). To add PrivateLink to an existing workspace, contact your
  Databricks representative.
  
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
  configuration for workspace storage.
  
  **Important**: For updating running workspaces, this API is unavailable on
  Mondays, Tuesdays, and Thursdays from 4:30pm-7:30pm PST due to routine
  maintenance. Plan your workspace updates accordingly. For questions about this
  schedule, contact your Databricks representative.
  
  **Important**: To update a running workspace, your workspace must have no
  running cluster instances, which includes all-purpose clusters, job clusters,
  and pools that might have running clusters. Terminate all cluster instances in
  the workspace before calling this API.
  
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
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		_, err = fmt.Sscan(args[0], &updateReq.WorkspaceId)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if !updateNoWait {
			spinner := ui.StartSpinner()
			info, err := a.Workspaces.UpdateAndWait(ctx, updateReq,
				retries.Timeout[deployment.Workspace](updateTimeout),
				func(i *retries.Info[deployment.Workspace]) {
					statusMessage := i.Info.WorkspaceStatusMessage
					spinner.Suffix = " " + statusMessage
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err = a.Workspaces.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Workspaces
