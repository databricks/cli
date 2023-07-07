package config_tests

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitConfig(t *testing.T) {
	b := load(t, "./autoload_git")
	assert.Equal(t, "foo", b.Config.Bundle.Git.Branch)

	// We need to ensure that we allow people to fork the repo, so we test against a regex.
	// Otherwise, their OriginURL will be different to ours.
	sshUrlRegex := `git@github.com:\w+\/cli.git`
	httpsUrlRegex := `https:\/\/github\.com\/\w+\/cli(?:.git)?`
	regexList := []*regexp.Regexp{regexp.MustCompile(sshUrlRegex), regexp.MustCompile(httpsUrlRegex)}

	match := false
	for _, regex := range regexList {
		if regex.MatchString(b.Config.Bundle.Git.OriginURL) {
			match = true
			break
		}
	}
	assert.True(t, match)
}
