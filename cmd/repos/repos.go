// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package repos

import (
	"fmt"

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

}

var createCmd = &cobra.Command{
	Use:   "create URL PROVIDER",
	Short: `Create a repo.`,
	Long: `Create a repo.
  
  Creates a repo in the workspace and links it to the remote Git repo specified.
  Note that repos created programmatically must be linked to a remote Git repo,
  unlike repos created in the browser.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		createReq.Url = args[0]
		createReq.Provider = args[1]

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

}

var deleteCmd = &cobra.Command{
	Use:   "delete REPO_ID",
	Short: `Delete a repo.`,
	Long: `Delete a repo.
  
  Deletes the specified repo.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Repos.RepoInfoPathToIdMap(ctx, repos.List{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The ID for the corresponding repo to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding repo to access")
		}
		_, err = fmt.Sscan(args[0], &deleteReq.RepoId)
		if err != nil {
			return fmt.Errorf("invalid REPO_ID: %s", args[0])
		}

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

}

var getCmd = &cobra.Command{
	Use:   "get REPO_ID",
	Short: `Get a repo.`,
	Long: `Get a repo.
  
  Returns the repo with the given repo ID.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Repos.RepoInfoPathToIdMap(ctx, repos.List{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The ID for the corresponding repo to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding repo to access")
		}
		_, err = fmt.Sscan(args[0], &getReq.RepoId)
		if err != nil {
			return fmt.Errorf("invalid REPO_ID: %s", args[0])
		}

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

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
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
	updateCmd.Flags().StringVar(&updateReq.Tag, "tag", updateReq.Tag, `Tag that the local version of the repo is checked out to.`)

}

var updateCmd = &cobra.Command{
	Use:   "update REPO_ID",
	Short: `Update a repo.`,
	Long: `Update a repo.
  
  Updates the repo to a different branch or tag, or updates the repo to the
  latest commit on the same branch.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Repos.RepoInfoPathToIdMap(ctx, repos.List{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The ID for the corresponding repo to access")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding repo to access")
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
	},
}

// end service Repos

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
