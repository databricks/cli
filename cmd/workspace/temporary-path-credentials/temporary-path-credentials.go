// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package temporary_path_credentials

import (
	"fmt"

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
		Use:   "temporary-path-credentials",
		Short: `Temporary Path Credentials refer to short-lived, downscoped credentials used to access external cloud storage locations registered in Databricks.`,
		Long: `Temporary Path Credentials refer to short-lived, downscoped credentials used
  to access external cloud storage locations registered in Databricks. These
  credentials are employed to provide secure and time-limited access to data in
  cloud environments such as AWS, Azure, and Google Cloud. Each cloud provider
  has its own type of credentials: AWS uses temporary session tokens via AWS
  Security Token Service (STS), Azure utilizes Shared Access Signatures (SAS)
  for its data storage services, and Google Cloud supports temporary credentials
  through OAuth 2.0.

  Temporary path credentials ensure that data access is limited in scope and
  duration, reducing the risk of unauthorized access or misuse. To use the
  temporary path credentials API, a metastore admin needs to enable the
  external_access_enabled flag (off by default) at the metastore level. A user
  needs to be granted the EXTERNAL USE LOCATION permission by external location
  owner. For requests on existing external tables, user also needs to be granted
  the EXTERNAL USE SCHEMA permission at the schema level by catalog admin.

  Note that EXTERNAL USE SCHEMA is a schema level permission that can only be
  granted by catalog admin explicitly and is not included in schema ownership or
  ALL PRIVILEGES on the schema for security reasons. Similarly, EXTERNAL USE
  LOCATION is an external location level permission that can only be granted by
  external location owner explicitly and is not included in external location
  ownership or ALL PRIVILEGES on the external location for security reasons.

  This API only supports temporary path credentials for external locations and
  external tables, and volumes will be supported in the future.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGenerateTemporaryPathCredentials())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start generate-temporary-path-credentials command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateTemporaryPathCredentialsOverrides []func(
	*cobra.Command,
	*catalog.GenerateTemporaryPathCredentialRequest,
)

func newGenerateTemporaryPathCredentials() *cobra.Command {
	cmd := &cobra.Command{}

	var generateTemporaryPathCredentialsReq catalog.GenerateTemporaryPathCredentialRequest
	var generateTemporaryPathCredentialsJson flags.JsonFlag

	cmd.Flags().Var(&generateTemporaryPathCredentialsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&generateTemporaryPathCredentialsReq.DryRun, "dry-run", generateTemporaryPathCredentialsReq.DryRun, `Optional.`)

	cmd.Use = "generate-temporary-path-credentials URL OPERATION"
	cmd.Short = `Generate a temporary path credential.`
	cmd.Long = `Generate a temporary path credential.

  Get a short-lived credential for directly accessing cloud storage locations
  registered in Databricks. The Generate Temporary Path Credentials API is only
  supported for external storage paths, specifically external locations and
  external tables. Managed tables are not supported by this API. The metastore
  must have **external_access_enabled** flag set to true (default false). The
  caller must have the **EXTERNAL_USE_LOCATION** privilege on the external
  location; this privilege can only be granted by external location owners. For
  requests on existing external tables, the caller must also have the
  **EXTERNAL_USE_SCHEMA** privilege on the parent schema; this privilege can
  only be granted by catalog owners.

  Arguments:
    URL: URL for path-based access.
    OPERATION: The operation being performed on the path.
      Supported values: [PATH_CREATE_TABLE, PATH_READ, PATH_READ_WRITE]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'url', 'operation' in your JSON input")
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
			diags := generateTemporaryPathCredentialsJson.Unmarshal(&generateTemporaryPathCredentialsReq)
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
			generateTemporaryPathCredentialsReq.Url = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &generateTemporaryPathCredentialsReq.Operation)
			if err != nil {
				return fmt.Errorf("invalid OPERATION: %s", args[1])
			}

		}

		response, err := w.TemporaryPathCredentials.GenerateTemporaryPathCredentials(ctx, generateTemporaryPathCredentialsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateTemporaryPathCredentialsOverrides {
		fn(cmd, &generateTemporaryPathCredentialsReq)
	}

	return cmd
}

// end service TemporaryPathCredentials
