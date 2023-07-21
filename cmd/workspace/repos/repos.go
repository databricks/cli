// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package repos

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: `The Repos API allows users to manage their git repos.`,
		Long: `The Repos API allows users to manage their git repos. Users can use the API to
  access all repos that they have manage permissions on.
  
  Databricks Repos is a visual Git client in Databricks. It supports common Git
  operations such a cloning a repository, committing and pushing, pulling,
  branch management, and visual comparison of diffs when committing.
  
  Within Repos you can develop code in notebooks or other files and follow data
  science and engineering code development best practices using Git for version
  control, collaboration, and CI/CD.`,
		GroupID: "workspace",
		Annotations: map[string]string{
			"package": "workspace",
		},
	}

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
	*workspace.CreateRepo,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq workspace.CreateRepo
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Path, "path", createReq.Path, `Desired path for the repo in the workspace.`)
	// TODO: complex arg: sparse_checkout

	cmd.Use = "create URL PROVIDER"
	cmd.Short = `Create a repo.`
	cmd.Long = `Create a repo.
  
  Creates a repo in the workspace and links it to the remote Git repo specified.
  Note that repos created programmatically must be linked to a remote Git repo,
  unlike repos created in the browser.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
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
		} else {
			createReq.Url = args[0]
			createReq.Provider = args[1]
		}

		response, err := w.Repos.Create(ctx, createReq)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*workspace.DeleteRepoRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq workspace.DeleteRepoRequest

	// TODO: short flags

	cmd.Use = "delete REPO_ID"
	cmd.Short = `Delete a repo.`
	cmd.Long = `Delete a repo.
  
  Deletes the specified repo.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.RepoId)
		if err != nil {
			return fmt.Errorf("invalid REPO_ID: %s", args[0])
		}

		err = w.Repos.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*workspace.GetRepoRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq workspace.GetRepoRequest

	// TODO: short flags

	cmd.Use = "get REPO_ID"
	cmd.Short = `Get a repo.`
	cmd.Long = `Get a repo.
  
  Returns the repo with the given repo ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.RepoId)
		if err != nil {
			return fmt.Errorf("invalid REPO_ID: %s", args[0])
		}

		response, err := w.Repos.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*workspace.ListReposRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq workspace.ListReposRequest
	var listJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&listReq.NextPageToken, "next-page-token", listReq.NextPageToken, `Token used to get the next page of results.`)
	cmd.Flags().StringVar(&listReq.PathPrefix, "path-prefix", listReq.PathPrefix, `Filters repos that have paths starting with the given path prefix.`)

	cmd.Use = "list"
	cmd.Short = `Get repos.`
	cmd.Long = `Get repos.
  
  Returns repos that the calling user has Manage permissions on. Results are
  paginated with each page containing twenty repos.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Repos.ListAll(ctx, listReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*workspace.UpdateRepo,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq workspace.UpdateRepo
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Branch, "branch", updateReq.Branch, `Branch that the local version of the repo is checked out to.`)
	// TODO: complex arg: sparse_checkout
	cmd.Flags().StringVar(&updateReq.Tag, "tag", updateReq.Tag, `Tag that the local version of the repo is checked out to.`)

	cmd.Use = "update REPO_ID"
	cmd.Short = `Update a repo.`
	cmd.Long = `Update a repo.
  
  Updates the repo to a different branch or tag, or updates the repo to the
  latest commit on the same branch.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

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
		_, err = fmt.Sscan(args[0], &updateReq.RepoId)
		if err != nil {
			return fmt.Errorf("invalid REPO_ID: %s", args[0])
		}

		err = w.Repos.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
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

// end service Repos
