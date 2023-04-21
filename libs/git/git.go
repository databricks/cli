package git

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/databricks/bricks/folders"
	giturls "github.com/whilp/git-urls"
	"gopkg.in/ini.v1"
)

type configLoader struct {
	gitDirPath string
}

var ErrNotARepository = fmt.Errorf("current working directory is not a git repository")

var (
	ErrRemoteOriginNotDefined    = fmt.Errorf("remote `origin` is not defined in .git/config")
	ErrRemoteOriginUrlNotDefined = fmt.Errorf("git origin url is not defined in .git/config")
)

func NewConfigLoader() (*configLoader, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := folders.FindDirWithLeaf(wd, ".git")
	if err != nil {
		return nil, err
	}
	gitDirPath := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		return nil, ErrNotARepository
	}
	return &configLoader{
		gitDirPath: gitDirPath,
	}, nil
}

// A commit hash is a SHA-1 hash, that is a hex string of exactly 40 characters
func isCommitHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	re := regexp.MustCompile("^[0-9a-f]+$")
	return re.MatchString(s)
}

// We expect the contents of .git/HEAD file to either be
// 1. A commit hash, if the current checkout is a specific commit
// 2. Reference to a file containing the commit hash. eg: "ref: refs/heads/my-branch-name"
func (l *configLoader) Head() (string, error) {
	headfile := filepath.Join(l.gitDirPath, "HEAD")
	b, err := os.ReadFile(headfile)
	if err != nil {
		return "", err
	}
	content := string(b)
	return strings.TrimSuffix(content, "\n"), nil
}

// TODO: probablt need to trim newlines here and in the commit function
// returns the name of the current selected branch in the repo, by reading the
// .git/HEAD file
func (l *configLoader) Branch() (string, error) {
	head, err := l.Head()
	if err != nil {
		return "", err
	}
	if isCommitHash(head) {
		return "", nil
	}

	// The expected format of .git/HEAD is "ref: refs/heads/my-branch-name"
	prefix := "ref: refs/heads/"
	if !strings.HasPrefix(head, prefix) {
		return "", fmt.Errorf("unexpected content in .git/HEAD: %s", head)
	}
	return strings.TrimPrefix(head, prefix), nil
}

// reads .git/refs/heads/my-branch-name file to get the latest commit hash for branch
func (l *configLoader) branchCommit() (string, error) {
	branch, err := l.Branch()
	if err != nil {
		return "", err
	}
	path := filepath.Join(l.gitDirPath, "refs", "heads", branch)
	b, err := os.ReadFile(path)
	// .git/refs/heads/my-branch-name file does not exist for branches without
	// any commits in them
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	content := string(b)
	commitHash := strings.TrimSuffix(content, "\n")
	if !isCommitHash(commitHash) {
		return "", fmt.Errorf("content of file %s is not a commit hash", path)
	}
	return commitHash, nil
}

// return the current commit hash
func (l *configLoader) Commit() (string, error) {
	head, err := l.Head()
	if err != nil {
		return "", err
	}
	// return early if .git/HEAD already contains a commit hash. This is the case
	// when you directly checkout a commit
	if isCommitHash(head) {
		return head, nil
	}
	// we only do a single level of resolutions for refs to get the commit hash
	return l.branchCommit()
}

// True if the error represents that a remote url for "origin" is not defined
// in .git/config
func IsErrOriginUrlNotDefined(err error) bool {
	return (err == ErrRemoteOriginNotDefined || err == ErrRemoteOriginUrlNotDefined)
}

// Origin finds the git repository the project is cloned from, so that
// we could automatically verify if this project is checked out in repos
// home folder of the user according to recommended best practices. Can
// also be used to determine a good enough default project name.
func (l *configLoader) Origin() (*url.URL, error) {
	file := fmt.Sprintf("%s/config", l.gitDirPath)
	gitConfig, err := ini.Load(file)
	if err != nil {
		return nil, err
	}
	section := gitConfig.Section(`remote "origin"`)
	if section == nil {
		return nil, ErrRemoteOriginNotDefined
	}
	url := section.Key("url")
	if url == nil || url.Value() == "" {
		return nil, ErrRemoteOriginUrlNotDefined
	}
	return giturls.Parse(url.Value())
}

// HttpsOrigin returns URL in the format expected by Databricks Repos
// platform functionality. Gradually expand implementation to work with
// other formats of git URLs.
func (l *configLoader) HttpsOrigin() (string, error) {
	origin, err := l.Origin()
	if IsErrOriginUrlNotDefined(err) {
		return "", nil
	}
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
	// Remove `.git` suffix, if present.
	origin.Path = strings.TrimSuffix(origin.Path, ".git")
	return origin.String(), nil
}

// RepositoryName returns repository name as last path entry from detected
// git repository up the tree or returns error if it fails to do so.
func (l *configLoader) RepositoryName() (string, error) {
	origin, err := l.Origin()
	if err != nil {
		return "", err
	}
	base := path.Base(origin.Path)
	return strings.TrimSuffix(base, ".git"), nil
}
