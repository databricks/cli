package git

import (
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRepositoryInfoFromWorkspace(t *testing.T) {
	dir := setupWorktree(t)
	worktree := filepath.Join(dir, "my_worktree")
	writeGitDir(t, dir, "gitdir: ../.git/worktrees/my_worktree")
	writeGitCommonDir(t, dir, "../..")
	want := RepositoryInfo{
		OriginURL:     "https://example.test/repo.git",
		LatestCommit:  strings.Repeat("1", 40),
		CurrentBranch: "projects/task",
		WorktreeRoot:  worktree,
	}
	testutil.WriteFile(t, filepath.Join(dir, ".git", "refs", "heads", want.CurrentBranch), want.LatestCommit)
	testutil.WriteFile(t, filepath.Join(dir, ".git", "worktrees", "my_worktree", "HEAD"), "ref: refs/heads/"+want.CurrentBranch)
	testutil.WriteFile(t, filepath.Join(dir, ".git", "config"), "[remote \"origin\"]\n\turl = "+want.OriginURL)
	file := testutil.Touch(t, worktree, "subdir", "README.md")
	malformed := t.TempDir()
	testutil.WriteFile(t, filepath.Join(malformed, ".git"), "invalid gitfile")

	for _, tc := range []struct {
		name    string
		path    string
		body    string
		status  int
		want    RepositoryInfo
		wantErr error
	}{
		{name: "worktree", path: worktree, want: want},
		{name: "subdirectory", path: filepath.Dir(file), want: want},
		{name: "file", path: file, want: want},
		{
			name: "workspace metadata takes precedence", path: worktree,
			body: `{"git_info":{"branch":"api-main","head_commit_id":"2222222222222222222222222222222222222222","path":"/Users/me/api","url":"https://example.test/api.git"}}`,
			want: RepositoryInfo{
				OriginURL: "https://example.test/api.git", LatestCommit: strings.Repeat("2", 40),
				CurrentBranch: "api-main", WorktreeRoot: "/Workspace/Users/me/api",
			},
		},
		{name: "git directory", path: dir},
		{name: "no git metadata", path: t.TempDir()},
		{name: "missing path", path: filepath.Join(worktree, "missing")},
		{name: "malformed gitfile", path: malformed, want: RepositoryInfo{WorktreeRoot: malformed}},
		{
			name: "permission denied", path: worktree, status: http.StatusForbidden,
			body: `{"error_code":"PERMISSION_DENIED","message":"denied"}`, wantErr: apierr.ErrPermissionDenied,
		},
		{
			name: "workspace path missing", path: worktree, status: http.StatusNotFound,
			body: `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"missing"}`, wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			if body == "" {
				body = `{}`
			}
			cfg, server := qa.HTTPFixtures{{
				Method:   "GET",
				Resource: "/api/2.0/workspace/get-status?path=" + url.QueryEscape(tc.path) + "&return_git_info=true",
				Response: body,
				Status:   tc.status,
			}}.Config(t)
			t.Cleanup(server.Close)
			w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

			got, err := fetchRepositoryInfoFromWorkspace(t.Context(), tc.path, w)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}
