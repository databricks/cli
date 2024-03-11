// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package vector_search_endpoints

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
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
	}

	// Add methods
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newListEndpoints())

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
	// TODO: short flags
	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-endpoint NAME ENDPOINT_TYPE"
	cmd.Short = `Create an endpoint.`
	cmd.Long = `Create an endpoint.
  
  Create a new endpoint.

  Arguments:
    NAME: Name of endpoint
    ENDPOINT_TYPE: Type of endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'endpoint_type' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createEndpointJson.Unmarshal(&createEndpointReq)
			if err != nil {
				return err
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

	// TODO: short flags

	cmd.Use = "delete-endpoint ENDPOINT_NAME"
	cmd.Short = `Delete an endpoint.`
	cmd.Long = `Delete an endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the endpoint`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Use = "get-endpoint ENDPOINT_NAME"
	cmd.Short = `Get an endpoint.`
	cmd.Long = `Get an endpoint.

  Arguments:
    ENDPOINT_NAME: Name of the endpoint`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, `Token for pagination.`)

	cmd.Use = "list-endpoints"
	cmd.Short = `List all endpoints.`
	cmd.Long = `List all endpoints.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

// end service VectorSearchEndpoints
