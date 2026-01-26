package git

import "strings"

// ParseGitHubURL extracts the repository URL, subdirectory, and branch from a GitHub URL.
// Input: https://github.com/user/repo/tree/main/templates/starter
// Output: repoURL="https://github.com/user/repo", subdir="templates/starter", branch="main"
func ParseGitHubURL(url string) (repoURL, subdir, branch string) {
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Check for /tree/branch/path pattern
	if idx := strings.Index(url, "/tree/"); idx != -1 {
		repoURL = url[:idx]
		rest := url[idx+6:] // Skip "/tree/"

		// Split into branch and path
		parts := strings.SplitN(rest, "/", 2)
		branch = parts[0]
		if len(parts) > 1 {
			subdir = parts[1]
		}
		return repoURL, subdir, branch
	}

	// No /tree/ pattern, just a repo URL
	return url, "", ""
}
