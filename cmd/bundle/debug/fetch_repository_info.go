package debug

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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
// [git.FetchRepositoryInfo] resolves for a path. It exists so that function can
// be exercised through the CLI on both a fake and a real workspace; nothing in
// the product calls it.
func NewFetchRepositoryInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fetch-repository-info",
		Short:  "Report the git metadata resolved for a path",
		Args:   root.NoArgs,
		Hidden: true,
	}

	var path string
	var workspaceAPI bool
	cmd.Flags().StringVar(&path, "path", ".", "Path to resolve git metadata for")
	cmd.Flags().BoolVar(&workspaceAPI, "workspace-api", false,
		"Read the metadata through the workspace API, the path taken on a Databricks Runtime")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		// FetchRepositoryInfo only reads the workspace API on a Databricks Runtime,
		// which cannot be detected off-cluster, so that path is selected explicitly.
		fetch := git.FetchRepositoryInfo
		if workspaceAPI {
			fetch = git.FetchRepositoryInfoWorkspace
		}

		info, err := fetch(ctx, path, w)
		if err != nil {
			return err
		}

		out := repositoryInfoOutput{
			WorktreeRoot:  info.WorktreeRoot,
			CurrentBranch: info.CurrentBranch,
			LatestCommit:  info.LatestCommit,
			OriginURL:     info.OriginURL,
		}

		switch root.OutputType(cmd) {
		case flags.OutputText:
			cmdio.LogString(ctx, "worktree_root: "+out.WorktreeRoot)
			cmdio.LogString(ctx, "current_branch: "+out.CurrentBranch)
			cmdio.LogString(ctx, "latest_commit: "+out.LatestCommit)
			cmdio.LogString(ctx, "origin_url: "+out.OriginURL)
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write(buf)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
