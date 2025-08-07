// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package temporary_table_credentials

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "temporary-table-credentials",
		Short: `Temporary Table Credentials refer to short-lived, downscoped credentials used to access cloud storage locationswhere table data is stored in Databricks.`,
		Long: `Temporary Table Credentials refer to short-lived, downscoped credentials used
  to access cloud storage locationswhere table data is stored in Databricks.
  These credentials are employed to provide secure and time-limitedaccess to
  data in cloud environments such as AWS, Azure, and Google Cloud. Each cloud
  provider has its own typeof credentials: AWS uses temporary session tokens via
  AWS Security Token Service (STS), Azure utilizesShared Access Signatures (SAS)
  for its data storage services, and Google Cloud supports temporary
  credentialsthrough OAuth 2.0.Temporary table credentials ensure that data
  access is limited in scope and duration, reducing the risk ofunauthorized
  access or misuse. To use the temporary table credentials API, a metastore
  admin needs to enable the external_access_enabled flag (off by default) at the
  metastore level, and user needs to be granted the EXTERNAL USE SCHEMA
  permission at the schema level by catalog admin. Note that EXTERNAL USE SCHEMA
  is a schema level permission that can only be granted by catalog admin
  explicitly and is not included in schema ownership or ALL PRIVILEGES on the
  schema for security reason.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGenerateTemporaryTableCredentials())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start generate-temporary-table-credentials command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateTemporaryTableCredentialsOverrides []func(
	*cobra.Command,
	*catalog.GenerateTemporaryTableCredentialRequest,
)

func newGenerateTemporaryTableCredentials() *cobra.Command {
	cmd := &cobra.Command{}

	var generateTemporaryTableCredentialsReq catalog.GenerateTemporaryTableCredentialRequest
	var generateTemporaryTableCredentialsJson flags.JsonFlag

	cmd.Flags().Var(&generateTemporaryTableCredentialsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&generateTemporaryTableCredentialsReq.Operation, "operation", `The operation performed against the table data, either READ or READ_WRITE. Supported values: [READ, READ_WRITE]`)
	cmd.Flags().StringVar(&generateTemporaryTableCredentialsReq.TableId, "table-id", generateTemporaryTableCredentialsReq.TableId, `UUID of the table to read or write.`)

	cmd.Use = "generate-temporary-table-credentials"
	cmd.Short = `Generate a temporary table credential.`
	cmd.Long = `Generate a temporary table credential.
  
  Get a short-lived credential for directly accessing the table data on cloud
  storage. The metastore must have external_access_enabled flag set to true
  (default false). The caller must have EXTERNAL_USE_SCHEMA privilege on the
  parent schema and this privilege can only be granted by catalog owners.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := generateTemporaryTableCredentialsJson.Unmarshal(&generateTemporaryTableCredentialsReq)
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

		response, err := w.TemporaryTableCredentials.GenerateTemporaryTableCredentials(ctx, generateTemporaryTableCredentialsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateTemporaryTableCredentialsOverrides {
		fn(cmd, &generateTemporaryTableCredentialsReq)
	}

	return cmd
}

// end service TemporaryTableCredentials
