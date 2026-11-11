// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_functions

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/aifunctions"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ai-functions",
		Short:   `Transform and enrich data with AI on Databricks.`,
		Long:    `Transform and enrich data with AI on Databricks.`,
		GroupID: "aifunctions",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newAiClassify())
	cmd.AddCommand(newAiDecide())
	cmd.AddCommand(newAiExtract())
	cmd.AddCommand(newAiParseDocument())
	cmd.AddCommand(newCancelBatch())
	cmd.AddCommand(newCreateBatch())
	cmd.AddCommand(newGetBatch())
	cmd.AddCommand(newListBatches())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start ai-classify command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var aiClassifyOverrides []func(
	*cobra.Command,
	*aifunctions.AiClassifyRequest,
)

func newAiClassify() *cobra.Command {
	cmd := &cobra.Command{}

	var aiClassifyReq aifunctions.AiClassifyRequest
	var aiClassifyJson flags.JsonFlag

	cmd.Flags().Var(&aiClassifyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: options

	cmd.Use = "ai-classify"
	cmd.Short = `Classify text into labels.`
	cmd.Long = `Classify text into labels.

  Classifies content according to a set of provided labels. For REST API
  requests, the default rate limit is 1,200 requests per minute per workspace.
  Contact your Databricks account team to request a higher limit.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := aiClassifyJson.Unmarshal(&aiClassifyReq)
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

		response, err := w.AiFunctions.AiClassify(ctx, aiClassifyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range aiClassifyOverrides {
		fn(cmd, &aiClassifyReq)
	}

	return cmd
}

// start ai-decide command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var aiDecideOverrides []func(
	*cobra.Command,
	*aifunctions.AiDecideRequest,
)

func newAiDecide() *cobra.Command {
	cmd := &cobra.Command{}

	var aiDecideReq aifunctions.AiDecideRequest
	var aiDecideJson flags.JsonFlag

	cmd.Flags().Var(&aiDecideJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: options

	cmd.Use = "ai-decide"
	cmd.Short = `*Beta* Get fast structured decisions.`
	cmd.Long = `This command is in Beta and may change without notice.

Get fast structured decisions.

  Turn text and structured data into decisions your application can use. Define
  questions and criteria to choose an option, estimate a probability, or assign
  a score given a provided state.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := aiDecideJson.Unmarshal(&aiDecideReq)
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

		response, err := w.AiFunctions.AiDecide(ctx, aiDecideReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range aiDecideOverrides {
		fn(cmd, &aiDecideReq)
	}

	return cmd
}

// start ai-extract command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var aiExtractOverrides []func(
	*cobra.Command,
	*aifunctions.AiExtractRequest,
)

func newAiExtract() *cobra.Command {
	cmd := &cobra.Command{}

	var aiExtractReq aifunctions.AiExtractRequest
	var aiExtractJson flags.JsonFlag

	cmd.Flags().Var(&aiExtractJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: options

	cmd.Use = "ai-extract"
	cmd.Short = `Extract structured data from text.`
	cmd.Long = `Extract structured data from text.

  Extracts structured data from text and documents according to a provided
  schema. For REST API requests, the default rate limit is 120 requests per
  minute per workspace. Contact your Databricks account team to request a higher
  limit.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := aiExtractJson.Unmarshal(&aiExtractReq)
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

		response, err := w.AiFunctions.AiExtract(ctx, aiExtractReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range aiExtractOverrides {
		fn(cmd, &aiExtractReq)
	}

	return cmd
}

// start ai-parse-document command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var aiParseDocumentOverrides []func(
	*cobra.Command,
	*aifunctions.AiParseDocumentRequest,
)

func newAiParseDocument() *cobra.Command {
	cmd := &cobra.Command{}

	var aiParseDocumentReq aifunctions.AiParseDocumentRequest
	var aiParseDocumentJson flags.JsonFlag

	cmd.Flags().Var(&aiParseDocumentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: options

	cmd.Use = "ai-parse-document CONTENT"
	cmd.Short = `Parse documents into structured content.`
	cmd.Long = `Parse documents into structured content.

  Parse structured content from unstructured documents.

  Arguments:
    CONTENT: The document to parse. The REST API accepts a Unity Catalog volume path to
      the source file or a base64 data URL containing the file bytes (for
      example, data:application/pdf;base64,<encoded-file-bytes> or
      data:image/jpeg;base64,<encoded-file-bytes>). Supported formats: PDF,
      DOCX, DOC, PPTX, PPT, JPG, JPEG, PNG, TIFF. Accepts up to 100 pages and
      100 MB per document.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'content' in your JSON input")
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
			diags := aiParseDocumentJson.Unmarshal(&aiParseDocumentReq)
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
			aiParseDocumentReq.Content = args[0]
		}

		response, err := w.AiFunctions.AiParseDocument(ctx, aiParseDocumentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range aiParseDocumentOverrides {
		fn(cmd, &aiParseDocumentReq)
	}

	return cmd
}

// start cancel-batch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelBatchOverrides []func(
	*cobra.Command,
	*aifunctions.CancelBatchRequest,
)

func newCancelBatch() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelBatchReq aifunctions.CancelBatchRequest

	cmd.Use = "cancel-batch NAME"
	cmd.Short = `Cancel a batch.`
	cmd.Long = `Cancel a batch.

  Cancel a batch. Cancellation is asynchronous: requests in flight drain first,
  so read the batch to observe the CANCELLING and CANCELLED statuses.

  Arguments:
    NAME: ID of the batch, in the form batch_<id>.`

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

		cancelBatchReq.Name = args[0]

		response, err := w.AiFunctions.CancelBatch(ctx, cancelBatchReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelBatchOverrides {
		fn(cmd, &cancelBatchReq)
	}

	return cmd
}

// start create-batch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createBatchOverrides []func(
	*cobra.Command,
	*aifunctions.CreateBatchRequest,
)

