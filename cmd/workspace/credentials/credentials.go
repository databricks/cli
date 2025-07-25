// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package credentials

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
		Use:   "credentials",
		Short: `A credential represents an authentication and authorization mechanism for accessing services on your cloud tenant.`,
		Long: `A credential represents an authentication and authorization mechanism for
  accessing services on your cloud tenant. Each credential is subject to Unity
  Catalog access-control policies that control which users and groups can access
  the credential.
  
  To create credentials, you must be a Databricks account admin or have the
  CREATE SERVICE CREDENTIAL privilege. The user who creates the credential can
  delegate ownership to another user or group to manage permissions on it.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateCredential())
	cmd.AddCommand(newDeleteCredential())
	cmd.AddCommand(newGenerateTemporaryServiceCredential())
	cmd.AddCommand(newGetCredential())
	cmd.AddCommand(newListCredentials())
	cmd.AddCommand(newUpdateCredential())
	cmd.AddCommand(newValidateCredential())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCredentialOverrides []func(
	*cobra.Command,
	*catalog.CreateCredentialRequest,
)

func newCreateCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var createCredentialReq catalog.CreateCredentialRequest
	var createCredentialJson flags.JsonFlag

	cmd.Flags().Var(&createCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	cmd.Flags().StringVar(&createCredentialReq.Comment, "comment", createCredentialReq.Comment, `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	cmd.Flags().Var(&createCredentialReq.Purpose, "purpose", `Indicates the purpose of the credential. Supported values: [SERVICE, STORAGE]`)
	cmd.Flags().BoolVar(&createCredentialReq.ReadOnly, "read-only", createCredentialReq.ReadOnly, `Whether the credential is usable only for read operations.`)
	cmd.Flags().BoolVar(&createCredentialReq.SkipValidation, "skip-validation", createCredentialReq.SkipValidation, `Optional.`)

	cmd.Use = "create-credential NAME"
	cmd.Short = `Create a credential.`
	cmd.Long = `Create a credential.
  
  Creates a new credential. The type of credential to be created is determined
  by the **purpose** field, which should be either **SERVICE** or **STORAGE**.
  
  The caller must be a metastore admin or have the metastore privilege
  **CREATE_STORAGE_CREDENTIAL** for storage credentials, or
  **CREATE_SERVICE_CREDENTIAL** for service credentials.

  Arguments:
    NAME: The credential name. The name must be unique among storage and service
      credentials within the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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
			diags := createCredentialJson.Unmarshal(&createCredentialReq)
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
			createCredentialReq.Name = args[0]
		}

		response, err := w.Credentials.CreateCredential(ctx, createCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCredentialOverrides {
		fn(cmd, &createCredentialReq)
	}

	return cmd
}

// start delete-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCredentialOverrides []func(
	*cobra.Command,
	*catalog.DeleteCredentialRequest,
)

func newDeleteCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteCredentialReq catalog.DeleteCredentialRequest

	cmd.Flags().BoolVar(&deleteCredentialReq.Force, "force", deleteCredentialReq.Force, `Force an update even if there are dependent services (when purpose is **SERVICE**) or dependent external locations and external tables (when purpose is **STORAGE**).`)

	cmd.Use = "delete-credential NAME_ARG"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.
  
  Deletes a service or storage credential from the metastore. The caller must be
  an owner of the credential.

  Arguments:
    NAME_ARG: Name of the credential.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteCredentialReq.NameArg = args[0]

		err = w.Credentials.DeleteCredential(ctx, deleteCredentialReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCredentialOverrides {
		fn(cmd, &deleteCredentialReq)
	}

	return cmd
}

// start generate-temporary-service-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateTemporaryServiceCredentialOverrides []func(
	*cobra.Command,
	*catalog.GenerateTemporaryServiceCredentialRequest,
)

func newGenerateTemporaryServiceCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var generateTemporaryServiceCredentialReq catalog.GenerateTemporaryServiceCredentialRequest
	var generateTemporaryServiceCredentialJson flags.JsonFlag

	cmd.Flags().Var(&generateTemporaryServiceCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: azure_options
	// TODO: complex arg: gcp_options

	cmd.Use = "generate-temporary-service-credential CREDENTIAL_NAME"
	cmd.Short = `Generate a temporary service credential.`
	cmd.Long = `Generate a temporary service credential.
  
  Returns a set of temporary credentials generated using the specified service
  credential. The caller must be a metastore admin or have the metastore
  privilege **ACCESS** on the service credential.

  Arguments:
    CREDENTIAL_NAME: The name of the service credential used to generate a temporary credential`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'credential_name' in your JSON input")
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
			diags := generateTemporaryServiceCredentialJson.Unmarshal(&generateTemporaryServiceCredentialReq)
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
			generateTemporaryServiceCredentialReq.CredentialName = args[0]
		}

		response, err := w.Credentials.GenerateTemporaryServiceCredential(ctx, generateTemporaryServiceCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateTemporaryServiceCredentialOverrides {
		fn(cmd, &generateTemporaryServiceCredentialReq)
	}

	return cmd
}

// start get-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCredentialOverrides []func(
	*cobra.Command,
	*catalog.GetCredentialRequest,
)

func newGetCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var getCredentialReq catalog.GetCredentialRequest

	cmd.Use = "get-credential NAME_ARG"
	cmd.Short = `Get a credential.`
	cmd.Long = `Get a credential.
  
  Gets a service or storage credential from the metastore. The caller must be a
  metastore admin, the owner of the credential, or have any permission on the
  credential.

  Arguments:
    NAME_ARG: Name of the credential.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getCredentialReq.NameArg = args[0]

		response, err := w.Credentials.GetCredential(ctx, getCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCredentialOverrides {
		fn(cmd, &getCredentialReq)
	}

	return cmd
}

