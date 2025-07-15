// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package aibi_dashboard_embedding_approved_domains

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aibi-dashboard-embedding-approved-domains",
		Short: `Controls the list of domains approved to host the embedded AI/BI dashboards.`,
		Long: `Controls the list of domains approved to host the embedded AI/BI dashboards.
  The approved domains list can't be mutated when the current access policy is
  not set to ALLOW_APPROVED_DOMAINS.`,
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*settings.DeleteAibiDashboardEmbeddingApprovedDomainsSettingRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq settings.DeleteAibiDashboardEmbeddingApprovedDomainsSettingRequest

	cmd.Flags().StringVar(&deleteReq.Etag, "etag", deleteReq.Etag, `etag used for versioning.`)

	cmd.Use = "delete"
	cmd.Short = `Delete AI/BI dashboard embedding approved domains.`
	cmd.Long = `Delete AI/BI dashboard embedding approved domains.
  
  Delete the list of domains approved to host embedded AI/BI dashboards,
  reverting back to the default empty list.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Settings.AibiDashboardEmbeddingApprovedDomains().Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*settings.GetAibiDashboardEmbeddingApprovedDomainsSettingRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetAibiDashboardEmbeddingApprovedDomainsSettingRequest

	cmd.Flags().StringVar(&getReq.Etag, "etag", getReq.Etag, `etag used for versioning.`)

	cmd.Use = "get"
	cmd.Short = `Retrieve the list of domains approved to host embedded AI/BI dashboards.`
	cmd.Long = `Retrieve the list of domains approved to host embedded AI/BI dashboards.
  
  Retrieves the list of domains approved to host embedded AI/BI dashboards.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Settings.AibiDashboardEmbeddingApprovedDomains().Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*settings.UpdateAibiDashboardEmbeddingApprovedDomainsSettingRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq settings.UpdateAibiDashboardEmbeddingApprovedDomainsSettingRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update"
	cmd.Short = `Update the list of domains approved to host embedded AI/BI dashboards.`
	cmd.Long = `Update the list of domains approved to host embedded AI/BI dashboards.
  
  Updates the list of domains approved to host embedded AI/BI dashboards. This
  update will fail if the current workspace access policy is not
  ALLOW_APPROVED_DOMAINS.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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

		response, err := w.Settings.AibiDashboardEmbeddingApprovedDomains().Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// end service AibiDashboardEmbeddingApprovedDomains
