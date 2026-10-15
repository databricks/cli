// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vector_search_endpoints

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	vectorsearch "github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vector-search-endpoints",
		Short:   `**Endpoint**: Represents the compute resources to host AI Search indexes.`,
		Long:    `**Endpoint**: Represents the compute resources to host AI Search indexes.`,
		GroupID: "vectorsearch",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newListEndpoints())
	cmd.AddCommand(newPatchEndpoint())
	cmd.AddCommand(newRetrieveUserVisibleMetrics())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdateEndpointBudgetPolicy())
	cmd.AddCommand(newUpdateEndpointCustomTags())
	cmd.AddCommand(newUpdatePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createEndpointOverrides []func(
	*cobra.Command,
	*vectorsearch.CreateEndpointRequest,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var createEndpointReq vectorsearch.CreateEndpointRequest

	var createEndpointJson flags.JsonFlag

	var createEndpointSkipWait bool
	var createEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&createEndpointSkipWait, "no-wait", createEndpointSkipWait, `do not wait to reach ONLINE state`)
	cmd.Flags().DurationVar(&createEndpointTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ONLINE state`)

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createEndpointReq.BudgetPolicyId, "budget-policy-id", `The budget policy id to be applied.`)
	flags.Int64PtrVar(cmd.Flags(), &createEndpointReq.TargetQps, "target-qps", `Target QPS for the endpoint.`)
	flags.StringPtrVar(cmd.Flags(), &createEndpointReq.UsagePolicyId, "usage-policy-id", `The usage policy id to be applied once we've migrated to usage policies.`)

	cmd.Use = "create-endpoint NAME ENDPOINT_TYPE"
	cmd.Short = `Create an endpoint.`
	cmd.Long = `Create an endpoint.

  Create a new endpoint.

  Arguments:
    NAME: Name of the AI Search endpoint
    ENDPOINT_TYPE: Type of endpoint
      Supported values: [STANDARD, STORAGE_OPTIMIZED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'endpoint_type' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createEndpointJson.Unmarshal(&createEndpointReq)
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
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createEndpointReq.EndpointType = vectorsearch.EndpointType(args[1])
		}

		sdkReq, err := vectorsearch.CreateEndpointRequestToSDK(&createEndpointReq)
		if err != nil {
			return err
		}
		wait, err := w.VectorSearch.CreateEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createEndpointSkipWait {
			return cmdio.Render(ctx, map[string]any{"name": wait.GetName()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create-endpoint to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createEndpointTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := vectorsearch.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createEndpointOverrides {
		fn(cmd, &createEndpointReq)
	}

	return cmd
}

// start delete-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteEndpointOverrides []func(
	*cobra.Command,
	*sdkvectorsearch.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteEndpointReq sdkvectorsearch.DeleteEndpointRequest

	cmd.Use = "delete-endpoint ENDPOINT_NAME"
	cmd.Short = `Delete an endpoint.`
	cmd.Long = `Delete an endpoint.

  Delete an AI Search endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the AI Search endpoint`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteEndpointReq.Name = new(args[0])

		sdkReq := &deleteEndpointReq
		response, err := w.VectorSearch.DeleteEndpoint(ctx, *sdkReq)
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
	for _, fn := range deleteEndpointOverrides {
		fn(cmd, &deleteEndpointReq)
	}

	return cmd
}

// start get-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEndpointOverrides []func(
	*cobra.Command,
	*sdkvectorsearch.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var getEndpointReq sdkvectorsearch.GetEndpointRequest

	cmd.Use = "get-endpoint ENDPOINT_NAME"
	cmd.Short = `Get an endpoint.`
	cmd.Long = `Get an endpoint.

  Get details for a single AI Search endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the endpoint`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getEndpointReq.Name = new(args[0])

		sdkReq := &getEndpointReq
		response, err := w.VectorSearch.GetEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEndpointOverrides {
		fn(cmd, &getEndpointReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels ENDPOINT_ID"
	cmd.Short = `Get vector search endpoint permission levels.`
	cmd.Long = `Get vector search endpoint permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    ENDPOINT_ID: The vector search endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		_ = args[0] // TODO: field not in sdk-go request: endpoint_id

		getPermissionLevelsReq.RequestObjectType = new("vector-search-endpoints")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions ENDPOINT_ID"
	cmd.Short = `Get vector search endpoint permissions.`
	cmd.Long = `Get vector search endpoint permissions.

  Gets the permissions of a vector search endpoint. Vector search endpoints can
  inherit permissions from their root object.

  Arguments:
    ENDPOINT_ID: The vector search endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		_ = args[0] // TODO: field not in sdk-go request: endpoint_id

		getPermissionsReq.RequestObjectType = new("vector-search-endpoints")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
	}

	return cmd
}

// start list-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEndpointsOverrides []func(
	*cobra.Command,
	*vectorsearch.ListEndpointsRequest,
)

func newListEndpoints() *cobra.Command {
	cmd := &cobra.Command{}
	var listEndpointsReq vectorsearch.ListEndpointsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEndpointsLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEndpointsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listEndpointsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-endpoints"
	cmd.Short = `List all endpoints.`
	cmd.Long = `List all endpoints.

  List all AI Search endpoints in the workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := vectorsearch.ListEndpointsRequestToSDK(&listEndpointsReq)
		if err != nil {
			return err
		}
		if listEndpointsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEndpointsLimit)
		}
		if listEndpointsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEndpointsLimit)
		}
		response := w.VectorSearch.ListEndpointsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, vectorsearch.EndpointFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEndpointsOverrides {
		fn(cmd, &listEndpointsReq)
	}

	return cmd
}

