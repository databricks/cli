// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage_credentials

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
		Annotations: map[string]string{
			"package": "catalog",
		},
	}

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
	*catalog.CreateStorageCredential,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateStorageCredential
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Comment associated with the credential.`)
	// TODO: output-only field
	cmd.Flags().BoolVar(&createReq.ReadOnly, "read-only", createReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	cmd.Flags().BoolVar(&createReq.SkipValidation, "skip-validation", createReq.SkipValidation, `Supplying true to this argument skips validation of the created credential.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a storage credential.`
	cmd.Long = `Create a storage credential.
  
  Creates a new storage credential.

  Arguments:
    NAME: The credential name. The name must be unique within the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}

		response, err := w.StorageCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteStorageCredentialRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteStorageCredentialRequest

	// TODO: short flags

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external locations or external tables.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.
  
  Deletes a storage credential from the metastore. The caller must be an owner
  of the storage credential.

  Arguments:
    NAME: Name of the storage credential.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Storage Credentials drop-down."
			names, err := w.StorageCredentials.StorageCredentialInfoNameToIdMap(ctx, catalog.ListStorageCredentialsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Storage Credentials drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Name of the storage credential")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of the storage credential")
		}
		deleteReq.Name = args[0]

		err = w.StorageCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetStorageCredentialRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetStorageCredentialRequest

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get a credential.`
	cmd.Long = `Get a credential.
  
  Gets a storage credential from the metastore. The caller must be a metastore
  admin, the owner of the storage credential, or have some permission on the
  storage credential.

  Arguments:
    NAME: Name of the storage credential.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Storage Credentials drop-down."
			names, err := w.StorageCredentials.StorageCredentialInfoNameToIdMap(ctx, catalog.ListStorageCredentialsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Storage Credentials drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Name of the storage credential")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of the storage credential")
		}
		getReq.Name = args[0]

		response, err := w.StorageCredentials.Get(ctx, getReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*catalog.ListStorageCredentialsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListStorageCredentialsRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of storage credentials to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List credentials.`
	cmd.Long = `List credentials.
  
  Gets an array of storage credentials (as __StorageCredentialInfo__ objects).
  The array is limited to only those storage credentials the caller has
  permission to access. If the caller is a metastore admin, retrieval of
  credentials is unrestricted. For unpaginated request, there is no guarantee of
  a specific ordering of the elements in the array. For paginated request,
  elements are ordered by their name.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.StorageCredentials.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateStorageCredential,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateStorageCredential
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Comment associated with the credential.`)
	// TODO: output-only field
	cmd.Flags().BoolVar(&updateReq.Force, "force", updateReq.Force, `Force update even if there are dependent external locations or external tables.`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the storage credential.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of credential.`)
	cmd.Flags().BoolVar(&updateReq.ReadOnly, "read-only", updateReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	cmd.Flags().BoolVar(&updateReq.SkipValidation, "skip-validation", updateReq.SkipValidation, `Supplying true to this argument skips validation of the updated credential.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a credential.`
	cmd.Long = `Update a credential.
  
  Updates a storage credential on the metastore.

  Arguments:
    NAME: Name of the storage credential.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Storage Credentials drop-down."
			names, err := w.StorageCredentials.StorageCredentialInfoNameToIdMap(ctx, catalog.ListStorageCredentialsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Storage Credentials drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Name of the storage credential")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of the storage credential")
		}
		updateReq.Name = args[0]

		response, err := w.StorageCredentials.Update(ctx, updateReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// start validate command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var validateOverrides []func(
	*cobra.Command,
	*catalog.ValidateStorageCredential,
)

func newValidate() *cobra.Command {
	cmd := &cobra.Command{}

	var validateReq catalog.ValidateStorageCredential
	var validateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&validateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_managed_identity
	// TODO: complex arg: azure_service_principal
	// TODO: complex arg: cloudflare_api_token
	// TODO: output-only field
	cmd.Flags().StringVar(&validateReq.ExternalLocationName, "external-location-name", validateReq.ExternalLocationName, `The name of an existing external location to validate.`)
	cmd.Flags().BoolVar(&validateReq.ReadOnly, "read-only", validateReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	// TODO: any: storage_credential_name
	cmd.Flags().StringVar(&validateReq.Url, "url", validateReq.Url, `The external location url to validate.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = validateJson.Unmarshal(&validateReq)
			if err != nil {
				return err
			}
		}

		response, err := w.StorageCredentials.Validate(ctx, validateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newValidate())
	})
}

// end service StorageCredentials
