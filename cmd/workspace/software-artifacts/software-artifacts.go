// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package software_artifacts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/softwareartifacts"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "software-artifacts",
	Short:   `*Public Preview* Manage software artifacts and their versions in Unity Catalog.`,
	GroupID: "catalog",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `This command is in Public Preview and may change without notice.

Manage software artifacts and their versions in Unity Catalog.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PUBLIC_PREVIEW",
		"launch_stage_display": "Public Preview",
	}

	// Add methods
	cmd.AddCommand(newDeleteSoftwareArtifact())
	cmd.AddCommand(newDeleteSoftwareArtifactVersion())
	cmd.AddCommand(newGetSoftwareArtifact())
	cmd.AddCommand(newGetSoftwareArtifactVersion())
	cmd.AddCommand(newListSoftwareArtifactVersions())
	cmd.AddCommand(newListSoftwareArtifacts())
	cmd.AddCommand(newResolveSoftwareArtifactVersion())
	cmd.AddCommand(newUpdateSoftwareArtifact())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start delete-software-artifact command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSoftwareArtifactOverrides []func(
	*cobra.Command,
	*softwareartifacts.DeleteSoftwareArtifactRequest,
)

func newDeleteSoftwareArtifact() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSoftwareArtifactReq softwareartifacts.DeleteSoftwareArtifactRequest

	cmd.Flags().BoolVar(&deleteSoftwareArtifactReq.Force, "force", deleteSoftwareArtifactReq.Force, `Force deletion even if the software artifact has versions, deleting them with it.`)

	cmd.Use = "delete-software-artifact NAME"
	cmd.Short = `*Public Preview* Delete a software artifact.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a software artifact.

  Deletes a software artifact. Fails if the artifact still has versions unless
  force is set.

  Arguments:
    NAME: Resource name of the software artifact, of the form
      software-artifacts/{catalog}.{schema}.{artifact}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSoftwareArtifactReq.Name = args[0]

		err = w.SoftwareArtifacts.DeleteSoftwareArtifact(ctx, deleteSoftwareArtifactReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSoftwareArtifactOverrides {
		fn(cmd, &deleteSoftwareArtifactReq)
	}

	return cmd
}

// start delete-software-artifact-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSoftwareArtifactVersionOverrides []func(
	*cobra.Command,
	*softwareartifacts.DeleteSoftwareArtifactVersionRequest,
)

func newDeleteSoftwareArtifactVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSoftwareArtifactVersionReq softwareartifacts.DeleteSoftwareArtifactVersionRequest

	cmd.Use = "delete-software-artifact-version NAME"
	cmd.Short = `*Public Preview* Delete a software artifact version.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a software artifact version.

  Deletes a software artifact version.

  Arguments:
    NAME: Resource name of the version, of the form
      software-artifacts/{catalog}.{schema}.{artifact}/versions/{version},
      where {version} is the version's content digest, for example
      sha256:<hex>.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSoftwareArtifactVersionReq.Name = args[0]

		err = w.SoftwareArtifacts.DeleteSoftwareArtifactVersion(ctx, deleteSoftwareArtifactVersionReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSoftwareArtifactVersionOverrides {
		fn(cmd, &deleteSoftwareArtifactVersionReq)
	}

	return cmd
}

// start get-software-artifact command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSoftwareArtifactOverrides []func(
	*cobra.Command,
	*softwareartifacts.GetSoftwareArtifactRequest,
)

func newGetSoftwareArtifact() *cobra.Command {
	cmd := &cobra.Command{}

	var getSoftwareArtifactReq softwareartifacts.GetSoftwareArtifactRequest

	cmd.Use = "get-software-artifact NAME"
	cmd.Short = `*Public Preview* Get a software artifact.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a software artifact.

  Gets a software artifact by its resource name. Returns NOT_FOUND if the
  artifact does not exist.

  Arguments:
    NAME: Resource name of the artifact, of the form
      software-artifacts/{catalog}.{schema}.{artifact}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSoftwareArtifactReq.Name = args[0]

		response, err := w.SoftwareArtifacts.GetSoftwareArtifact(ctx, getSoftwareArtifactReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSoftwareArtifactOverrides {
		fn(cmd, &getSoftwareArtifactReq)
	}

	return cmd
}