// start list-credentials command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCredentialsOverrides []func(
	*cobra.Command,
	*catalog.ListCredentialsRequest,
)

func newListCredentials() *cobra.Command {
	cmd := &cobra.Command{}

	var listCredentialsReq catalog.ListCredentialsRequest

	cmd.Flags().IntVar(&listCredentialsReq.MaxResults, "max-results", listCredentialsReq.MaxResults, `Maximum number of credentials to return.`)
	cmd.Flags().StringVar(&listCredentialsReq.PageToken, "page-token", listCredentialsReq.PageToken, `Opaque token to retrieve the next page of results.`)
	cmd.Flags().Var(&listCredentialsReq.Purpose, "purpose", `Return only credentials for the specified purpose. Supported values: [SERVICE, STORAGE]`)

	cmd.Use = "list-credentials"
	cmd.Short = `List credentials.`
	cmd.Long = `List credentials.
  
  Gets an array of credentials (as __CredentialInfo__ objects).
  
  The array is limited to only the credentials that the caller has permission to
  access. If the caller is a metastore admin, retrieval of credentials is
  unrestricted. There is no guarantee of a specific ordering of the elements in
  the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Credentials.ListCredentials(ctx, listCredentialsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCredentialsOverrides {
		fn(cmd, &listCredentialsReq)
	}

	return cmd
}

// start update-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCredentialOverrides []func(
	*cobra.Command,
	*catalog.UpdateCredentialRequest,
)

func newUpdateCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var updateCredentialReq catalog.UpdateCredentialRequest
	var updateCredentialJson flags.JsonFlag

	cmd.Flags().Var(&updateCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	cmd.Flags().StringVar(&updateCredentialReq.Comment, "comment", updateCredentialReq.Comment, `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	cmd.Flags().BoolVar(&updateCredentialReq.Force, "force", updateCredentialReq.Force, `Force an update even if there are dependent services (when purpose is **SERVICE**) or dependent external locations and external tables (when purpose is **STORAGE**).`)
	cmd.Flags().Var(&updateCredentialReq.IsolationMode, "isolation-mode", `Whether the current securable is accessible from all workspaces or a specific set of workspaces. Supported values: [ISOLATION_MODE_ISOLATED, ISOLATION_MODE_OPEN]`)
	cmd.Flags().StringVar(&updateCredentialReq.NewName, "new-name", updateCredentialReq.NewName, `New name of credential.`)
	cmd.Flags().StringVar(&updateCredentialReq.Owner, "owner", updateCredentialReq.Owner, `Username of current owner of credential.`)
	cmd.Flags().BoolVar(&updateCredentialReq.ReadOnly, "read-only", updateCredentialReq.ReadOnly, `Whether the credential is usable only for read operations.`)
	cmd.Flags().BoolVar(&updateCredentialReq.SkipValidation, "skip-validation", updateCredentialReq.SkipValidation, `Supply true to this argument to skip validation of the updated credential.`)

	cmd.Use = "update-credential NAME_ARG"
	cmd.Short = `Update a credential.`
	cmd.Long = `Update a credential.
  
  Updates a service or storage credential on the metastore.
  
  The caller must be the owner of the credential or a metastore admin or have
  the MANAGE permission. If the caller is a metastore admin, only the
  __owner__ field can be changed.

  Arguments:
    NAME_ARG: Name of the credential.`

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
			diags := updateCredentialJson.Unmarshal(&updateCredentialReq)
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
		updateCredentialReq.NameArg = args[0]

		response, err := w.Credentials.UpdateCredential(ctx, updateCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCredentialOverrides {
		fn(cmd, &updateCredentialReq)
	}

	return cmd
}

// start validate-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var validateCredentialOverrides []func(
	*cobra.Command,
	*catalog.ValidateCredentialRequest,
)

func newValidateCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var validateCredentialReq catalog.ValidateCredentialRequest
	var validateCredentialJson flags.JsonFlag

	cmd.Flags().Var(&validateCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	cmd.Flags().StringVar(&validateCredentialReq.CredentialName, "credential-name", validateCredentialReq.CredentialName, `Required.`)
	// TODO: complex arg: databricks_gcp_service_account
	cmd.Flags().StringVar(&validateCredentialReq.ExternalLocationName, "external-location-name", validateCredentialReq.ExternalLocationName, `The name of an existing external location to validate.`)
	cmd.Flags().Var(&validateCredentialReq.Purpose, "purpose", `The purpose of the credential. Supported values: [SERVICE, STORAGE]`)
	cmd.Flags().BoolVar(&validateCredentialReq.ReadOnly, "read-only", validateCredentialReq.ReadOnly, `Whether the credential is only usable for read operations.`)
	cmd.Flags().StringVar(&validateCredentialReq.Url, "url", validateCredentialReq.Url, `The external location url to validate.`)

	cmd.Use = "validate-credential"
	cmd.Short = `Validate a credential.`
	cmd.Long = `Validate a credential.
  
  Validates a credential.
  
  For service credentials (purpose is **SERVICE**), either the
  __credential_name__ or the cloud-specific credential must be provided.
  
  For storage credentials (purpose is **STORAGE**), at least one of
  __external_location_name__ and __url__ need to be provided. If only one of
  them is provided, it will be used for validation. And if both are provided,
  the __url__ will be used for validation, and __external_location_name__ will
  be ignored when checking overlapping urls. Either the __credential_name__ or
  the cloud-specific credential must be provided.
  
  The caller must be a metastore admin or the credential owner or have the
  required permission on the metastore and the credential (e.g.,
  **CREATE_EXTERNAL_LOCATION** when purpose is **STORAGE**).`

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
			diags := validateCredentialJson.Unmarshal(&validateCredentialReq)
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

		response, err := w.Credentials.ValidateCredential(ctx, validateCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range validateCredentialOverrides {
		fn(cmd, &validateCredentialReq)
	}

	return cmd
}

// end service Credentials
