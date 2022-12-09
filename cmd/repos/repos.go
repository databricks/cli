package repos

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
}

// start create command

var createReq repos.CreateRepo

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Path, "path", createReq.Path, `Desired path for the repo in the workspace.`)
	createCmd.Flags().StringVar(&createReq.Provider, "provider", createReq.Provider, `Git provider.`)
	createCmd.Flags().StringVar(&createReq.Url, "url", createReq.Url, `URL of the Git repository to be linked.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a repo.`,
	Long: `Create a repo.
  
  Creates a repo in the workspace and links it to the remote Git repo specified.
  Note that repos created programmatically must be linked to a remote Git repo,
  unlike repos created in the browser.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq repos.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.RepoId, "repo-id", deleteReq.RepoId, `The ID for the corresponding repo to access.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a repo.`,
	Long: `Delete a repo.
  
  Deletes the specified repo.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Repos.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq repos.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.RepoId, "repo-id", getReq.RepoId, `The ID for the corresponding repo to access.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a repo.`,
	Long: `Get a repo.
  
  Returns the repo with the given repo ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq repos.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.NextPageToken, "next-page-token", listReq.NextPageToken, `Token used to get the next page of results.`)
	listCmd.Flags().StringVar(&listReq.PathPrefix, "path-prefix", listReq.PathPrefix, `Filters repos that have paths starting with the given path prefix.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get repos.`,
	Long: `Get repos.
  
  Returns repos that the calling user has Manage permissions on. Results are
  paginated with each page containing twenty repos.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq repos.UpdateRepo

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Branch, "branch", updateReq.Branch, `Branch that the local version of the repo is checked out to.`)
	updateCmd.Flags().Int64Var(&updateReq.RepoId, "repo-id", updateReq.RepoId, `The ID for the corresponding repo to access.`)
	updateCmd.Flags().StringVar(&updateReq.Tag, "tag", updateReq.Tag, `Tag that the local version of the repo is checked out to.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a repo.`,
	Long: `Update a repo.
  
  Updates the repo to a different branch or tag, or updates the repo to the
  latest commit on the same branch.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Repos.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Repos

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
