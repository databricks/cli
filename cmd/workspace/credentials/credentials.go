// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package credentials

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	uccredentials "github.com/databricks/cli/libs/sdktypes/uc/credentials/v1"
	sdkuccredentials "github.com/databricks/sdk-go/uc/credentials/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*uccredentials.CreateCredentialRequest,
)

func newCreateCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var createCredentialReq uccredentials.CreateCredentialRequest

	var createCredentialJson flags.JsonFlag

	cmd.Flags().Var(&createCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	flags.StringPtrVar(cmd.Flags(), &createCredentialReq.Comment, "comment", `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	// TODO: field not in sdk-go request: purpose
	flags.BoolPtrVar(cmd.Flags(), &createCredentialReq.ReadOnly, "read-only", `Whether the credential is usable only for read operations.`)
	flags.BoolPtrVar(cmd.Flags(), &createCredentialReq.SkipValidation, "skip-validation", `Optional.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createCredentialJson.Unmarshal(&createCredentialReq)
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
			createCredentialReq.Name = new(args[0])
		}

		sdkReq, err := uccredentials.CreateCredentialRequestToSDK(&createCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.CreateCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.StorageCredentialInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*uccredentials.DeleteCredentialRequest,
)

func newDeleteCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCredentialReq uccredentials.DeleteCredentialRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteCredentialReq.Force, "force", `Force an update even if there are dependent services (when purpose is **SERVICE**) or dependent external locations and external tables (when purpose is **STORAGE**).`)

	cmd.Use = "delete-credential NAME_ARG"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.

  Deletes a service or storage credential from the metastore. The caller must be
  an owner of the credential.

  Arguments:
    NAME_ARG: Name of the credential.`

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

		deleteCredentialReq.NameArg = new(args[0])

		sdkReq, err := uccredentials.DeleteCredentialRequestToSDK(&deleteCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.DeleteCredential(ctx, *sdkReq)
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
	*uccredentials.GenerateTemporaryServiceCredentialRequest,
)

func newGenerateTemporaryServiceCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var generateTemporaryServiceCredentialReq uccredentials.GenerateTemporaryServiceCredentialRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'credential_name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := generateTemporaryServiceCredentialJson.Unmarshal(&generateTemporaryServiceCredentialReq)
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
			generateTemporaryServiceCredentialReq.CredentialName = new(args[0])
		}

		sdkReq, err := uccredentials.GenerateTemporaryServiceCredentialRequestToSDK(&generateTemporaryServiceCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.GenerateTemporaryServiceCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.TemporaryCredentialsFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkuccredentials.GetCredentialRequest,
)

func newGetCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var getCredentialReq sdkuccredentials.GetCredentialRequest

	cmd.Use = "get-credential NAME_ARG"
	cmd.Short = `Get a credential.`
	cmd.Long = `Get a credential.

  Gets a service or storage credential from the metastore. The caller must be a
  metastore admin, the owner of the credential, or have any permission on the
  credential.

  Arguments:
    NAME_ARG: Name of the credential.`

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

		getCredentialReq.NameArg = new(args[0])

		sdkReq := &getCredentialReq
		response, err := w.Credentials.GetCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.StorageCredentialInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*uccredentials.ListCredentialsRequest,
)

func newListCredentials() *cobra.Command {
	cmd := &cobra.Command{}
	var listCredentialsReq uccredentials.ListCredentialsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listCredentialsLimit int

	flags.BoolPtrVar(cmd.Flags(), &listCredentialsReq.IncludeUnbound, "include-unbound", `Whether to include credentials not bound to the workspace.`)
	flags.IntPtrVar(cmd.Flags(), &listCredentialsReq.MaxResults, "max-results", `Maximum number of credentials to return.`)
	// TODO: field not in sdk-go request: purpose

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listCredentialsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listCredentialsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-credentials"
	cmd.Short = `List credentials.`
	cmd.Long = `List credentials.

  Gets an array of credentials (as __CredentialInfo__ objects).

  The array is limited to only the credentials that the caller has permission to
  access. If the caller is a metastore admin, retrieval of credentials is
  unrestricted. There is no guarantee of a specific ordering of the elements in
  the array.

  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.`

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

		sdkReq, err := uccredentials.ListCredentialsRequestToSDK(&listCredentialsReq)
		if err != nil {
			return err
		}
		if listCredentialsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listCredentialsLimit)
		}
		if listCredentialsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listCredentialsLimit)
		}
		response := w.Credentials.ListCredentialsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, uccredentials.CredentialInfoFromSDK))
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
	*uccredentials.UpdateCredentialRequest,
)

func newUpdateCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var updateCredentialReq uccredentials.UpdateCredentialRequest

	var updateCredentialJson flags.JsonFlag

	cmd.Flags().Var(&updateCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	flags.StringPtrVar(cmd.Flags(), &updateCredentialReq.Comment, "comment", `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	flags.BoolPtrVar(cmd.Flags(), &updateCredentialReq.Force, "force", `Force an update even if there are dependent services (when purpose is **SERVICE**) or dependent external locations and external tables (when purpose is **STORAGE**).`)
	flags.EnumVar(cmd.Flags(), &updateCredentialReq.IsolationMode, "isolation-mode", `Whether the current securable is accessible from all workspaces or a specific set of workspaces. Supported values: [ISOLATION_MODE_ISOLATED, ISOLATION_MODE_OPEN]`, "IsolationMode", "ISOLATION_MODE_ISOLATED", "ISOLATION_MODE_OPEN")
	flags.StringPtrVar(cmd.Flags(), &updateCredentialReq.NewName, "new-name", `New name of credential.`)
	flags.StringPtrVar(cmd.Flags(), &updateCredentialReq.Owner, "owner", `Username of current owner of credential.`)
	flags.BoolPtrVar(cmd.Flags(), &updateCredentialReq.ReadOnly, "read-only", `Whether the credential is usable only for read operations.`)
	flags.BoolPtrVar(cmd.Flags(), &updateCredentialReq.SkipValidation, "skip-validation", `Supply true to this argument to skip validation of the updated credential.`)

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
			diags := updateCredentialJson.Unmarshal(&updateCredentialReq)
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
		updateCredentialReq.NameArg = new(args[0])

		sdkReq, err := uccredentials.UpdateCredentialRequestToSDK(&updateCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.UpdateCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.StorageCredentialInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*uccredentials.ValidateCredentialRequest,
)

func newValidateCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var validateCredentialReq uccredentials.ValidateCredentialRequest

	var validateCredentialJson flags.JsonFlag

	cmd.Flags().Var(&validateCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	flags.StringPtrVar(cmd.Flags(), &validateCredentialReq.CredentialName, "credential-name", `Required.`)
	// TODO: complex arg: databricks_gcp_service_account
	flags.StringPtrVar(cmd.Flags(), &validateCredentialReq.ExternalLocationName, "external-location-name", `The name of an existing external location to validate.`)
	// TODO: field not in sdk-go request: purpose
	flags.BoolPtrVar(cmd.Flags(), &validateCredentialReq.ReadOnly, "read-only", `Whether the credential is only usable for read operations.`)
	flags.StringPtrVar(cmd.Flags(), &validateCredentialReq.Url, "url", `The external location url to validate.`)

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

		if cmd.Flags().Changed("json") {
			diags := validateCredentialJson.Unmarshal(&validateCredentialReq)
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

		sdkReq, err := uccredentials.ValidateCredentialRequestToSDK(&validateCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.ValidateCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.ValidateCredentialResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
