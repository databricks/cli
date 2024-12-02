package git

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

type GitRepositoryInfo struct {
	OriginURL     string
	LatestCommit  string
	CurrentBranch string
	WorktreeRoot  vfs.Path
}

type gitInfo struct {
	Branch       string `json:"branch"`
	HeadCommitID string `json:"head_commit_id"`
	Path         string `json:"path"`
	URL          string `json:"url"`
}

type response struct {
	GitInfo *gitInfo `json:"git_info,omitempty"`
}

func FetchRepositoryInfo(ctx context.Context, path vfs.Path, w *databricks.WorkspaceClient) (GitRepositoryInfo, error) {
	if strings.HasPrefix(path.Native(), "/Workspace/") && dbr.RunsOnRuntime(ctx) {
		return fetchRepositoryInfoAPI(ctx, path, w)
	} else {
		return fetchRepositoryInfoDotGit(ctx, path)
	}
}

func fetchRepositoryInfoAPI(ctx context.Context, path vfs.Path, w *databricks.WorkspaceClient) (GitRepositoryInfo, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return GitRepositoryInfo{}, err
	}

	var response response
	const apiEndpoint = "/api/2.0/workspace/get-status"

	err = apiClient.Do(
		ctx,
		http.MethodGet,
		apiEndpoint,
		nil,
		map[string]string{
			"path":            path.Native(),
			"return_git_info": "true",
		},
		&response,
	)

	if err != nil {
		return GitRepositoryInfo{}, err
	}

	// Check if GitInfo is present and extract relevant fields
	gi := response.GitInfo
	if gi == nil {
		log.Warnf(ctx, "Failed to load git info from %s", apiEndpoint)
	} else {
		fixedPath := ensureWorkspacePrefix(gi.Path)
		return GitRepositoryInfo{
			OriginURL:     gi.URL,
			LatestCommit:  gi.HeadCommitID,
			CurrentBranch: gi.Branch,
			WorktreeRoot:  vfs.MustNew(fixedPath),
		}, nil
	}

	return GitRepositoryInfo{
		WorktreeRoot: path,
	}, nil
}

func ensureWorkspacePrefix(path string) string {
	if !strings.HasPrefix(path, "/Workspace/") {
		return "/Workspace/" + path
	}
	return path
}

func fetchRepositoryInfoDotGit(ctx context.Context, path vfs.Path) (GitRepositoryInfo, error) {
	rootDir, err := vfs.FindLeafInTree(path, GitDirectoryName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return GitRepositoryInfo{}, err
		}
		rootDir = path
	}

	repo, err := NewRepository(rootDir)
	if err != nil {
		return GitRepositoryInfo{}, err
	}

	branch, err := repo.CurrentBranch()
	if err != nil {
		return GitRepositoryInfo{}, nil
	}

	commit, err := repo.LatestCommit()
	if err != nil {
		return GitRepositoryInfo{}, nil
	}

	return GitRepositoryInfo{
		OriginURL:     repo.OriginUrl(),
		LatestCommit:  commit,
		CurrentBranch: branch,
		WorktreeRoot:  rootDir,
	}, nil
}
