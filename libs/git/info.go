package git

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
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

// Prefix of the workspace file system mount, as seen from a DBR session.
const workspaceMountPrefix = "/Workspace/"

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
	if strings.HasPrefix(path, workspaceMountPrefix) && dbr.RunsOnRuntime(ctx) && !hasDotGit(ctx, path) {
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
	if gi != nil {
		fixedPath := ensureWorkspacePrefix(gi.Path)
		result.OriginURL = gi.URL
		result.LatestCommit = gi.HeadCommitID
		result.CurrentBranch = gi.Branch
		result.WorktreeRoot = fixedPath
	} else {
		log.Infof(ctx, "Failed to load git info from %s", apiEndpoint)
	}

	return result, nil
}

// hasDotGit reports whether a .git belonging to path's own Git folder is readable
// on the workspace mount.
//
// The new type of in-workspace Git folder (Git in Dataplane) exposes a real .git on
// the /Workspace mount, while get-status returns only id+path for it, omitting the
// origin URL, branch and commit. Reading .git recovers all three. Classic Repos have
// no .git on the mount and still go through the API, which returns their git info
// inline.
//
// The search stops below the owner root (/Workspace/Users/<user>, /Workspace/Shared,
// ...) rather than walking to the filesystem root: a Git folder always carries its
// .git at its own root, which lives inside the owner root, so anything found at or
// above it belongs to a different repository. Without the bound, an unrelated .git in
// the user's workspace home would be reported as this bundle's provenance and could
// fail the deploy in ValidateGitDetails.
func hasDotGit(ctx context.Context, path string) bool {
	root, err := findDotGitBelow(path, ownerRoot(path))
	if err != nil {
		// Anything other than "not found" (a permission or mount error) leaves the
		// state on disk unknown, so fall back to the API rather than guess.
		log.Debugf(ctx, "failed to look for %s under %s: %s", GitDirectoryName, path, err)
		return false
	}
	return root != ""
}

// findDotGitBelow returns the directory holding .git at or below dir, stopping before
// ceiling, or "" if there is none. A non-nil error means the lookup itself failed.
func findDotGitBelow(dir, ceiling string) (string, error) {
	for dir != ceiling {
		_, err := os.Stat(path.Join(dir, GitDirectoryName))
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}

		next := path.Dir(dir)
		if dir == next {
			return "", nil
		}
		dir = next
	}
	return "", nil
}

// ownerRoot returns the workspace directory that owns p's Git folders, e.g.
// /Workspace/Users/me@databricks.com or /Workspace/Shared. Git folders are created
// inside it, never at it.
func ownerRoot(p string) string {
	rest, ok := strings.CutPrefix(p, workspaceMountPrefix)
	if !ok {
		return strings.TrimSuffix(workspaceMountPrefix, "/")
	}

	// Users and Repos are keyed by principal, so the owner root includes it.
	depth := 1
	scope, _, _ := strings.Cut(rest, "/")
	if scope == "Users" || scope == "Repos" {
		depth = 2
	}

	segments := strings.Split(rest, "/")
	if len(segments) > depth {
		segments = segments[:depth]
	}
	return workspaceMountPrefix + strings.Join(segments, "/")
}

func ensureWorkspacePrefix(p string) string {
	if !strings.HasPrefix(p, workspaceMountPrefix) {
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