// start patch-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchEndpointOverrides []func(
	*cobra.Command,
	*vectorsearch.PatchEndpointRequest,
)

func newPatchEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var patchEndpointReq vectorsearch.PatchEndpointRequest

	var patchEndpointJson flags.JsonFlag

	cmd.Flags().Var(&patchEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Int64PtrVar(cmd.Flags(), &patchEndpointReq.TargetQps, "target-qps", `Target QPS for the endpoint.`)

	cmd.Use = "patch-endpoint ENDPOINT_NAME"
	cmd.Short = `*Public Preview* Update an endpoint.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update an endpoint.

  Update an endpoint

  Arguments:
    ENDPOINT_NAME: Name of the AI Search endpoint`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := patchEndpointJson.Unmarshal(&patchEndpointReq)
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
		patchEndpointReq.Name = new(args[0])

		sdkReq, err := vectorsearch.PatchEndpointRequestToSDK(&patchEndpointReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.PatchEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchEndpointOverrides {
		fn(cmd, &patchEndpointReq)
	}

	return cmd
}

// start retrieve-user-visible-metrics command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var retrieveUserVisibleMetricsOverrides []func(
	*cobra.Command,
	*vectorsearch.RetrieveUserVisibleMetricsRequest,
)

func newRetrieveUserVisibleMetrics() *cobra.Command {
	cmd := &cobra.Command{}
	var retrieveUserVisibleMetricsReq vectorsearch.RetrieveUserVisibleMetricsRequest

	var retrieveUserVisibleMetricsJson flags.JsonFlag

	cmd.Flags().Var(&retrieveUserVisibleMetricsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: end_time
	flags.IntPtrVar(cmd.Flags(), &retrieveUserVisibleMetricsReq.GranularityInSeconds, "granularity-in-seconds", `Granularity in seconds.`)
	// TODO: array: metrics
	flags.StringPtrVar(cmd.Flags(), &retrieveUserVisibleMetricsReq.PageToken, "page-token", `Token for pagination.`)
	// TODO: complex arg: start_time

	cmd.Use = "retrieve-user-visible-metrics NAME"
	cmd.Short = `Retrieve user-visible metrics for an endpoint.`
	cmd.Long = `Retrieve user-visible metrics for an endpoint.

  Retrieve user-visible metrics for an endpoint

  Arguments:
    NAME: AI Search endpoint name`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := retrieveUserVisibleMetricsJson.Unmarshal(&retrieveUserVisibleMetricsReq)
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
		retrieveUserVisibleMetricsReq.Name = new(args[0])

		sdkReq, err := vectorsearch.RetrieveUserVisibleMetricsRequestToSDK(&retrieveUserVisibleMetricsReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.RetrieveUserVisibleMetrics(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.RetrieveUserVisibleMetricsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range retrieveUserVisibleMetricsOverrides {
		fn(cmd, &retrieveUserVisibleMetricsReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions ENDPOINT_ID"
	cmd.Short = `Set vector search endpoint permissions.`
	cmd.Long = `Set vector search endpoint permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    ENDPOINT_ID: The vector search endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		_ = args[0] // TODO: field not in sdk-go request: endpoint_id

		setPermissionsReq.RequestObjectType = new("vector-search-endpoints")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start update-endpoint-budget-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateEndpointBudgetPolicyOverrides []func(
	*cobra.Command,
	*vectorsearch.PatchEndpointBudgetPolicyRequest,
)

func newUpdateEndpointBudgetPolicy() *cobra.Command {
	cmd := &cobra.Command{}
	var updateEndpointBudgetPolicyReq vectorsearch.PatchEndpointBudgetPolicyRequest

	var updateEndpointBudgetPolicyJson flags.JsonFlag

	cmd.Flags().Var(&updateEndpointBudgetPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-endpoint-budget-policy ENDPOINT_NAME BUDGET_POLICY_ID"
	cmd.Short = `*Public Preview* Update the budget policy of an endpoint.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the budget policy of an endpoint.

  Update the budget policy of an endpoint

  Arguments:
    ENDPOINT_NAME: Name of the AI Search endpoint
    BUDGET_POLICY_ID: The budget policy id to be applied`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only ENDPOINT_NAME as positional arguments. Provide 'budget_policy_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateEndpointBudgetPolicyJson.Unmarshal(&updateEndpointBudgetPolicyReq)
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
		updateEndpointBudgetPolicyReq.Name = new(args[0])
		if !cmd.Flags().Changed("json") {
			updateEndpointBudgetPolicyReq.BudgetPolicyId = new(args[1])
		}

		sdkReq, err := vectorsearch.PatchEndpointBudgetPolicyRequestToSDK(&updateEndpointBudgetPolicyReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.PatchEndpointBudgetPolicy(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.PatchEndpointBudgetPolicyResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateEndpointBudgetPolicyOverrides {
		fn(cmd, &updateEndpointBudgetPolicyReq)
	}

	return cmd
}

// start update-endpoint-custom-tags command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateEndpointCustomTagsOverrides []func(
	*cobra.Command,
	*vectorsearch.UpdateEndpointCustomTagsRequest,
)

func newUpdateEndpointCustomTags() *cobra.Command {
	cmd := &cobra.Command{}
	var updateEndpointCustomTagsReq vectorsearch.UpdateEndpointCustomTagsRequest

	var updateEndpointCustomTagsJson flags.JsonFlag

	cmd.Flags().Var(&updateEndpointCustomTagsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-endpoint-custom-tags ENDPOINT_NAME"
	cmd.Short = `Update the custom tags of an endpoint.`
	cmd.Long = `Update the custom tags of an endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the AI Search endpoint`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateEndpointCustomTagsJson.Unmarshal(&updateEndpointCustomTagsReq)
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
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateEndpointCustomTagsReq.Name = new(args[0])

		sdkReq, err := vectorsearch.UpdateEndpointCustomTagsRequestToSDK(&updateEndpointCustomTagsReq)
		if err != nil {
			return err
		}
		response, err := w.VectorSearch.UpdateEndpointCustomTags(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := vectorsearch.UpdateEndpointCustomTagsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateEndpointCustomTagsOverrides {
		fn(cmd, &updateEndpointCustomTagsReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions ENDPOINT_ID"
	cmd.Short = `Update vector search endpoint permissions.`
	cmd.Long = `Update vector search endpoint permissions.

  Updates the permissions on a vector search endpoint. Vector search endpoints
  can inherit permissions from their root object.

  Arguments:
    ENDPOINT_ID: The vector search endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		_ = args[0] // TODO: field not in sdk-go request: endpoint_id

		updatePermissionsReq.RequestObjectType = new("vector-search-endpoints")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service VectorSearchEndpoints
