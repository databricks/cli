// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package temporary_volume_credentials

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
		Use:   "temporary-volume-credentials",
		Short: `Temporary Volume Credentials refer to short-lived, downscoped credentials used to access cloud storage locations where volume data is stored in Databricks.`,
		Long: `Temporary Volume Credentials refer to short-lived, downscoped credentials used
  to access cloud storage locations where volume data is stored in Databricks.
  These credentials are employed to provide secure and time-limited access to
  data in cloud environments such as AWS, Azure, and Google Cloud. Each cloud
  provider has its own type of credentials: AWS uses temporary session tokens
  via AWS Security Token Service (STS), Azure utilizes Shared Access Signatures
  (SAS) for its data storage services, and Google Cloud supports temporary
  credentials through OAuth 2.0.
  
  Temporary volume credentials ensure that data access is limited in scope and
  duration, reducing the risk of unauthorized access or misuse. To use the
  temporary volume credentials API, a metastore admin needs to enable the
  external_access_enabled flag (off by default) at the metastore level, and user
  needs to be granted the EXTERNAL USE SCHEMA permission at the schema level by
  catalog owner. Note that EXTERNAL USE SCHEMA is a schema level permission that
  can only be granted by catalog owner explicitly and is not included in schema
  ownership or ALL PRIVILEGES on the schema for security reasons.`,
		GroupID: "catalog",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGenerateTemporaryVolumeCredentials())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start generate-temporary-volume-credentials command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateTemporaryVolumeCredentialsOverrides []func(
	*cobra.Command,
	*catalog.GenerateTemporaryVolumeCredentialRequest,
)

func newGenerateTemporaryVolumeCredentials() *cobra.Command {
	cmd := &cobra.Command{}

	var generateTemporaryVolumeCredentialsReq catalog.GenerateTemporaryVolumeCredentialRequest
	var generateTemporaryVolumeCredentialsJson flags.JsonFlag

	cmd.Flags().Var(&generateTemporaryVolumeCredentialsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&generateTemporaryVolumeCredentialsReq.Operation, "operation", `The operation performed against the volume data, either READ_VOLUME or WRITE_VOLUME. Supported values: [READ_VOLUME, WRITE_VOLUME]`)
	cmd.Flags().StringVar(&generateTemporaryVolumeCredentialsReq.VolumeId, "volume-id", generateTemporaryVolumeCredentialsReq.VolumeId, `Id of the volume to read or write.`)

	cmd.Use = "generate-temporary-volume-credentials"
	cmd.Short = `Generate a temporary volume credential.`
	cmd.Long = `Generate a temporary volume credential.
  
  Get a short-lived credential for directly accessing the volume data on cloud
  storage. The metastore must have **external_access_enabled** flag set to true
  (default false). The caller must have the **EXTERNAL_USE_SCHEMA** privilege on
  the parent schema and this privilege can only be granted by catalog owners.`

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
			diags := generateTemporaryVolumeCredentialsJson.Unmarshal(&generateTemporaryVolumeCredentialsReq)
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

		response, err := w.TemporaryVolumeCredentials.GenerateTemporaryVolumeCredentials(ctx, generateTemporaryVolumeCredentialsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateTemporaryVolumeCredentialsOverrides {
		fn(cmd, &generateTemporaryVolumeCredentialsReq)
	}

	return cmd
}

// end service TemporaryVolumeCredentials
