// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage_credentials

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
		Use:   "storage-credentials",
		Short: `A storage credential represents an authentication and authorization mechanism for accessing data stored on your cloud tenant.`,
		Long: `A storage credential represents an authentication and authorization mechanism
  for accessing data stored on your cloud tenant. Each storage credential is
  subject to Unity Catalog access-control policies that control which users and
  groups can access the credential. If a user does not have access to a storage
  credential in Unity Catalog, the request fails and Unity Catalog does not
  attempt to authenticate to your cloud tenant on the user’s behalf.

  Databricks recommends using external locations rather than using storage
  credentials directly.

  To create storage credentials, you must be a Databricks account admin. The
  account admin who creates the storage credential can delegate ownership to
  another user or group to manage permissions on it.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newValidate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*uccredentials.CreateStorageCredentialRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq uccredentials.CreateStorageCredentialRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	flags.BoolPtrVar(cmd.Flags(), &createReq.ReadOnly, "read-only", `Whether the credential is usable only for read operations.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.SkipValidation, "skip-validation", `Supplying true to this argument skips validation of the created credential.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a storage credential.`
	cmd.Long = `Create a storage credential.

  Creates a new storage credential.

  The caller must be a metastore admin or have the **CREATE_STORAGE_CREDENTIAL**
  privilege on the metastore.

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
			diags := createJson.Unmarshal(&createReq)
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
			createReq.Name = new(args[0])
		}

		sdkReq, err := uccredentials.CreateStorageCredentialRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.CreateStorageCredential(ctx, *sdkReq)
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
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*uccredentials.DeleteStorageCredentialRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq uccredentials.DeleteStorageCredentialRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force an update even if there are dependent external locations or external tables (when purpose is **STORAGE**) or dependent services (when purpose is **SERVICE**).`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.

  Deletes a storage credential from the metastore. The caller must be an owner
  of the storage credential.

  Arguments:
    NAME: Name of the storage credential.`

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

		deleteReq.NameArg = new(args[0])

		sdkReq, err := uccredentials.DeleteStorageCredentialRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.DeleteStorageCredential(ctx, *sdkReq)
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
	*sdkuccredentials.GetStorageCredentialRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkuccredentials.GetStorageCredentialRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a credential.`
	cmd.Long = `Get a credential.

  Gets a storage credential from the metastore. The caller must be a metastore
  admin, the owner of the storage credential, or have some permission on the
  storage credential.

  Arguments:
    NAME: Name of the storage credential.`

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

		getReq.NameArg = new(args[0])

		sdkReq := &getReq
		response, err := w.Credentials.GetStorageCredential(ctx, *sdkReq)
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
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*uccredentials.ListStorageCredentialsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq uccredentials.ListStorageCredentialsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeUnbound, "include-unbound", `Whether to include credentials not bound to the workspace.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of storage credentials to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List credentials.`
	cmd.Long = `List credentials.

  Gets an array of storage credentials (as __StorageCredentialInfo__ objects).
  The array is limited to only those storage credentials the caller has
  permission to access. If the caller is a metastore admin, retrieval of
  credentials is unrestricted. There is no guarantee of a specific ordering of
  the elements in the array.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.`

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

		sdkReq, err := uccredentials.ListStorageCredentialsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Credentials.ListStorageCredentialsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, uccredentials.StorageCredentialInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*uccredentials.UpdateStorageCredentialRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq uccredentials.UpdateStorageCredentialRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `Comment associated with the credential.`)
	// TODO: complex arg: databricks_gcp_service_account
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Force, "force", `Force update even if there are dependent external locations or external tables.`)
	flags.EnumVar(cmd.Flags(), &updateReq.IsolationMode, "isolation-mode", `Whether the current securable is accessible from all workspaces or a specific set of workspaces. Supported values: [ISOLATION_MODE_ISOLATED, ISOLATION_MODE_OPEN]`, "IsolationMode", "ISOLATION_MODE_ISOLATED", "ISOLATION_MODE_OPEN")
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the storage credential.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `Username of current owner of credential.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.ReadOnly, "read-only", `Whether the credential is usable only for read operations.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.SkipValidation, "skip-validation", `Supplying true to this argument skips validation of the updated credential.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a credential.`
	cmd.Long = `Update a credential.

  Updates a storage credential on the metastore.

  The caller must be the owner of the storage credential or a metastore admin.
  If the caller is a metastore admin, only the **owner** field can be changed.

  Arguments:
    NAME: Name of the storage credential.`

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
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.NameArg = new(args[0])

		sdkReq, err := uccredentials.UpdateStorageCredentialRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.UpdateStorageCredential(ctx, *sdkReq)
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
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// start validate command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var validateOverrides []func(
	*cobra.Command,
	*uccredentials.ValidateStorageCredentialRequest,
)

func newValidate() *cobra.Command {
	cmd := &cobra.Command{}
	var validateReq uccredentials.ValidateStorageCredentialRequest

	var validateJson flags.JsonFlag

	cmd.Flags().Var(&validateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	// TODO: complex arg: databricks_gcp_service_account
	flags.StringPtrVar(cmd.Flags(), &validateReq.ExternalLocationName, "external-location-name", `The name of an existing external location to validate.`)
	flags.BoolPtrVar(cmd.Flags(), &validateReq.ReadOnly, "read-only", `Whether the storage credential is only usable for read operations.`)
	flags.StringPtrVar(cmd.Flags(), &validateReq.StorageCredentialName, "storage-credential-name", `Required.`)
	flags.StringPtrVar(cmd.Flags(), &validateReq.Url, "url", `The external location url to validate.`)

	cmd.Use = "validate"
	cmd.Short = `Validate a storage credential.`
	cmd.Long = `Validate a storage credential.

  Validates a storage credential. At least one of __external_location_name__ and
  __url__ need to be provided. If only one of them is provided, it will be used
  for validation. And if both are provided, the __url__ will be used for
  validation, and __external_location_name__ will be ignored when checking
  overlapping urls.

  Either the __storage_credential_name__ or the cloud-specific credential must
  be provided.

  The caller must be a metastore admin or the storage credential owner or have
  the **CREATE_EXTERNAL_LOCATION** privilege on the metastore and the storage
  credential.`

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
			diags := validateJson.Unmarshal(&validateReq)
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

		sdkReq, err := uccredentials.ValidateStorageCredentialRequestToSDK(&validateReq)
		if err != nil {
			return err
		}
		response, err := w.Credentials.ValidateStorageCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.ValidateStorageCredentialResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range validateOverrides {
		fn(cmd, &validateReq)
	}

	return cmd
}

// end service StorageCredentials
