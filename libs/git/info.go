package git

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"path"
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
	WorktreeRoot  string
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
	if gi != nil {
		fixedPath := ensureWorkspacePrefix(gi.Path)
		return GitRepositoryInfo{
			OriginURL:     gi.URL,
			LatestCommit:  gi.HeadCommitID,
			CurrentBranch: gi.Branch,
			WorktreeRoot:  fixedPath,
		}, nil
	}

	log.Warnf(ctx, "Failed to load git info from %s", apiEndpoint)
	return GitRepositoryInfo{}, nil
}

func ensureWorkspacePrefix(p string) string {
	if !strings.HasPrefix(p, "/Workspace/") {
		return path.Join("/Workspace", p)
	}
	return p
}

func fetchRepositoryInfoDotGit(ctx context.Context, path vfs.Path) (GitRepositoryInfo, error) {
	rootDir, err := vfs.FindLeafInTree(path, GitDirectoryName)
	if err != nil || rootDir == nil {
		if errors.Is(err, fs.ErrNotExist) {
			return GitRepositoryInfo{}, nil
		}
		return GitRepositoryInfo{}, err
	}

	result := GitRepositoryInfo{
		WorktreeRoot: rootDir.Native(),
	}

	repo, err := NewRepository(rootDir)
	if err != nil {
		log.Warnf(ctx, "failed to read .git: %s", err)

		// return early since operations below won't work
		return result, nil
	}

	result.OriginURL = repo.OriginUrl()

	result.CurrentBranch, err = repo.CurrentBranch()
	if err != nil {
		log.Warnf(ctx, "failed to load current branch: %s", err)
	}

	result.LatestCommit, err = repo.LatestCommit()
	if err != nil {
		log.Warnf(ctx, "failed to load latest commit: %s", err)
	}

	return result, nil
}
