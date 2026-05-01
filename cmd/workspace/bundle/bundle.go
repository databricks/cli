// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package bundle

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   `Service for managing bundle deployment metadata.`,
		Long:    `Service for managing bundle deployment metadata.`,
		GroupID: "bundle",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCompleteVersion())
	cmd.AddCommand(newCreateDeployment())
	cmd.AddCommand(newCreateOperation())
	cmd.AddCommand(newCreateVersion())
	cmd.AddCommand(newDeleteDeployment())
	cmd.AddCommand(newGetDeployment())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newGetResource())
	cmd.AddCommand(newGetVersion())
	cmd.AddCommand(newHeartbeat())
	cmd.AddCommand(newListDeployments())
	cmd.AddCommand(newListOperations())
	cmd.AddCommand(newListResources())
	cmd.AddCommand(newListVersions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start complete-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var completeVersionOverrides []func(
	*cobra.Command,
	*bundle.CompleteVersionRequest,
)

func newCompleteVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var completeVersionReq bundle.CompleteVersionRequest
	var completeVersionJson flags.JsonFlag

	cmd.Flags().Var(&completeVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&completeVersionReq.Force, "force", completeVersionReq.Force, `If true, force-completes the version even if the caller is not the original creator.`)

	cmd.Use = "complete-version NAME COMPLETION_REASON"
	cmd.Short = `Complete a version.`
	cmd.Long = `Complete a version.
  
  Marks a version as complete and releases the deployment lock.
  
  The server atomically: 1. Sets the version status to the provided terminal
  status. 2. Sets complete_time to the current server timestamp. 3. Releases
  the lock on the parent deployment. 4. Updates the parent deployment's status
  and last_version_id.

  Arguments:
    NAME: The name of the version to complete. Format:
      deployments/{deployment_id}/versions/{version_id}
    COMPLETION_REASON: The reason for completing the version. Must be a terminal reason:
      VERSION_COMPLETE_SUCCESS, VERSION_COMPLETE_FAILURE, or
      VERSION_COMPLETE_FORCE_ABORT. 
      Supported values: [VERSION_COMPLETE_FAILURE, VERSION_COMPLETE_FORCE_ABORT, VERSION_COMPLETE_LEASE_EXPIRED, VERSION_COMPLETE_SUCCESS]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'completion_reason' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := completeVersionJson.Unmarshal(&completeVersionReq)
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
		completeVersionReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &completeVersionReq.CompletionReason)
			if err != nil {
				return fmt.Errorf("invalid COMPLETION_REASON: %s", args[1])
			}

		}

		response, err := w.Bundle.CompleteVersion(ctx, completeVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range completeVersionOverrides {
		fn(cmd, &completeVersionReq)
	}

	return cmd
}

// start create-deployment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDeploymentOverrides []func(
	*cobra.Command,
	*bundle.CreateDeploymentRequest,
)

func newCreateDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var createDeploymentReq bundle.CreateDeploymentRequest
	createDeploymentReq.Deployment = bundle.Deployment{}
	var createDeploymentJson flags.JsonFlag

	cmd.Flags().Var(&createDeploymentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDeploymentReq.Deployment.DisplayName, "display-name", createDeploymentReq.Deployment.DisplayName, `Human-readable name for the deployment.`)
	cmd.Flags().StringVar(&createDeploymentReq.Deployment.TargetName, "target-name", createDeploymentReq.Deployment.TargetName, `The bundle target name associated with this deployment.`)

	cmd.Use = "create-deployment DEPLOYMENT_ID"
	cmd.Short = `Create a deployment.`
	cmd.Long = `Create a deployment.
  
  Creates a new deployment in the workspace.
  
  The caller must provide a deployment_id which becomes the final component of
  the deployment's resource name. If a deployment with the same ID already
  exists, the server returns ALREADY_EXISTS.

  Arguments:
    DEPLOYMENT_ID: The ID to use for the deployment, which will become the final component of
      the deployment's resource name (i.e. deployments/{deployment_id}).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDeploymentJson.Unmarshal(&createDeploymentReq.Deployment)
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
		createDeploymentReq.DeploymentId = args[0]

		response, err := w.Bundle.CreateDeployment(ctx, createDeploymentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDeploymentOverrides {
		fn(cmd, &createDeploymentReq)
	}

	return cmd
}

// start create-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOperationOverrides []func(
	*cobra.Command,
	*bundle.CreateOperationRequest,
)

func newCreateOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var createOperationReq bundle.CreateOperationRequest
	createOperationReq.Operation = bundle.Operation{}
	var createOperationJson flags.JsonFlag

	cmd.Flags().Var(&createOperationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-operation PARENT"
	cmd.Short = `Create an operation.`
	cmd.Long = `Create an operation.
  
  Creates a resource operation under a version.
  
  The caller must provide a resource_key which becomes the final component of
  the operation's name. If an operation with the same key already exists under
  the version, the server returns ALREADY_EXISTS.
  
  On success the server also updates the corresponding deployment-level Resource
  (creating it if this is the first operation for that resource_key, or removing
  it if action_type is DELETE).

  Arguments:
    PARENT: The parent version where this operation will be recorded. Format:
      deployments/{deployment_id}/versions/{version_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, RESOURCE_KEY as positional arguments. Provide 'status' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createOperationJson.Unmarshal(&createOperationReq.Operation)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		createOperationReq.Parent = args[0]

		response, err := w.Bundle.CreateOperation(ctx, createOperationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOperationOverrides {
		fn(cmd, &createOperationReq)
	}

	return cmd
}

// start create-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createVersionOverrides []func(
	*cobra.Command,
	*bundle.CreateVersionRequest,
)

func newCreateVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var createVersionReq bundle.CreateVersionRequest
	createVersionReq.Version = bundle.Version{}
	var createVersionJson flags.JsonFlag

	cmd.Flags().Var(&createVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createVersionReq.Version.CliVersion, "cli-version", createVersionReq.Version.CliVersion, `CLI version used to initiate the version.`)
	cmd.Flags().StringVar(&createVersionReq.Version.DisplayName, "display-name", createVersionReq.Version.DisplayName, `Display name for the deployment, captured at the time of this version.`)
	cmd.Flags().StringVar(&createVersionReq.Version.TargetName, "target-name", createVersionReq.Version.TargetName, `Target name of the deployment, captured at the time of this version.`)

	cmd.Use = "create-version PARENT VERSION_ID VERSION_TYPE"
	cmd.Short = `Create a version.`
	cmd.Long = `Create a version.
  
  Creates a new version under a deployment.
  
  Creating a version acquires an exclusive lock on the deployment, preventing
  concurrent deploys. The caller provides a version_id which the server
  validates equals last_version_id + 1 on the deployment.

  Arguments:
    PARENT: The parent deployment where this version will be created. Format:
      deployments/{deployment_id}
    VERSION_ID: The version ID the caller expects to create. The server validates this
      equals last_version_id + 1 on the deployment. If it doesn't match, the
      server returns ABORTED.
    VERSION_TYPE: Type of version (deploy or destroy). 
      Supported values: [VERSION_TYPE_DEPLOY, VERSION_TYPE_DESTROY]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, VERSION_ID as positional arguments. Provide 'version_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createVersionJson.Unmarshal(&createVersionReq.Version)
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
		createVersionReq.Parent = args[0]
		createVersionReq.VersionId = args[1]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createVersionReq.Version.VersionType)
			if err != nil {
				return fmt.Errorf("invalid VERSION_TYPE: %s", args[2])
			}

		}

		response, err := w.Bundle.CreateVersion(ctx, createVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createVersionOverrides {
		fn(cmd, &createVersionReq)
	}

	return cmd
}

// start delete-deployment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDeploymentOverrides []func(
	*cobra.Command,
	*bundle.DeleteDeploymentRequest,
)

func newDeleteDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDeploymentReq bundle.DeleteDeploymentRequest

	cmd.Use = "delete-deployment NAME"
	cmd.Short = `Delete a deployment.`
	cmd.Long = `Delete a deployment.
  
  Deletes a deployment.
  
  The deployment is marked as deleted. It and all its children (versions and
  their operations) will be permanently deleted after the retention policy
  expires. If the deployment has an in-progress version, the server returns
  RESOURCE_CONFLICT.

  Arguments:
    NAME: Resource name of the deployment to delete. Format:
      deployments/{deployment_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDeploymentReq.Name = args[0]

		err = w.Bundle.DeleteDeployment(ctx, deleteDeploymentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDeploymentOverrides {
		fn(cmd, &deleteDeploymentReq)
	}

	return cmd
}

// start get-deployment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDeploymentOverrides []func(
	*cobra.Command,
	*bundle.GetDeploymentRequest,
)

func newGetDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var getDeploymentReq bundle.GetDeploymentRequest

	cmd.Use = "get-deployment NAME"
	cmd.Short = `Get a deployment.`
	cmd.Long = `Get a deployment.
  
  Retrieves a deployment by its resource name.

  Arguments:
    NAME: Resource name of the deployment to retrieve. Format:
      deployments/{deployment_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDeploymentReq.Name = args[0]

		response, err := w.Bundle.GetDeployment(ctx, getDeploymentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDeploymentOverrides {
		fn(cmd, &getDeploymentReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*bundle.GetOperationRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getOperationReq bundle.GetOperationRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get an operation.`
	cmd.Long = `Get an operation.
  
  Retrieves a resource operation by its resource name.

  Arguments:
    NAME: The name of the resource operation to retrieve. Format:
      deployments/{deployment_id}/versions/{version_id}/operations/{resource_key}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getOperationReq.Name = args[0]

		response, err := w.Bundle.GetOperation(ctx, getOperationReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// start get-resource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getResourceOverrides []func(
	*cobra.Command,
	*bundle.GetResourceRequest,
)

func newGetResource() *cobra.Command {
	cmd := &cobra.Command{}

	var getResourceReq bundle.GetResourceRequest

	cmd.Use = "get-resource NAME"
	cmd.Short = `Get a resource.`
	cmd.Long = `Get a resource.
  
  Retrieves a deployment resource by its resource name.

  Arguments:
    NAME: The name of the resource to retrieve. Format:
      deployments/{deployment_id}/resources/{resource_key}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getResourceReq.Name = args[0]

		response, err := w.Bundle.GetResource(ctx, getResourceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getResourceOverrides {
		fn(cmd, &getResourceReq)
	}

	return cmd
}

// start get-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getVersionOverrides []func(
	*cobra.Command,
	*bundle.GetVersionRequest,
)

func newGetVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var getVersionReq bundle.GetVersionRequest

	cmd.Use = "get-version NAME"
	cmd.Short = `Get a version.`
	cmd.Long = `Get a version.
  
  Retrieves a version by its resource name.

  Arguments:
    NAME: The name of the version to retrieve. Format:
      deployments/{deployment_id}/versions/{version_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getVersionReq.Name = args[0]

		response, err := w.Bundle.GetVersion(ctx, getVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getVersionOverrides {
		fn(cmd, &getVersionReq)
	}

	return cmd
}

// start heartbeat command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var heartbeatOverrides []func(
	*cobra.Command,
	*bundle.HeartbeatRequest,
)

func newHeartbeat() *cobra.Command {
	cmd := &cobra.Command{}

	var heartbeatReq bundle.HeartbeatRequest

	cmd.Use = "heartbeat NAME"
	cmd.Short = `Send a version heartbeat.`
	cmd.Long = `Send a version heartbeat.
  
  Sends a heartbeat to renew the lock held by a version.
  
  The server validates that the version is the active (non-terminal) version on
  the parent deployment and resets the lock expiry. If the lock has already
  expired or the version is no longer active, the server returns ABORTED.

  Arguments:
    NAME: The version whose lock to renew. Format:
      deployments/{deployment_id}/versions/{version_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		heartbeatReq.Name = args[0]

		response, err := w.Bundle.Heartbeat(ctx, heartbeatReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range heartbeatOverrides {
		fn(cmd, &heartbeatReq)
	}

	return cmd
}

// start list-deployments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDeploymentsOverrides []func(
	*cobra.Command,
	*bundle.ListDeploymentsRequest,
)

func newListDeployments() *cobra.Command {
	cmd := &cobra.Command{}

	var listDeploymentsReq bundle.ListDeploymentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDeploymentsLimit int

	cmd.Flags().IntVar(&listDeploymentsReq.PageSize, "page-size", listDeploymentsReq.PageSize, `The maximum number of deployments to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDeploymentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDeploymentsReq.PageToken, "page-token", listDeploymentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-deployments"
	cmd.Short = `List deployments.`
	cmd.Long = `List deployments.
  
  Lists deployments in the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Bundle.ListDeployments(ctx, listDeploymentsReq)
		if listDeploymentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDeploymentsLimit)
		}
		if listDeploymentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDeploymentsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDeploymentsOverrides {
		fn(cmd, &listDeploymentsReq)
	}

	return cmd
}

// start list-operations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOperationsOverrides []func(
	*cobra.Command,
	*bundle.ListOperationsRequest,
)

func newListOperations() *cobra.Command {
	cmd := &cobra.Command{}

	var listOperationsReq bundle.ListOperationsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listOperationsLimit int

	cmd.Flags().IntVar(&listOperationsReq.PageSize, "page-size", listOperationsReq.PageSize, `The maximum number of operations to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listOperationsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listOperationsReq.PageToken, "page-token", listOperationsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-operations PARENT"
	cmd.Short = `List operations.`
	cmd.Long = `List operations.
  
  Lists resource operations under a version.

  Arguments:
    PARENT: The parent version. Format:
      deployments/{deployment_id}/versions/{version_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listOperationsReq.Parent = args[0]

		response := w.Bundle.ListOperations(ctx, listOperationsReq)
		if listOperationsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listOperationsLimit)
		}
		if listOperationsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listOperationsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOperationsOverrides {
		fn(cmd, &listOperationsReq)
	}

	return cmd
}

// start list-resources command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listResourcesOverrides []func(
	*cobra.Command,
	*bundle.ListResourcesRequest,
)

func newListResources() *cobra.Command {
	cmd := &cobra.Command{}

	var listResourcesReq bundle.ListResourcesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listResourcesLimit int

	cmd.Flags().IntVar(&listResourcesReq.PageSize, "page-size", listResourcesReq.PageSize, `The maximum number of resources to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listResourcesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listResourcesReq.PageToken, "page-token", listResourcesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-resources PARENT"
	cmd.Short = `List resources.`
	cmd.Long = `List resources.
  
  Lists resources under a deployment.

  Arguments:
    PARENT: The parent deployment. Format: deployments/{deployment_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listResourcesReq.Parent = args[0]

		response := w.Bundle.ListResources(ctx, listResourcesReq)
		if listResourcesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listResourcesLimit)
		}
		if listResourcesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listResourcesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listResourcesOverrides {
		fn(cmd, &listResourcesReq)
	}

	return cmd
}

// start list-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listVersionsOverrides []func(
	*cobra.Command,
	*bundle.ListVersionsRequest,
)

func newListVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var listVersionsReq bundle.ListVersionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listVersionsLimit int

	cmd.Flags().IntVar(&listVersionsReq.PageSize, "page-size", listVersionsReq.PageSize, `The maximum number of versions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listVersionsReq.PageToken, "page-token", listVersionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-versions PARENT"
	cmd.Short = `List versions.`
	cmd.Long = `List versions.
  
  Lists versions under a deployment, ordered by version_id descending (most
  recent first).

  Arguments:
    PARENT: The parent deployment. Format: deployments/{deployment_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listVersionsReq.Parent = args[0]

		response := w.Bundle.ListVersions(ctx, listVersionsReq)
		if listVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listVersionsLimit)
		}
		if listVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listVersionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listVersionsOverrides {
		fn(cmd, &listVersionsReq)
	}

	return cmd
}

// end service Bundle
