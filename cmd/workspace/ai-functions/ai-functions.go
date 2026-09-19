// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_functions

import (
	"errors"

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
	cmd.AddCommand(newAiExtract())
	cmd.AddCommand(newAiParseDocument())

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

  Classifies content according to a set of provided labels.`

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
  schema.`

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
    CONTENT: The document to parse, given as a Unity Catalog volume path to the source
      file (the REST API accepts only a UC volume path, not inline binary data).
      Supported formats: PDF, DOCX, DOC, PPTX, PPT, JPG, JPEG, PNG, TIFF.
      Accepts up to 100 pages and 100 MB per document.`

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

// end service AiFunctions
