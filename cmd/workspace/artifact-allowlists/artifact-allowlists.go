// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package artifact_allowlists

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
		Use:   "artifact-allowlists",
		Short: `In Databricks Runtime 13.3 and above, you can add libraries and init scripts to the allowlist in UC so that users can leverage these artifacts on compute configured with shared access mode.`,
		Long: `In Databricks Runtime 13.3 and above, you can add libraries and init scripts
  to the allowlist in UC so that users can leverage these artifacts on compute
  configured with shared access mode.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetArtifactAllowlistRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetArtifactAllowlistRequest

	cmd.Use = "get ARTIFACT_TYPE"
	cmd.Short = `Get an artifact allowlist.`
	cmd.Long = `Get an artifact allowlist.
  
  Get the artifact allowlist of a certain artifact type. The caller must be a
  metastore admin or have the **MANAGE ALLOWLIST** privilege on the metastore.

  Arguments:
    ARTIFACT_TYPE: The artifact type of the allowlist. 
      Supported values: [INIT_SCRIPT, LIBRARY_JAR, LIBRARY_MAVEN]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.ArtifactType)
		if err != nil {
			return fmt.Errorf("invalid ARTIFACT_TYPE: %s", args[0])
		}

		response, err := w.ArtifactAllowlists.Get(ctx, getReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.SetArtifactAllowlist,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.SetArtifactAllowlist
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update ARTIFACT_TYPE"
	cmd.Short = `Set an artifact allowlist.`
	cmd.Long = `Set an artifact allowlist.
  
  Set the artifact allowlist of a certain artifact type. The whole artifact
  allowlist is replaced with the new allowlist. The caller must be a metastore
  admin or have the **MANAGE ALLOWLIST** privilege on the metastore.

  Arguments:
    ARTIFACT_TYPE: The artifact type of the allowlist. 
      Supported values: [INIT_SCRIPT, LIBRARY_JAR, LIBRARY_MAVEN]`

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
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		_, err = fmt.Sscan(args[0], &updateReq.ArtifactType)
		if err != nil {
			return fmt.Errorf("invalid ARTIFACT_TYPE: %s", args[0])
		}

		response, err := w.ArtifactAllowlists.Update(ctx, updateReq)
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

// end service ArtifactAllowlists
