// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspaces

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	workspaces "github.com/databricks/cli/libs/sdktypes/workspaces/v1"
	sdkworkspaces "github.com/databricks/sdk-go/workspaces/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: `These APIs manage workspaces for this account.`,
		Long: `These APIs manage workspaces for this account. A Databricks workspace is an
  environment for accessing all of your Databricks assets. The workspace
  organizes objects (notebooks, libraries, and experiments) into folders, and
  provides access to data and computational resources such as clusters and jobs.

  These endpoints are available if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,
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
	cmd.AddCommand(newUpdate())

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
	*workspaces.CreateWorkspaceRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq workspaces.CreateWorkspaceRequest

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.AwsRegion, "aws-region", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.Cloud, "cloud", `DEPRECATED: This field is being ignored by the server and will be removed in the future.`)
	// TODO: complex arg: cloud_resource_container
	flags.EnumVar(cmd.Flags(), &createReq.ComputeMode, "compute-mode", `If the compute mode is SERVERLESS, a serverless workspace is created that comes pre-configured with serverless compute and default storage, providing a fully-managed, enterprise-ready SaaS experience. Supported values: [HYBRID, SERVERLESS]`, "CustomerFacingComputeMode", "HYBRID", "SERVERLESS")
	flags.StringPtrVar(cmd.Flags(), &createReq.CredentialsId, "credentials-id", `ID of the workspace's credential configuration object.`)
	// TODO: map via StringToStringVar: custom_tags
	flags.StringPtrVar(cmd.Flags(), &createReq.DeploymentName, "deployment-name", `The deployment name defines part of the subdomain for the workspace.`)
	// TODO: complex arg: gcp_managed_network_config
	// TODO: complex arg: gke_config
	flags.StringPtrVar(cmd.Flags(), &createReq.Location, "location", `The Google Cloud region of the workspace data plane in your Google account (for example, us-east4).`)
	flags.StringPtrVar(cmd.Flags(), &createReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", `The ID of the workspace's managed services encryption key configuration object.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.NetworkConnectivityConfigId, "network-connectivity-config-id", `The object ID of network connectivity config.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.NetworkId, "network-id", `The ID of the workspace's network configuration object.`)
	flags.EnumVar(cmd.Flags(), &createReq.PricingTier, "pricing-tier", `Supported values: [
  COMMUNITY_EDITION,
  DEDICATED,
  ENTERPRISE,
  PREMIUM,
  STANDARD,
  UNKNOWN,
]`, "PricingTier", "COMMUNITY_EDITION", "DEDICATED", "ENTERPRISE", "PREMIUM", "STANDARD", "UNKNOWN")
	flags.StringPtrVar(cmd.Flags(), &createReq.PrivateAccessSettingsId, "private-access-settings-id", `ID of the workspace's private access settings object.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageConfigurationId, "storage-configuration-id", `ID of the workspace's storage configuration object.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", `The ID of the workspace's storage encryption key configuration object.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.WorkspaceName, "workspace-name", `The human-readable name of the workspace.`)

	cmd.Use = "create"
	cmd.Short = `Create a workspace.`
	cmd.Long = `Create a workspace.

  Creates a new workspace using a credential configuration and a storage
  configuration, an optional network configuration (if using a customer-managed
  VPC), an optional managed services key configuration (if using
  customer-managed keys for managed services), and an optional storage key
  configuration (if using customer-managed keys for storage). The key
  configurations used for managed services and storage encryption can be the
  same or different.

  Important: This operation is asynchronous. A response with HTTP status code
  200 means the request has been accepted and is in progress, but does not mean
  that the workspace deployed successfully and is running. The initial workspace
  status is typically PROVISIONING. Use the workspace ID (workspace_id) field in
  the response to identify the new workspace and make repeated GET requests with
  the workspace ID and check its status. The workspace becomes available when
  the status changes to RUNNING.

  You can share one customer-managed VPC with multiple workspaces in a single
  account. It is not required to create a new VPC for each workspace. However,
  you cannot reuse subnets or Security Groups between workspaces. If you plan to
  share one VPC with multiple workspaces, make sure you size your VPC and
  subnets accordingly. Because a Databricks Account API network configuration
  encapsulates this information, you cannot reuse a Databricks Account API
  network configuration across workspaces.

  For information about how to create a new workspace with this API including
  error handling, see [Create a new workspace using the Account API].

  Important: Customer-managed VPCs, PrivateLink, and customer-managed keys are
  supported on a limited set of deployment and subscription types. If you have
  questions about availability, contact your Databricks representative.

  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.

  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`

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

		sdkReq, err := workspaces.CreateWorkspaceRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		wait, err := a.Workspaces.CreateWorkspacePublic(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, map[string]any{"workspace_id": wait.GetWorkspaceId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := workspaces.WorkspaceFromSDK(response)
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
	*sdkworkspaces.DeleteWorkspaceRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkworkspaces.DeleteWorkspaceRequest

	cmd.Use = "delete WORKSPACE_ID"
	cmd.Short = `Delete a workspace.`
	cmd.Long = `Delete a workspace.

  Deletes a Databricks workspace, both specified by ID.`

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

		var deleteWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &deleteWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		deleteReq.WorkspaceId = &deleteWorkspaceIdArg

		sdkReq := &deleteReq
		response, err := a.Workspaces.DeleteWorkspacePublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := workspaces.WorkspaceFromSDK(response)
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
	*sdkworkspaces.GetWorkspaceRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkworkspaces.GetWorkspaceRequest

	cmd.Use = "get WORKSPACE_ID"
	cmd.Short = `Get a workspace.`
	cmd.Long = `Get a workspace.

  Gets information including status for a Databricks workspace, specified by ID.
  In the response, the workspace_status field indicates the current status.
  After initial workspace creation (which is asynchronous), make repeated GET
  requests with the workspace ID and check its status. The workspace becomes
  available when the status changes to RUNNING. For information about how to
  create a new workspace with this API **including error handling**, see [Create
  a new workspace using the Account API].

  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`

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

		var getWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &getWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		getReq.WorkspaceId = &getWorkspaceIdArg

		sdkReq := &getReq
		response, err := a.Workspaces.GetWorkspacePublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := workspaces.WorkspaceFromSDK(response)
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
	cmd.Short = `List workspaces.`
	cmd.Long = `List workspaces.

  Lists Databricks workspaces for an account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdkworkspaces.ListWorkspacesRequest{}
		response, err := a.Workspaces.ListWorkspacesPublic(ctx, *sdkReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*workspaces.UpdateWorkspaceRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq workspaces.UpdateWorkspaceRequest
	updateReq.CustomerFacingWorkspace = &workspaces.Workspace{}

	var updateJson flags.JsonFlag

	var updateSkipWait bool
	var updateTimeout time.Duration

	cmd.Flags().BoolVar(&updateSkipWait, "no-wait", updateSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&updateTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.UpdateMask, "update-mask", `The field mask must be a single string, with multiple fields separated by commas (no spaces).`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.AwsRegion, "aws-region", ``)
	// TODO: complex arg: azure_workspace_info
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.Cloud, "cloud", `The cloud name.`)
	// TODO: complex arg: cloud_resource_container
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.CredentialsId, "credentials-id", `ID of the workspace's credential configuration object.`)
	// TODO: map via StringToStringVar: custom_tags
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.DeploymentName, "deployment-name", ``)
	flags.EnumVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.ExpectedWorkspaceStatus, "expected-workspace-status", `A client owned field used to indicate the workspace status that the client expects to be in. Supported values: [
  BANNED,
  CANCELLING,
  FAILED,
  NOT_PROVISIONED,
  PROVISIONING,
  RUNNING,
]`, "WorkspaceStatus", "BANNED", "CANCELLING", "FAILED", "NOT_PROVISIONED", "PROVISIONING", "RUNNING")
	// TODO: complex arg: gcp_managed_network_config
	// TODO: complex arg: gke_config
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.Location, "location", `The Google Cloud region of the workspace data plane in your Google account (for example, us-east4).`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", `ID of the key configuration for encrypting managed services.`)
	// TODO: complex arg: network
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.NetworkConnectivityConfigId, "network-connectivity-config-id", `The object ID of network connectivity config.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.NetworkId, "network-id", `If this workspace is BYO VPC, then the network_id will be populated.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.PrivateAccessSettingsId, "private-access-settings-id", `ID of the workspace's private access settings object.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.StorageConfigurationId, "storage-configuration-id", `ID of the workspace's storage configuration object.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", `ID of the key configuration for encrypting workspace storage.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CustomerFacingWorkspace.WorkspaceName, "workspace-name", `The human-readable name of the workspace.`)

	cmd.Use = "update WORKSPACE_ID"
	cmd.Short = `Update a workspace.`
	cmd.Long = `Update a workspace.

  Updates a workspace.

  Arguments:
    WORKSPACE_ID: A unique integer ID for the workspace`

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

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.CustomerFacingWorkspace)
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
		var updateWorkspaceIdArg int64
		_, err = fmt.Sscan(args[0], &updateWorkspaceIdArg)
		if err != nil {
			return fmt.Errorf("invalid WORKSPACE_ID: %s", args[0])
		}
		updateReq.CustomerFacingWorkspace.WorkspaceId = &updateWorkspaceIdArg

		sdkReq, err := workspaces.UpdateWorkspaceRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		wait, err := a.Workspaces.UpdateWorkspacePublic(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if updateSkipWait {
			return cmdio.Render(ctx, map[string]any{"workspace_id": wait.GetWorkspaceId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for update to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(updateTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := workspaces.WorkspaceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service Workspaces