func newCreateBatch() *cobra.Command {
	cmd := &cobra.Command{}

	var createBatchReq aifunctions.CreateBatchRequest
	createBatchReq.Batch = aifunctions.Batch{}
	var createBatchJson flags.JsonFlag

	cmd.Flags().Var(&createBatchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createBatchReq.Batch.Name, "name", createBatchReq.Batch.Name, `ID of the batch, in the form batch_<id>.`)
	cmd.Flags().StringVar(&createBatchReq.Batch.OutputRef, "output-ref", createBatchReq.Batch.OutputRef, `Path on a Unity Catalog volume to write the JSONL output to.`)
	// TODO: map via StringToStringVar: params
	// TODO: complex arg: request_counts

	cmd.Use = "create-batch ENDPOINT INPUT_REF"
	cmd.Short = `Create a batch.`
	cmd.Long = `Create a batch.

  Run an AI function over every request in a file on a Unity Catalog volume. The
  batch runs asynchronously; read it to follow its progress.

  Arguments:
    ENDPOINT: The AI function to run for every request, for example ai_extract.
    INPUT_REF: Path of the JSONL input file on a Unity Catalog volume. Each line is one
      request.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'endpoint', 'input_ref' in your JSON input")
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
			diags := createBatchJson.Unmarshal(&createBatchReq.Batch)
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
			createBatchReq.Batch.Endpoint = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createBatchReq.Batch.InputRef = args[1]
		}

		response, err := w.AiFunctions.CreateBatch(ctx, createBatchReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createBatchOverrides {
		fn(cmd, &createBatchReq)
	}

	return cmd
}

// start get-batch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBatchOverrides []func(
	*cobra.Command,
	*aifunctions.GetBatchRequest,
)

func newGetBatch() *cobra.Command {
	cmd := &cobra.Command{}

	var getBatchReq aifunctions.GetBatchRequest

	cmd.Use = "get-batch NAME"
	cmd.Short = `Get a batch.`
	cmd.Long = `Get a batch.

  Read one batch, with its status and progress.

  Arguments:
    NAME: ID of the batch, in the form batch_<id>.`

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

		getBatchReq.Name = args[0]

		response, err := w.AiFunctions.GetBatch(ctx, getBatchReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBatchOverrides {
		fn(cmd, &getBatchReq)
	}

	return cmd
}

// start list-batches command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBatchesOverrides []func(
	*cobra.Command,
	*aifunctions.ListBatchesRequest,
)

func newListBatches() *cobra.Command {
	cmd := &cobra.Command{}

	var listBatchesReq aifunctions.ListBatchesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listBatchesLimit int

	cmd.Flags().IntVar(&listBatchesReq.PageSize, "page-size", listBatchesReq.PageSize, `Maximum number of batches to return in one page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listBatchesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listBatchesReq.PageToken, "page-token", listBatchesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-batches"
	cmd.Short = `List batches.`
	cmd.Long = `List batches.

  List the batches the caller created in this workspace.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.AiFunctions.ListBatches(ctx, listBatchesReq)
		if listBatchesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listBatchesLimit)
		}
		if listBatchesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listBatchesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBatchesOverrides {
		fn(cmd, &listBatchesReq)
	}

	return cmd
}

// end service AiFunctions
