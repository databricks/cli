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
}

var createReq repos.CreateRepo

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Path, "path", "", `Desired path for the repo in the workspace.`)
	createCmd.Flags().StringVar(&createReq.Provider, "provider", "", `Git provider.`)
	createCmd.Flags().StringVar(&createReq.Url, "url", "", `URL of the Git repository to be linked.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a repo.`,
	Long: `Create a repo.
  
  Creates a repo in the workspace and links it to the remote Git repo specified.
  Note that repos created programmatically must be linked to a remote Git repo,
  unlike repos created in the browser.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq repos.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.RepoId, "repo-id", 0, `The ID for the corresponding repo to access.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a repo.`,
	Long: `Delete a repo.
  
  Deletes the specified repo.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Repos.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq repos.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.RepoId, "repo-id", 0, `The ID for the corresponding repo to access.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a repo.`,
	Long: `Get a repo.
  
  Returns the repo with the given repo ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var listReq repos.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.NextPageToken, "next-page-token", "", `Token used to get the next page of results.`)
	listCmd.Flags().StringVar(&listReq.PathPrefix, "path-prefix", "", `Filters repos that have paths starting with the given path prefix.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get repos.`,
	Long: `Get repos.
  
  Returns repos that the calling user has Manage permissions on. Results are
  paginated with each page containing twenty repos.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Repos.ListAll(ctx, listReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var updateReq repos.UpdateRepo

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Branch, "branch", "", `Branch that the local version of the repo is checked out to.`)
	updateCmd.Flags().Int64Var(&updateReq.RepoId, "repo-id", 0, `The ID for the corresponding repo to access.`)
	updateCmd.Flags().StringVar(&updateReq.Tag, "tag", "", `Tag that the local version of the repo is checked out to.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a repo.`,
	Long: `Update a repo.
  
  Updates the repo to a different branch or tag, or updates the repo to the
  latest commit on the same branch.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Repos.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
