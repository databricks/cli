package repos

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/sdkclients"
	repos "github.com/databricks/cli/libs/sdktypes/repos/v1"
	sdkrepos "github.com/databricks/sdk-go/repos/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *repos.ListReposRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .Id}}	{{.Path}}	{{.Branch|blue}}	{{.Url|cyan}}
	{{end}}`)
}

func createOverride(createCmd *cobra.Command, createReq *repos.CreateRepoRequest) {
	createCmd.Use = "create URL [PROVIDER]"
	createCmd.Args = func(cmd *cobra.Command, args []string) error {
		// If the provider argument is not specified, we try to detect it from the URL.
		check := cobra.RangeArgs(1, 2)
		if cmd.Flags().Changed("json") {
			check = root.ExactArgs(0)
		}
		return check(cmd, args)
	}

	createJson := createCmd.Flag("json").Value.(*flags.JsonFlag)
	createCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			createReq.Url = new(args[0])
			if len(args) > 1 {
				createReq.Provider = new(args[1])
			} else {
				provider := DetectProvider(args[0])
				if provider == "" {
					return fmt.Errorf(
						"could not detect provider from URL %q; please specify", args[0])
				}
				createReq.Provider = new(provider)
			}
		}
		sdkReq, err := repos.CreateRepoRequestToSDK(createReq)
		if err != nil {
			return err
		}
		response, err := w.Repos.CreateRepo(ctx, *sdkReq)
		if err != nil {
			return err
		}
		out, err := repos.CreateRepoResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}
}

func deleteOverride(deleteCmd *cobra.Command, deleteReq *sdkrepos.DeleteRepoRequest) {
	deleteCmd.Use = "delete REPO_ID_OR_PATH"
	deleteCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		id, err := repoArgumentToRepoID(ctx, w, args)
		if err != nil {
			return err
		}
		deleteReq.Id = &id
		_, err = w.Repos.DeleteRepo(ctx, *deleteReq)
		if err != nil {
			return err
		}
		return nil
	}
}

func getOverride(getCmd *cobra.Command, getReq *sdkrepos.GetRepoRequest) {
	getCmd.Use = "get REPO_ID_OR_PATH"
	getCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		id, err := repoArgumentToRepoID(ctx, w, args)
		if err != nil {
			return err
		}
		getReq.Id = &id

		response, err := w.Repos.GetRepo(ctx, *getReq)
		if err != nil {
			return err
		}
		out, err := repos.GetRepoResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}
}

func updateOverride(updateCmd *cobra.Command, updateReq *repos.UpdateRepoRequest) {
	updateCmd.Use = "update REPO_ID_OR_PATH"

	updateJson := updateCmd.Flag("json").Value.(*flags.JsonFlag)
	updateCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		var diags diag.Diagnostics
		w := cmdctx.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			diags = updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			id, err := repoArgumentToRepoID(ctx, w, args)
			if err != nil {
				return err
			}
			updateReq.Id = &id
		}

		sdkReq, err := repos.UpdateRepoRequestToSDK(updateReq)
		if err != nil {
			return err
		}
		_, err = w.Repos.UpdateRepo(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}
}

func repoArgumentToRepoID(ctx context.Context, w *sdkclients.Clients, args []string) (int64, error) {
	// ---- Begin copy from cmd/workspace/repos/repos.go ----
	if len(args) == 0 {
		sp := cmdio.NewSpinner(ctx)
		sp.Update("No REPO_ID argument specified. Loading names for Repos drop-down.")
		names, err := cmdgen.NamedIdMap(ctx, w.Repos.ListReposIter(ctx, sdkrepos.ListReposRequest{}), []string{"Path"}, []string{"Id"})
		sp.Close()
		if err != nil {
			return 0, fmt.Errorf("failed to load names for Repos drop-down. Please manually specify required arguments. Original error: %w", err)
		}
		id, err := cmdio.Select(ctx, names, "The ID for the corresponding repo to access")
		if err != nil {
			return 0, err
		}
		args = append(args, id)
	}
	if len(args) != 1 {
		return 0, errors.New("expected to have the id for the corresponding repo to access")
	}
	// ---- End copy from cmd/workspace/repos/repos.go ----

	// If the argument is a repo ID, return it.
	arg := args[0]
	id, err := strconv.ParseInt(arg, 10, 64)
	if err == nil {
		return id, nil
	}

	// Look up the path via get-status. We don't gate on the object type: Git-CLI
	// folders report DIRECTORY rather than REPO, and the repos API is the
	// authority on whether the ID resolves to a repo.
	oi, err := w.Workspace.GetStatusByPath(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("failed to look up repo by path: %w", err)
	}
	return oi.ObjectId, nil
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	createOverrides = append(createOverrides, createOverride)
	deleteOverrides = append(deleteOverrides, deleteOverride)
	getOverrides = append(getOverrides, getOverride)
	updateOverrides = append(updateOverrides, updateOverride)
}
