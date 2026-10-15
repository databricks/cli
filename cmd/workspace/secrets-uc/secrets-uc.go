// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package secrets_uc

import (
	"encoding/json"
	"errors"
	"fmt"

	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	sdkcoretypes "github.com/databricks/sdk-go/core/types"
	"github.com/spf13/cobra"

	ucsecrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	sdkucsecrets "github.com/databricks/sdk-go/uc/secrets/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets-uc",
		Short: `A secret is a Unity Catalog securable object that stores sensitive credential data (such as passwords, tokens, and keys) within a three-level namespace (**catalog_name.schema_name.secret_name**).`,
		Long: `A secret is a Unity Catalog securable object that stores sensitive credential
  data (such as passwords, tokens, and keys) within a three-level namespace
  (**catalog_name.schema_name.secret_name**).

  Secrets can be managed using standard Unity Catalog permissions and are scoped
  to a schema within a catalog.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateSecret())
	cmd.AddCommand(newDeleteSecret())
	cmd.AddCommand(newGetSecret())
	cmd.AddCommand(newListSecrets())
	cmd.AddCommand(newUpdateSecret())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSecretOverrides []func(
	*cobra.Command,
	*ucsecrets.CreateSecretRequest,
)

func newCreateSecret() *cobra.Command {
	cmd := &cobra.Command{}
	var createSecretReq ucsecrets.CreateSecretRequest
	createSecretReq.Secret = &ucsecrets.Secret{}

	var createSecretJson flags.JsonFlag

	cmd.Flags().Var(&createSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSecretReq.Secret.Comment, "comment", `User-provided free-form text description of the secret.`)
	var expireTimeParam string
	cmd.Flags().StringVar(&expireTimeParam, "expire-time", expireTimeParam, `User-provided expiration time of the secret.`)
	flags.StringPtrVar(cmd.Flags(), &createSecretReq.Secret.Owner, "owner", `The owner of the secret.`)

	cmd.Use = "create-secret NAME CATALOG_NAME SCHEMA_NAME VALUE"
	cmd.Short = `Create a secret.`
	cmd.Long = `Create a secret.

  Creates a new secret in Unity Catalog.

  You must be the owner of the parent schema or have the **CREATE_SECRET** and
  **USE SCHEMA** privileges on the parent schema and **USE CATALOG** on the
  parent catalog.

  The secret is stored in the specified catalog and schema, and the **value**
  field contains the sensitive data to be securely stored.

  Arguments:
    NAME: The name of the secret, relative to its parent schema.
    CATALOG_NAME: The name of the catalog where the schema and the secret reside.
    SCHEMA_NAME: The name of the schema where the secret resides.
    VALUE: The secret value to store. This field is input-only and is not returned in
      responses — use the **effective_value** field (via GetSecret with
      **include_value** set to true) to read the secret value. The maximum size
      is 60 KiB (pre-encryption). Accepted content includes passwords, tokens,
      keys, and other sensitive credential data.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'catalog_name', 'schema_name', 'value' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createSecretJson.Unmarshal(&createSecretReq.Secret)
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
			createSecretReq.Secret.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.CatalogName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.SchemaName = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.Value = new(args[3])
		}

		if expireTimeParam != "" {
			var expireTimeField sdkcoretypes.Time
			err = json.Unmarshal([]byte(strconv.Quote(expireTimeParam)), &expireTimeField)
			if err != nil {
				return fmt.Errorf("invalid EXPIRE_TIME: %s", expireTimeParam)
			}
			createSecretReq.Secret.ExpireTime = &expireTimeField
		}

		sdkReq, err := ucsecrets.CreateSecretRequestToSDK(&createSecretReq)
		if err != nil {
			return err
		}
		response, err := w.SecretsUc.CreateSecret(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucsecrets.SecretFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSecretOverrides {
		fn(cmd, &createSecretReq)
	}

	return cmd
}

// start delete-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSecretOverrides []func(
	*cobra.Command,
	*sdkucsecrets.DeleteSecretRequest,
)

func newDeleteSecret() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSecretReq sdkucsecrets.DeleteSecretRequest

	cmd.Use = "delete-secret FULL_NAME"
	cmd.Short = `Delete a secret.`
	cmd.Long = `Delete a secret.

  Deletes a secret by its three-level (fully qualified) name.

  You must be the owner of the secret or a metastore admin.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).`

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

		deleteSecretReq.FullName = new(args[0])

		sdkReq := &deleteSecretReq
		err = w.SecretsUc.DeleteSecret(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSecretOverrides {
		fn(cmd, &deleteSecretReq)
	}

	return cmd
}

// start get-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSecretOverrides []func(
	*cobra.Command,
	*ucsecrets.GetSecretRequest,
)

func newGetSecret() *cobra.Command {
	cmd := &cobra.Command{}
	var getSecretReq ucsecrets.GetSecretRequest

	flags.BoolPtrVar(cmd.Flags(), &getSecretReq.IncludeValue, "include-value", `Whether to include the secret value in the response.`)

	cmd.Use = "get-secret FULL_NAME"
	cmd.Short = `Get a secret.`
	cmd.Long = `Get a secret.

  Gets a secret by its three-level (fully qualified) name.

  You must be a metastore admin, the owner of the secret, or have the **MANAGE**
  privilege on the secret.

  The secret value isn't returned by default. To retrieve it, you must also have
  the **READ_SECRET** privilege and set **include_value** to true in the
  request.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).`

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

		getSecretReq.FullName = new(args[0])

		sdkReq, err := ucsecrets.GetSecretRequestToSDK(&getSecretReq)
		if err != nil {
			return err
		}
		response, err := w.SecretsUc.GetSecret(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucsecrets.SecretFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSecretOverrides {
		fn(cmd, &getSecretReq)
	}

	return cmd
}

// start list-secrets command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSecretsOverrides []func(
	*cobra.Command,
	*ucsecrets.ListSecretsRequest,
)

