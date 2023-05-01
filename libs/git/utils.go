package git

import (
	"strings"

	giturls "github.com/whilp/git-urls"
)

func ToHttpsUrl(url string) (string, error) {
	originUrl, err := giturls.Parse(url)
	if err != nil {
		return "", err
	}
	if originUrl.Scheme == "https" {
		return originUrl.String(), nil
	}
	// if current repo is checked out with a SSH key
	if originUrl.Scheme != "https" {
		originUrl.Scheme = "https"
	}
	// `git@` is not required for HTTPS
	if originUrl.User != nil {
		originUrl.User = nil
	}
	// Remove `.git` suffix, if present.
	originUrl.Path = strings.TrimSuffix(originUrl.Path, ".git")
	return originUrl.String(), nil
}
