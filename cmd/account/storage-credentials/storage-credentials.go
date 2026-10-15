// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage_credentials

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:     "storage-credentials",
		Short:   `These APIs manage storage credentials for a particular metastore.`,
		Long:    `These APIs manage storage credentials for a particular metastore.`,
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
	*uccredentials.AccountsCreateStorageCredentialRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq uccredentials.AccountsCreateStorageCredentialRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: credential_info
	flags.BoolPtrVar(cmd.Flags(), &createReq.SkipValidation, "skip-validation", `Optional, default false.`)

	cmd.Use = "create METASTORE_ID"
	cmd.Short = `Create a storage credential.`
	cmd.Long = `Create a storage credential.

  Creates a new storage credential. The request object is specific to the cloud:
  - **AwsIamRole** for AWS credentials - **AzureServicePrincipal** for Azure
  credentials - **GcpServiceAccountKey** for GCP credentials

  The caller must be a metastore admin and have the CREATE_STORAGE_CREDENTIAL
  privilege on the metastore.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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
		createReq.MetastoreId = new(args[0])

		sdkReq, err := uccredentials.AccountsCreateStorageCredentialRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Credentials.CreateAccountsStorageCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.AccountsCreateStorageCredentialResponseFromSDK(response)
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
	*uccredentials.AccountsDeleteStorageCredentialRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq uccredentials.AccountsDeleteStorageCredentialRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force deletion even if the Storage Credential is not empty.`)

	cmd.Use = "delete METASTORE_ID STORAGE_CREDENTIAL_NAME"
	cmd.Short = `Delete a storage credential.`
	cmd.Long = `Delete a storage credential.

  Deletes a storage credential from the metastore. The caller must be an owner
  of the storage credential.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID
    STORAGE_CREDENTIAL_NAME: Name of the storage credential.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.MetastoreId = new(args[0])
		deleteReq.NameArg = new(args[1])

		sdkReq, err := uccredentials.AccountsDeleteStorageCredentialRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := a.Credentials.DeleteAccountsStorageCredential(ctx, *sdkReq)
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
	*sdkuccredentials.AccountsGetStorageCredentialRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkuccredentials.AccountsGetStorageCredentialRequest

	cmd.Use = "get METASTORE_ID STORAGE_CREDENTIAL_NAME"
	cmd.Short = `Gets the named storage credential.`
	cmd.Long = `Gets the named storage credential.

  Gets a storage credential from the metastore. The caller must be a metastore
  admin, the owner of the storage credential, or have a level of privilege on
  the storage credential.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID
    STORAGE_CREDENTIAL_NAME: Required. Name of the storage credential.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.MetastoreId = new(args[0])
		getReq.NameArg = new(args[1])

		sdkReq := &getReq
		response, err := a.Credentials.GetAccountsStorageCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.AccountsGetStorageCredentialResponseFromSDK(response)
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
	*sdkuccredentials.AccountsListStorageCredentialsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq sdkuccredentials.AccountsListStorageCredentialsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list METASTORE_ID"
	cmd.Short = `Get all storage credentials assigned to a metastore.`
	cmd.Long = `Get all storage credentials assigned to a metastore.

  Gets a list of all storage credentials that have been assigned to given
  metastore.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		listReq.MetastoreId = new(args[0])

		sdkReq := &listReq
		response, err := a.Credentials.ListAccountsStorageCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := uccredentials.AccountsListStorageCredentialsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*uccredentials.AccountsUpdateStorageCredentialRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq uccredentials.AccountsUpdateStorageCredentialRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: credential_info
	flags.BoolPtrVar(cmd.Flags(), &updateReq.SkipValidation, "skip-validation", `Optional.`)

	cmd.Use = "update METASTORE_ID STORAGE_CREDENTIAL_NAME"
	cmd.Short = `Updates a storage credential.`
	cmd.Long = `Updates a storage credential.

  Updates a storage credential on the metastore. The caller must be the owner of
  the storage credential. If the caller is a metastore admin, only the **owner**
  credential can be changed.

  Arguments:
    METASTORE_ID: Unity Catalog metastore ID
    STORAGE_CREDENTIAL_NAME: Name of the storage credential.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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
		updateReq.MetastoreId = new(args[0])
		updateReq.NameArg = new(args[1])

		sdkReq, err := uccredentials.AccountsUpdateStorageCredentialRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.Credentials.UpdateAccountsStorageCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := uccredentials.AccountsUpdateStorageCredentialResponseFromSDK(response)
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

// end service AccountStorageCredentials
