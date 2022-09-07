package git

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/databricks/bricks/folders"
	giturls "github.com/whilp/git-urls"
	"gopkg.in/ini.v1"
)

func Root() (string, error) {
	return folders.FindDirWithLeaf(".git")
}

// Origin finds the git repository the project is cloned from, so that
// we could automatically verify if this project is checked out in repos
// home folder of the user according to recommended best practices. Can
// also be used to determine a good enough default project name.
func Origin() (*url.URL, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}
	file := fmt.Sprintf("%s/.git/config", root)
	gitConfig, err := ini.Load(file)
	if err != nil {
		return nil, err
	}
	section := gitConfig.Section(`remote "origin"`)
	if section == nil {
		return nil, fmt.Errorf("remote `origin` is not defined in %s", file)
	}
	url := section.Key("url")
	if url == nil {
		return nil, fmt.Errorf("git origin url is not defined")
	}
	return giturls.Parse(url.Value())
}

// HttpsOrigin returns URL in the format expected by Databricks Repos
// platform functionality. Gradually expand implementation to work with
// other formats of git URLs.
func HttpsOrigin() (string, error) {
	origin, err := Origin()
	if err != nil {
		return "", err
	}
	// if current repo is checked out with a SSH key
	if origin.Scheme != "https" {
		origin.Scheme = "https"
	}
	// `git@` is not required for HTTPS, as Databricks Repos are checked
	// out using an API token instead of username. But does it hold true
	// for all of the git implementations?
	if origin.User != nil {
		origin.User = nil
	}
	return origin.String(), nil
}

// RepositoryName returns repository name as last path entry from detected
// git repository up the tree or returns error if it fails to do so.
func RepositoryName() (string, error) {
	origin, err := Origin()
	if err != nil {
		return "", err
	}
	base := path.Base(origin.Path)
	return strings.TrimSuffix(base, ".git"), nil
}
