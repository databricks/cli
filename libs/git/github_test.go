package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantRepoURL string
		wantSubdir  string
		wantBranch  string
	}{
		{
			name:        "simple repo URL",
			url:         "https://github.com/user/repo",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "",
		},
		{
			name:        "repo URL with trailing slash",
			url:         "https://github.com/user/repo/",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "",
		},
		{
			name:        "repo with branch",
			url:         "https://github.com/user/repo/tree/main",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "main",
		},
		{
			name:        "repo with branch and subdir",
			url:         "https://github.com/user/repo/tree/main/templates/starter",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "templates/starter",
			wantBranch:  "main",
		},
		{
			name:        "repo with branch and deep subdir",
			url:         "https://github.com/databricks/cli/tree/v0.1.0/libs/template/templates/default-python",
			wantRepoURL: "https://github.com/databricks/cli",
			wantSubdir:  "libs/template/templates/default-python",
			wantBranch:  "v0.1.0",
		},
		{
			name:        "repo with feature branch",
			url:         "https://github.com/user/repo/tree/feature/my-feature",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "my-feature",
			wantBranch:  "feature",
		},
		{
			name:        "repo URL with trailing slash and tree",
			url:         "https://github.com/user/repo/tree/main/",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepoURL, gotSubdir, gotBranch := ParseGitHubURL(tt.url)
			assert.Equal(t, tt.wantRepoURL, gotRepoURL, "repoURL mismatch")
			assert.Equal(t, tt.wantSubdir, gotSubdir, "subdir mismatch")
			assert.Equal(t, tt.wantBranch, gotBranch, "branch mismatch")
		})
	}
}
