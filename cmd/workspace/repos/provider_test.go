package repos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectProvider(t *testing.T) {
	for url, provider := range map[string]string{
		"https://user@bitbucket.org/user/repo.git":                           "bitbucketCloud",
		"https://github.com//user/repo.git":                                  "gitHub",
		"https://user@dev.azure.com/user/project/_git/repo":                  "azureDevOpsServices",
		"https://abc/user/repo.git":                                          "",
		"ewfgwergfwe":                                                        "",
		"https://foo@@bar":                                                   "",
		"https://git-codecommit.us-east-2.amazonaws.com/v1/repos/MyDemoRepo": "awsCodeCommit",
	} {
		assert.Equal(t, provider, DetectProvider(url))
	}
}
