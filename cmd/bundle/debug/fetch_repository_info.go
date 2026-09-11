package debug

import (
	"encoding/json"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/git"
	"github.com/spf13/cobra"
)

// repositoryInfoOutput mirrors git.RepositoryInfo, which has no JSON tags.
type repositoryInfoOutput struct {
	WorktreeRoot  string `json:"worktree_root"`
	CurrentBranch string `json:"current_branch"`
	LatestCommit  string `json:"latest_commit"`
	OriginURL     string `json:"origin_url"`
}

// NewFetchRepositoryInfoCommand returns a command that reports what
// [git.FetchRepositoryInfoAPI] resolves for a path. It exists so that function
// can be exercised through the CLI on both a fake and a real workspace; nothing
// in the product calls it.
//
// It always reads the workspace API, which [git.FetchRepositoryInfo] only does on
// a Databricks Runtime. That cannot be detected off-cluster, and reading .git is
// covered by the TestFetchRepositoryInfoDotGit integration tests.
func NewFetchRepositoryInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fetch-repository-info",
		Short:  "Report the git metadata the workspace API resolves for a path",
		Args:   root.NoArgs,
		Hidden: true,
	}

	var path string
	cmd.Flags().StringVar(&path, "path", ".", "Path to resolve git metadata for")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		info, err := git.FetchRepositoryInfoAPI(ctx, path, cmdctx.WorkspaceClient(ctx))
		if err != nil {
			return err
		}

		buf, err := json.MarshalIndent(repositoryInfoOutput{
			WorktreeRoot:  info.WorktreeRoot,
			CurrentBranch: info.CurrentBranch,
			LatestCommit:  info.LatestCommit,
			OriginURL:     info.OriginURL,
		}, "", "  ")
		if err != nil {
			return err
		}

		_, err = cmd.OutOrStdout().Write(append(buf, '\n'))
		return err
	}

	return cmd
}