// start get-software-artifact-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSoftwareArtifactVersionOverrides []func(
	*cobra.Command,
	*softwareartifacts.GetSoftwareArtifactVersionRequest,
)

func newGetSoftwareArtifactVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var getSoftwareArtifactVersionReq softwareartifacts.GetSoftwareArtifactVersionRequest

	cmd.Use = "get-software-artifact-version NAME"
	cmd.Short = `*Public Preview* Get a software artifact version.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a software artifact version.

  Gets a software artifact version by its resource name. Returns NOT_FOUND if
  the version does not exist.

  Arguments:
    NAME: Resource name of the version, of the form
      software-artifacts/{catalog}.{schema}.{artifact}/versions/{version},
      where {version} is the version's content digest.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSoftwareArtifactVersionReq.Name = args[0]

		response, err := w.SoftwareArtifacts.GetSoftwareArtifactVersion(ctx, getSoftwareArtifactVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSoftwareArtifactVersionOverrides {
		fn(cmd, &getSoftwareArtifactVersionReq)
	}

	return cmd
}

// start list-software-artifact-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSoftwareArtifactVersionsOverrides []func(
	*cobra.Command,
	*softwareartifacts.ListSoftwareArtifactVersionsRequest,
)

func newListSoftwareArtifactVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var listSoftwareArtifactVersionsReq softwareartifacts.ListSoftwareArtifactVersionsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSoftwareArtifactVersionsLimit int

	cmd.Flags().IntVar(&listSoftwareArtifactVersionsReq.PageSize, "page-size", listSoftwareArtifactVersionsReq.PageSize, `Maximum number of versions to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSoftwareArtifactVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSoftwareArtifactVersionsReq.PageToken, "page-token", listSoftwareArtifactVersionsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-software-artifact-versions PARENT"
	cmd.Short = `*Public Preview* List software artifact versions.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List software artifact versions.

  Lists the versions of a software artifact.

  Arguments:
    PARENT: Resource name of the parent artifact, of the form
      software-artifacts/{catalog}.{schema}.{artifact}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSoftwareArtifactVersionsReq.Parent = args[0]

		response := w.SoftwareArtifacts.ListSoftwareArtifactVersions(ctx, listSoftwareArtifactVersionsReq)
		if listSoftwareArtifactVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSoftwareArtifactVersionsLimit)
		}
		if listSoftwareArtifactVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSoftwareArtifactVersionsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSoftwareArtifactVersionsOverrides {
		fn(cmd, &listSoftwareArtifactVersionsReq)
	}

	return cmd
}

// start list-software-artifacts command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSoftwareArtifactsOverrides []func(
	*cobra.Command,
	*softwareartifacts.ListSoftwareArtifactsRequest,
)

func newListSoftwareArtifacts() *cobra.Command {
	cmd := &cobra.Command{}

	var listSoftwareArtifactsReq softwareartifacts.ListSoftwareArtifactsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSoftwareArtifactsLimit int

	cmd.Flags().IntVar(&listSoftwareArtifactsReq.PageSize, "page-size", listSoftwareArtifactsReq.PageSize, `Maximum number of software artifacts to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSoftwareArtifactsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSoftwareArtifactsReq.PageToken, "page-token", listSoftwareArtifactsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-software-artifacts PARENT"
	cmd.Short = `*Public Preview* List software artifacts.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List software artifacts.

  Lists software artifacts in a schema.

  Arguments:
    PARENT: Resource name of the parent schema, of the form
      schemas/{catalog}.{schema}.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSoftwareArtifactsReq.Parent = args[0]

		response := w.SoftwareArtifacts.ListSoftwareArtifacts(ctx, listSoftwareArtifactsReq)
		if listSoftwareArtifactsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSoftwareArtifactsLimit)
		}
		if listSoftwareArtifactsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSoftwareArtifactsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSoftwareArtifactsOverrides {
		fn(cmd, &listSoftwareArtifactsReq)
	}

	return cmd
}

// start resolve-software-artifact-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveSoftwareArtifactVersionOverrides []func(
	*cobra.Command,
	*softwareartifacts.ResolveSoftwareArtifactVersionRequest,
)

func newResolveSoftwareArtifactVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveSoftwareArtifactVersionReq softwareartifacts.ResolveSoftwareArtifactVersionRequest
	var resolveSoftwareArtifactVersionJson flags.JsonFlag

	cmd.Flags().Var(&resolveSoftwareArtifactVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-software-artifact-version PARENT ALIAS"
	cmd.Short = `*Public Preview* Resolve a software artifact version.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Resolve a software artifact version.

  Resolves a software artifact version from its tag alias. Returns NOT_FOUND
  if the artifact or the alias does not exist.

  Arguments:
    PARENT: Resource name of the software artifact containing the alias, of the form
      software-artifacts/{catalog}.{schema}.{artifact}.
    ALIAS: Image tag alias selecting the version. The alias must be a valid OCI image
      tag of at most 128 characters, made of letters, digits, underscores,
      periods, and hyphens, and must not start with a period or hyphen.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'alias' in your JSON input")
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
			diags := resolveSoftwareArtifactVersionJson.Unmarshal(&resolveSoftwareArtifactVersionReq)
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
		resolveSoftwareArtifactVersionReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			resolveSoftwareArtifactVersionReq.Alias = args[1]
		}

		response, err := w.SoftwareArtifacts.ResolveSoftwareArtifactVersion(ctx, resolveSoftwareArtifactVersionReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveSoftwareArtifactVersionOverrides {
		fn(cmd, &resolveSoftwareArtifactVersionReq)
	}

	return cmd
}

