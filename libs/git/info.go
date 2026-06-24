package git

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
)

type RepositoryInfo struct {
	// Various metadata about the repo. Each could be "" if it could not be read. No error is returned for such case.
	OriginURL     string
	LatestCommit  string
	CurrentBranch string

	// Absolute path to determined worktree root or "" if worktree root could not be determined.
	WorktreeRoot string
}

type gitInfo struct {
	Branch       string `json:"branch"`
	HeadCommitID string `json:"head_commit_id"`
	Path         string `json:"path"`
	URL          string `json:"url"`
	// ID of the git folder object. Some workspace git folders return only id+path
	// from get-status (omitting branch/commit/url), so the id lets us recover the
	// rest via the Repos API. See the fallback in fetchRepositoryInfoAPI.
	ID int64 `json:"id"`
}

type response struct {
	GitInfo *gitInfo `json:"git_info,omitempty"`
}

// Fetch repository information either by quering .git or by fetching it from API (for dabs-in-workspace case).
//   - In case we could not find git repository (including when the path does not exist), all string fields of RepositoryInfo will be "" and err will be nil.
//   - If there were any errors when trying to determine git root (e.g. API call returned an error or there were permission issues
//     reading the file system), all strings fields of RepositoryInfo will be "" and err will be non-nil.
//   - If we could determine git worktree root but there were errors when reading metadata (origin, branch, commit), those errors
//     will be logged as warnings, RepositoryInfo is guaranteed to have non-empty WorktreeRoot and other fields on best effort basis.
//   - In successful case, all fields are set to proper git repository metadata.
func FetchRepositoryInfo(ctx context.Context, path string, w *databricks.WorkspaceClient) (RepositoryInfo, error) {
	var info RepositoryInfo
	var err error
	if strings.HasPrefix(path, "/Workspace/") && dbr.RunsOnRuntime(ctx) {
		info, err = fetchRepositoryInfoAPI(ctx, path, w)
	} else {
		info, err = fetchRepositoryInfoDotGit(ctx, path)
	}

	// A path that does not exist just means there is no repository there, which
	// is not an error. Both backends report this as fs.ErrNotExist (the API
	// backend translates a workspace 404 to it), so it is normalized to a nil
	// error in a single place rather than special-cased by every caller.
	if errors.Is(err, fs.ErrNotExist) {
		return info, nil
	}
	return info, err
}

func fetchRepositoryInfoAPI(ctx context.Context, path string, w *databricks.WorkspaceClient) (RepositoryInfo, error) {
	result := RepositoryInfo{}

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
		auth.WorkspaceIDHeaders(w.Config),
		nil,
		map[string]string{
			"path":            path,
			"return_git_info": "true",
		},
		&response,
	)
	if err != nil {
		// The workspace API returns 404 when the path is not a workspace object
		// (for example, an ephemeral directory that is not part of a Repo).
		// Normalize it to fs.ErrNotExist, the same signal fetchRepositoryInfoDotGit
		// produces for a missing local path, so FetchRepositoryInfo can treat
		// "no path" as "no repository" uniformly.
		if apierr.IsMissing(err) {
			return result, fs.ErrNotExist
		}
		return result, err
	}

	// Check if GitInfo is present and extract relevant fields
	gi := response.GitInfo
	if gi == nil {
		log.Infof(ctx, "Failed to load git info from %s", apiEndpoint)
		return result, nil
	}

	result.OriginURL = gi.URL
	result.LatestCommit = gi.HeadCommitID
	result.CurrentBranch = gi.Branch
	result.WorktreeRoot = ensureWorkspacePrefix(gi.Path)

	// Some workspace git folders return only id+path from get-status and omit the
	// origin URL. When that happens, fetch the full provenance from the Repos API
	// by id. Classic repos return the URL inline and skip this extra call.
	if gi.ID != 0 && result.OriginURL == "" {
		repo, err := w.Repos.GetByRepoId(ctx, gi.ID)
		if err != nil {
			// Best effort: WorktreeRoot is already set, so degrade to partial info
			// rather than failing the deploy (see FetchRepositoryInfo's contract).
			log.Debugf(ctx, "Failed to load git info from Repos API for id %d: %v", gi.ID, err)
			return result, nil
		}
		result.OriginURL = repo.Url
		result.LatestCommit = repo.HeadCommitId
		result.CurrentBranch = repo.Branch
	}

	return result, nil
}

func ensureWorkspacePrefix(p string) string {
	if !strings.HasPrefix(p, "/Workspace/") {
		return path.Join("/Workspace", p)
	}
	return p
}

func fetchRepositoryInfoDotGit(ctx context.Context, path string) (RepositoryInfo, error) {
	result := RepositoryInfo{}

	rootDir, err := folders.FindDirWithLeaf(path, GitDirectoryName)
	if rootDir == "" {
		return result, err
	}

	result.WorktreeRoot = rootDir

	repo, err := NewRepository(ctx, vfs.MustNew(rootDir))
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
