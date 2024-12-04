package git

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

type GitRepositoryInfo struct {
	// Various metadata about the repo. Each could be "" if it could not be read. No error is returned for such case.
	OriginURL     string
	LatestCommit  string
	CurrentBranch string

	// Absolute path to determined worktree root or "" if worktree root could not be determined.
	WorktreeRoot string

	// vfs.Path variant of WorktreeRoot if WorktreeRoot is set; otherwise defaults to input path.
	GuessedWorktreeRoot vfs.Path
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

// Fetch repository information either by quering .git or by fetching it from API (for dabs-in-workspace case).
//   - In case we could not find git repository, all string fields of GitRepositoryInfo will be "" and err will be nil.
//   - If there were any errors when trying to determine git root (e.g. API call returned an error or there were permission issues
//     reading the file system), all strings fields of GitRepositoryInfo will be "" and err will be non-nil.
//   - For convenience, GuessedWorktreeRoot parameter will be set to path in the above two cases.
//   - If we could determine git worktree root but there were errors when reading metadata (origin, branch, commit), those errors
//     will be logged as warnings, GitRepositoryInfo is guaranteed to have non-empty WorktreeRoot and corresponding GuessedWorktreeRoot
//     and other fields on best effort basis. The err will be nil.
//   - In successful case, all fields are set to proper git repository metadata.
func FetchRepositoryInfo(ctx context.Context, path string, w *databricks.WorkspaceClient) (GitRepositoryInfo, error) {
	if strings.HasPrefix(path, "/Workspace/") && dbr.RunsOnRuntime(ctx) {
		return fetchRepositoryInfoAPI(ctx, path, w)
	} else {
		return fetchRepositoryInfoDotGit(ctx, path)
	}
}

func fetchRepositoryInfoAPI(ctx context.Context, path string, w *databricks.WorkspaceClient) (GitRepositoryInfo, error) {
	result := GitRepositoryInfo{
		// For convenience, this field defaults to input path, even if err is also set.
		GuessedWorktreeRoot: vfs.MustNew(path),
	}

	apiClient, err := client.New(w.Config)
	if err != nil {
		return result, err
	}

	var response response
	const apiEndpoint = "/api/2.0/workspace/get-status"

	err = apiClient.Do(
		ctx,
		http.MethodGet,
		apiEndpoint,
		nil,
		map[string]string{
			"path":            path,
			"return_git_info": "true",
		},
		&response,
	)

	if err != nil {
		return result, err
	}

	// Check if GitInfo is present and extract relevant fields
	gi := response.GitInfo
	if gi != nil {
		fixedPath := ensureWorkspacePrefix(gi.Path)
		result.OriginURL = gi.URL
		result.LatestCommit = gi.HeadCommitID
		result.CurrentBranch = gi.Branch
		result.WorktreeRoot = fixedPath
		// Note, this won't work on Windows since vfs.MustNew will call filepath.Abs
		result.GuessedWorktreeRoot = vfs.MustNew(fixedPath)
	} else {
		log.Warnf(ctx, "Failed to load git info from %s", apiEndpoint)
	}

	return result, nil
}

func ensureWorkspacePrefix(p string) string {
	if !strings.HasPrefix(p, "/Workspace/") {
		return path.Join("/Workspace", p)
	}
	return p
}

func fetchRepositoryInfoDotGit(ctx context.Context, path string) (GitRepositoryInfo, error) {
	result := GitRepositoryInfo{
		GuessedWorktreeRoot: vfs.MustNew(path),
	}

	rootDir, err := findLeafInTree(path, GitDirectoryName)
	if rootDir == "" {
		return result, err
	}

	result.WorktreeRoot = rootDir
	result.GuessedWorktreeRoot = vfs.MustNew(rootDir)

	repo, err := NewRepository(result.GuessedWorktreeRoot)
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

func findLeafInTree(p string, leafName string) (string, error) {
	var err error
	for i := 0; i < 10000; i++ {
		_, err = os.Stat(filepath.Join(p, leafName))

		if err == nil {
			// Found [leafName] in p
			return p, nil
		}

		// ErrNotExist means we continue traversal up the tree.
		if errors.Is(err, fs.ErrNotExist) {
			parent := filepath.Dir(p)
			if parent == p {
				return "", nil
			}
			p = parent
			continue
		}
		break
	}

	return "", err
}
