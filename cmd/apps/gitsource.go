package apps

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// gitScaffoldSource holds the git-backed deployment fields rendered into a
// scaffolded databricks.yml when `apps init` runs inside a Git repository.
// A zero value means "no Git source" and the template falls back to a plain
// workspace source_code_path.
type gitScaffoldSource struct {
	// URL is the https clone URL of the origin remote, with credentials and any
	// trailing ".git" stripped (e.g. https://github.com/my-org/my-repo).
	URL string
	// Provider is the Databricks git provider enum matching URL's host
	// (e.g. gitHub). Empty when the host maps to no known provider.
	Provider string
	// Branch is the repository's current branch.
	Branch string
	// SourceCodePath is the app's location relative to the repo root, forward-slashed.
	// "./" when the app is at the repo root.
	SourceCodePath string
}

// active reports whether a git-backed source was detected. url, provider, and
// branch are required together: git_repository needs url+provider, and a deploy
// needs a ref. When any is missing the scaffold keeps its plain source_code_path.
func (g gitScaffoldSource) active() bool {
	return g.URL != "" && g.Provider != "" && g.Branch != ""
}

// detectGitScaffoldSource derives git-backed deployment fields from the Git
// repository that will contain the scaffolded app at destDir. It returns a zero
// value (and never an error) when there is nothing safe to point a git-backed
// deploy at: destDir is not inside a Git repo, the repo has no origin remote,
// the origin host maps to no known provider, the branch is unknown (detached
// HEAD), or the app would live outside the detected repo.
func detectGitScaffoldSource(ctx context.Context, destDir string, w *databricks.WorkspaceClient) gitScaffoldSource {
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		log.Debugf(ctx, "git-source detection: resolving %q: %v", destDir, err)
		return gitScaffoldSource{}
	}

	// A brand-new subdirectory does not exist yet at detection time, so probe
	// from its parent (which does) to find the enclosing repo. In-place scaffolds
	// ("." into an existing repo) probe the directory itself.
	probe := absDest
	if _, statErr := os.Stat(probe); statErr != nil {
		probe = filepath.Dir(absDest)
	}

	info, err := git.FetchRepositoryInfo(ctx, probe, w)
	if err != nil {
		log.Debugf(ctx, "git-source detection: %v", err)
		return gitScaffoldSource{}
	}
	if info.WorktreeRoot == "" || info.OriginURL == "" || info.CurrentBranch == "" {
		return gitScaffoldSource{}
	}

	repoURL, provider := normalizeGitOrigin(info.OriginURL)
	if repoURL == "" || provider == "" {
		return gitScaffoldSource{}
	}

	sourcePath := "./"
	if rel, relErr := filepath.Rel(info.WorktreeRoot, absDest); relErr == nil {
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") {
			// The app would live outside the detected repo; a git-backed deploy
			// from this origin could not find it.
			return gitScaffoldSource{}
		}
		if rel != "." {
			sourcePath = rel
		}
	}

	return gitScaffoldSource{
		URL:            repoURL,
		Provider:       provider,
		Branch:         info.CurrentBranch,
		SourceCodePath: sourcePath,
	}
}

// isInGitRepo reports whether dir is inside a Git worktree. It is used to gate
// the interactive Git onboarding prompt, which only runs when the user has no
// Git repository set up at all.
func isInGitRepo(ctx context.Context, dir string, w *databricks.WorkspaceClient) bool {
	info, err := git.FetchRepositoryInfo(ctx, dir, w)
	return err == nil && info.WorktreeRoot != ""
}

// normalizeGitOrigin converts a raw git remote URL into an https clone URL and
// the matching Databricks git provider. It handles https/ssh URLs and scp-style
// remotes (git@host:owner/repo.git). It returns empty strings when the host maps
// to no known provider, since git_repository requires url and provider together.
func normalizeGitOrigin(raw string) (repoURL, provider string) {
	host, path := splitGitRemote(strings.TrimSpace(raw))
	if host == "" {
		return "", ""
	}
	provider = providerForHost(host)
	if provider == "" {
		return "", ""
	}
	path = strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	if path == "" {
		return "", ""
	}
	return "https://" + host + "/" + path, provider
}

// splitGitRemote extracts the host and repository path from a git remote URL,
// supporting both scheme URLs (https://, ssh://) and scp-style (git@host:path).
func splitGitRemote(raw string) (host, path string) {
	if raw == "" {
		return "", ""
	}
	if !strings.Contains(raw, "://") {
		// scp-style: [user@]host:owner/repo(.git)
		if at := strings.LastIndex(raw, "@"); at != -1 {
			raw = raw[at+1:]
		}
		hostPart, pathPart, ok := strings.Cut(raw, ":")
		if !ok {
			return "", ""
		}
		return hostPart, pathPart
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	return u.Hostname(), strings.TrimPrefix(u.Path, "/")
}

// providerForHost maps a git host to its Databricks git provider enum value.
// Unknown/self-hosted hosts return "" — the provider cannot be inferred safely,
// so the scaffold falls back to a plain source_code_path.
func providerForHost(host string) string {
	switch strings.ToLower(host) {
	case "github.com":
		return "gitHub"
	case "gitlab.com":
		return "gitLab"
	case "bitbucket.org":
		return "bitbucketCloud"
	case "dev.azure.com":
		return "azureDevOpsServices"
	default:
		if strings.HasSuffix(strings.ToLower(host), ".visualstudio.com") {
			return "azureDevOpsServices"
		}
		return ""
	}
}
