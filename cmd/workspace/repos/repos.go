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
	Annotations: map[string]string{
		"package": "workspace",
	},
}

// start create command

var createReq workspace.CreateRepo
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Path, "path", createReq.Path, `Desired path for the repo in the workspace.`)
	// TODO: complex arg: sparse_checkout

}

var createCmd = &cobra.Command{
	Use:   "create URL PROVIDER",
	Short: `Create a repo.`,
	Long: `Create a repo.
  
  Creates a repo in the workspace and links it to the remote Git repo specified.
  Note that repos created programmatically must be linked to a remote Git repo,
  unlike repos created in the browser.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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
	},
}

// start delete command

var deleteReq workspace.DeleteRepoRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete REPO_ID",
	Short: `Delete a repo.`,
	Long: `Delete a repo.
  
  Deletes the specified repo.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No REPO_ID argument specified. Loading names for Repos drop-down."
				names, err := w.Repos.RepoInfoPathToIdMap(ctx, workspace.ListReposRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Repos drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID for the corresponding repo to access")
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
		}

		err = w.Repos.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq workspace.GetRepoRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get REPO_ID",
	Short: `Get a repo.`,
	Long: `Get a repo.
  
  Returns the repo with the given repo ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No REPO_ID argument specified. Loading names for Repos drop-down."
				names, err := w.Repos.RepoInfoPathToIdMap(ctx, workspace.ListReposRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Repos drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID for the corresponding repo to access")
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
		}

		response, err := w.Repos.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq workspace.ListReposRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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
	},
}

// start update command

var updateReq workspace.UpdateRepo
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Branch, "branch", updateReq.Branch, `Branch that the local version of the repo is checked out to.`)
	// TODO: complex arg: sparse_checkout
	updateCmd.Flags().StringVar(&updateReq.Tag, "tag", updateReq.Tag, `Tag that the local version of the repo is checked out to.`)

}

var updateCmd = &cobra.Command{
	Use:   "update REPO_ID",
	Short: `Update a repo.`,
	Long: `Update a repo.
  
  Updates the repo to a different branch or tag, or updates the repo to the
  latest commit on the same branch.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No REPO_ID argument specified. Loading names for Repos drop-down."
				names, err := w.Repos.RepoInfoPathToIdMap(ctx, workspace.ListReposRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Repos drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID for the corresponding repo to access")
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
		}

		err = w.Repos.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Repos