func newListSecrets() *cobra.Command {
	cmd := &cobra.Command{}
	var listSecretsReq ucsecrets.ListSecretsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSecretsLimit int

	flags.StringPtrVar(cmd.Flags(), &listSecretsReq.CatalogName, "catalog-name", `The name of the catalog under which to list secrets.`)
	flags.IntPtrVar(cmd.Flags(), &listSecretsReq.PageSize, "page-size", `Maximum number of secrets to return.`)
	flags.StringPtrVar(cmd.Flags(), &listSecretsReq.SchemaName, "schema-name", `The name of the schema under which to list secrets.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSecretsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSecretsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-secrets"
	cmd.Short = `List secrets.`
	cmd.Long = `List secrets.

  Lists secrets in Unity Catalog.

  You must be a metastore admin, the owner of the secret, or have the **MANAGE**
  privilege on the secret.

  Both **catalog_name** and **schema_name** must be specified together to filter
  secrets within a specific schema. Results are paginated; use the
  **page_token** field from the response to retrieve subsequent pages.`

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

		sdkReq, err := ucsecrets.ListSecretsRequestToSDK(&listSecretsReq)
		if err != nil {
			return err
		}
		if listSecretsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSecretsLimit)
		}
		if listSecretsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSecretsLimit)
		}
		response := w.SecretsUc.ListSecretsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucsecrets.SecretFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSecretsOverrides {
		fn(cmd, &listSecretsReq)
	}

	return cmd
}

// start update-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSecretOverrides []func(
	*cobra.Command,
	*ucsecrets.UpdateSecretRequest,
)

func newUpdateSecret() *cobra.Command {
	cmd := &cobra.Command{}
	var updateSecretReq ucsecrets.UpdateSecretRequest
	updateSecretReq.Secret = &ucsecrets.Secret{}

	var updateSecretJson flags.JsonFlag

	cmd.Flags().Var(&updateSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateSecretReq.Secret.Comment, "comment", `User-provided free-form text description of the secret.`)
	var expireTimeParam string
	cmd.Flags().StringVar(&expireTimeParam, "expire-time", expireTimeParam, `User-provided expiration time of the secret.`)
	flags.StringPtrVar(cmd.Flags(), &updateSecretReq.Secret.Owner, "owner", `The owner of the secret.`)

	cmd.Use = "update-secret FULL_NAME UPDATE_MASK NAME CATALOG_NAME SCHEMA_NAME VALUE"
	cmd.Short = `Update a secret.`
	cmd.Long = `Update a secret.

  Updates an existing secret in Unity Catalog.

  You must be the owner of the secret or a metastore admin. If you are a
  metastore admin, only the **owner** field can be changed.

  Use the **update_mask** field to specify which fields to update. Supported
  updatable fields include **value**, **comment**, **owner**, and
  **expire_time**.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).
    UPDATE_MASK: The field mask specifying which fields of the secret to update. - If
      **update_mask** is **"*"**, all fields specified in **secret** are
      updated. - If **update_mask** specifies one or more fields, only those
      fields are updated. Each specified field must be set in **secret**.
      Supported fields: **value**, **comment**, **owner**, **expire_time**. To
      change the secret name, delete and recreate the secret.
    NAME: The name of the secret, relative to its parent schema.
    CATALOG_NAME: The name of the catalog where the schema and the secret reside.
    SCHEMA_NAME: The name of the schema where the secret resides.
    VALUE: The secret value to store. This field is input-only and is not returned in
      responses — use the **effective_value** field (via GetSecret with
      **include_value** set to true) to read the secret value. The maximum size
      is 60 KiB (pre-encryption). Accepted content includes passwords, tokens,
      keys, and other sensitive credential data.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only FULL_NAME, UPDATE_MASK as positional arguments. Provide 'name', 'catalog_name', 'schema_name', 'value' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateSecretJson.Unmarshal(&updateSecretReq.Secret)
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
		updateSecretReq.FullName = new(args[0])
		if args[1] != "" {
			updateSecretReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.Name = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.CatalogName = new(args[3])
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.SchemaName = new(args[4])
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.Value = new(args[5])
		}

		if expireTimeParam != "" {
			var expireTimeField sdkcoretypes.Time
			err = json.Unmarshal([]byte(strconv.Quote(expireTimeParam)), &expireTimeField)
			if err != nil {
				return fmt.Errorf("invalid EXPIRE_TIME: %s", expireTimeParam)
			}
			updateSecretReq.Secret.ExpireTime = &expireTimeField
		}

		sdkReq, err := ucsecrets.UpdateSecretRequestToSDK(&updateSecretReq)
		if err != nil {
			return err
		}
		response, err := w.SecretsUc.UpdateSecret(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucsecrets.SecretFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSecretOverrides {
		fn(cmd, &updateSecretReq)
	}

	return cmd
}

// end service SecretsUc
