// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vector_search_endpoints

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vector-search-endpoints",
		Short:   `**Endpoint**: Represents the compute resources to host vector search indexes.`,
		Long:    `**Endpoint**: Represents the compute resources to host vector search indexes.`,
		GroupID: "vectorsearch",
		Annotations: map[string]string{
			"package": "vectorsearch",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newListEndpoints())
	cmd.AddCommand(newUpdateEndpointBudgetPolicy())
	cmd.AddCommand(newUpdateEndpointCustomTags())

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
	*vectorsearch.CreateEndpoint,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var createEndpointReq vectorsearch.CreateEndpoint
	var createEndpointJson flags.JsonFlag

	var createEndpointSkipWait bool
	var createEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&createEndpointSkipWait, "no-wait", createEndpointSkipWait, `do not wait to reach ONLINE state`)
	cmd.Flags().DurationVar(&createEndpointTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ONLINE state`)

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createEndpointReq.BudgetPolicyId, "budget-policy-id", createEndpointReq.BudgetPolicyId, `The budget policy id to be applied.`)

	cmd.Use = "create-endpoint NAME ENDPOINT_TYPE"
	cmd.Short = `Create an endpoint.`
	cmd.Long = `Create an endpoint.
  
  Create a new endpoint.

  Arguments:
    NAME: Name of the vector search endpoint
    ENDPOINT_TYPE: Type of endpoint 
      Supported values: [STANDARD]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'endpoint_type' in your JSON input")
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
			diags := createEndpointJson.Unmarshal(&createEndpointReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createEndpointReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createEndpointReq.EndpointType)
			if err != nil {
				return fmt.Errorf("invalid ENDPOINT_TYPE: %s", args[1])
			}
		}

		wait, err := w.VectorSearchEndpoints.CreateEndpoint(ctx, createEndpointReq)
		if err != nil {
			return err
		}
		if createEndpointSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *vectorsearch.EndpointInfo) {
			if i.EndpointStatus == nil {
				return
			}
			status := i.EndpointStatus.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.EndpointStatus != nil {
				statusMessage = i.EndpointStatus.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(createEndpointTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
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
	*vectorsearch.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteEndpointReq vectorsearch.DeleteEndpointRequest

	cmd.Use = "delete-endpoint ENDPOINT_NAME"
	cmd.Short = `Delete an endpoint.`
	cmd.Long = `Delete an endpoint.
  
  Delete a vector search endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the vector search endpoint`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteEndpointReq.EndpointName = args[0]

		err = w.VectorSearchEndpoints.DeleteEndpoint(ctx, deleteEndpointReq)
		if err != nil {
			return err
		}
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
	*vectorsearch.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var getEndpointReq vectorsearch.GetEndpointRequest

	cmd.Use = "get-endpoint ENDPOINT_NAME"
	cmd.Short = `Get an endpoint.`
	cmd.Long = `Get an endpoint.
  
  Get details for a single vector search endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the endpoint`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getEndpointReq.EndpointName = args[0]

		response, err := w.VectorSearchEndpoints.GetEndpoint(ctx, getEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, `Token for pagination.`)

	cmd.Use = "list-endpoints"
	cmd.Short = `List all endpoints.`
	cmd.Long = `List all endpoints.
  
  List all vector search endpoints in the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.VectorSearchEndpoints.ListEndpoints(ctx, listEndpointsReq)
		return cmdio.RenderIterator(ctx, response)
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
	cmd.Short = `Update the budget policy of an endpoint.`
	cmd.Long = `Update the budget policy of an endpoint.
  
  Update the budget policy of an endpoint

  Arguments:
    ENDPOINT_NAME: Name of the vector search endpoint
    BUDGET_POLICY_ID: The budget policy id to be applied`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only ENDPOINT_NAME as positional arguments. Provide 'budget_policy_id' in your JSON input")
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
			diags := updateEndpointBudgetPolicyJson.Unmarshal(&updateEndpointBudgetPolicyReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateEndpointBudgetPolicyReq.EndpointName = args[0]
		if !cmd.Flags().Changed("json") {
			updateEndpointBudgetPolicyReq.BudgetPolicyId = args[1]
		}

		response, err := w.VectorSearchEndpoints.UpdateEndpointBudgetPolicy(ctx, updateEndpointBudgetPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
    ENDPOINT_NAME: Name of the vector search endpoint`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := updateEndpointCustomTagsJson.Unmarshal(&updateEndpointCustomTagsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateEndpointCustomTagsReq.EndpointName = args[0]

		response, err := w.VectorSearchEndpoints.UpdateEndpointCustomTags(ctx, updateEndpointCustomTagsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// end service VectorSearchEndpoints
