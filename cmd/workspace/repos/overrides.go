package repos

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .Id}}	{{.Path}}	{{.Branch|blue}}	{{.Url|cyan}}
	{{end}}`)

	createCmd.Use = "create URL [PROVIDER]"
	createCmd.Args = func(cmd *cobra.Command, args []string) error {
		// If the provider argument is not specified, we try to detect it from the URL.
		check := cobra.RangeArgs(1, 2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}
	createCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Url = args[0]
			if len(args) > 1 {
				createReq.Provider = args[1]
			} else {
				createReq.Provider = DetectProvider(createReq.Url)
				if createReq.Provider == "" {
					return fmt.Errorf(
						"could not detect provider from URL %q; please specify", createReq.Url)
				}
			}
		}
		response, err := w.Repos.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	deleteCmd.Use = "delete REPO_ID_OR_PATH"
	deleteCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.RepoId, err = repoArgumentToRepoID(ctx, w, args)
		if err != nil {
			return err
		}
		err = w.Repos.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	getCmd.Use = "get REPO_ID_OR_PATH"
	getCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getReq.RepoId, err = repoArgumentToRepoID(ctx, w, args)
		if err != nil {
			return err
		}

		response, err := w.Repos.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	updateCmd.Use = "update REPO_ID_OR_PATH"
	updateCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.RepoId, err = repoArgumentToRepoID(ctx, w, args)
			if err != nil {
				return err
			}
		}

		err = w.Repos.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	}
}

func repoArgumentToRepoID(ctx context.Context, w *databricks.WorkspaceClient, args []string) (int64, error) {
	// ---- Begin copy from cmd/workspace/repos/repos.go ----
	if len(args) == 0 {
		promptSpinner := cmdio.Spinner(ctx)
		promptSpinner <- "No REPO_ID argument specified. Loading names for Repos drop-down."
		names, err := w.Repos.RepoInfoPathToIdMap(ctx, workspace.ListReposRequest{})
		close(promptSpinner)
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
		return 0, fmt.Errorf("expected to have the id for the corresponding repo to access")
	}
	// ---- End copy from cmd/workspace/repos/repos.go ----

	// If the argument is a repo ID, return it.
	arg := args[0]
	id, err := strconv.ParseInt(arg, 10, 64)
	if err == nil {
		return id, nil
	}

	// If the argument cannot be parsed as a repo ID, try to look it up by name.
	oi, err := w.Workspace.GetStatusByPath(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("failed to look up repo by path: %w", err)
	}
	if oi.ObjectType != workspace.ObjectTypeRepo {
		return 0, fmt.Errorf("object at path %q is not a repo", arg)
	}
	return oi.ObjectId, nil
}