// start update-software-artifact command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSoftwareArtifactOverrides []func(
	*cobra.Command,
	*softwareartifacts.UpdateSoftwareArtifactRequest,
)

func newUpdateSoftwareArtifact() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSoftwareArtifactReq softwareartifacts.UpdateSoftwareArtifactRequest
	updateSoftwareArtifactReq.SoftwareArtifact = softwareartifacts.SoftwareArtifact{}
	var updateSoftwareArtifactJson flags.JsonFlag

	cmd.Flags().Var(&updateSoftwareArtifactJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSoftwareArtifactReq.SoftwareArtifact.Name, "name", updateSoftwareArtifactReq.SoftwareArtifact.Name, `Resource name of the artifact, of the form software-artifacts/{catalog}.{schema}.{artifact}.`)
	cmd.Flags().StringVar(&updateSoftwareArtifactReq.SoftwareArtifact.Owner, "owner", updateSoftwareArtifactReq.SoftwareArtifact.Owner, `Principal that owns the artifact, given as a user's email address, a service principal's application ID, or a group name.`)

	cmd.Use = "update-software-artifact NAME UPDATE_MASK"
	cmd.Short = `*Public Preview* Update a software artifact.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a software artifact.

  Updates a software artifact. The name is immutable; only the owner may be
  changed. The destination owner must exist in the account. The caller needs
  MANAGE_ACCESS_CONTROL on the artifact. The response contains its name and the
  requested owner; audit fields are omitted.

  Arguments:
    NAME: Resource name of the artifact, of the form
      software-artifacts/{catalog}.{schema}.{artifact}.
    UPDATE_MASK: Fields to update, relative to software_artifact. Accepts owner or *;
      owner is the only mutable field, so both select the same update.
      Output-only values in the resource are ignored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSoftwareArtifactJson.Unmarshal(&updateSoftwareArtifactReq.SoftwareArtifact)
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
		updateSoftwareArtifactReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSoftwareArtifactReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.SoftwareArtifacts.UpdateSoftwareArtifact(ctx, updateSoftwareArtifactReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSoftwareArtifactOverrides {
		fn(cmd, &updateSoftwareArtifactReq)
	}

	return cmd
}

// end service SoftwareArtifacts
