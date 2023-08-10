package git

import (
	"strings"

	giturls "github.com/whilp/git-urls"
)

// Return an origin URL as an HTTPS URL.
// The transformations in this function are not guaranteed to work for all
// Git providers. They are only guaranteed to work for GitHub.
func ToHttpsUrl(url string) (string, error) {
	origin, err := giturls.Parse(url)
	if err != nil {
		return "", err
	}
	// If this repository is checked out over SSH
	if origin.Scheme != "https" {
		origin.Scheme = "https"
	}
	// Basic auth is not applicable for an HTTPS URL.
	if origin.User != nil {
		origin.User = nil
	}
	// Remove `.git` suffix, if present.
	origin.Path = strings.TrimSuffix(origin.Path, ".git")
	return origin.String(), nil
}
