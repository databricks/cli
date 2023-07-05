package repos

import (
	"net/url"
	"regexp"
	"strings"
)

var gitProviders = map[string]string{
	"github.com":    "gitHub",
	"dev.azure.com": "azureDevOpsServices",
	"gitlab.com":    "gitLab",
	"bitbucket.org": "bitbucketCloud",
}

var awsCodeCommitRegexp = regexp.MustCompile(`^git-codecommit\.[^.]+\.amazonaws.com$`)

func DetectProvider(rawURL string) string {
	provider := ""
	u, err := url.Parse(rawURL)
	if err != nil {
		return provider
	}
	if v, ok := gitProviders[strings.ToLower(u.Host)]; ok {
		provider = v
	} else if awsCodeCommitRegexp.MatchString(u.Host) {
		provider = "awsCodeCommit"
	}
	return provider
}
