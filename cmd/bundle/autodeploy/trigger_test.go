package autodeploy

import (
	"testing"

	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/stretchr/testify/assert"
)

func TestInferProvider(t *testing.T) {
	tests := []struct {
		url      string
		provider gitwebhooks.GitProvider
		ok       bool
	}{
		{"https://GitHub.com/acme/repo.git", gitwebhooks.GitProviderGitHub, true},
		{"https://dev.azure.com/acme/project/_git/repo", gitwebhooks.GitProviderAzureDevOpsServices, true},
		// A GitHub Enterprise or self-managed GitLab host could be either.
		{"https://git.example.test/acme/repo", "", false},
	}
	for _, tc := range tests {
		provider, ok := inferProvider(tc.url)
		assert.Equal(t, tc.provider, provider, tc.url)
		assert.Equal(t, tc.ok, ok, tc.url)
	}
}

func TestQuote(t *testing.T) {
	assert.Equal(t, `"a\"b\\c"`, quote(`a"b\c`))
}
