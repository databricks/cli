package git

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/folders"
	giturls "github.com/whilp/git-urls"
)

const gitIgnoreFileName = ".gitignore"

// Repository represents a Git repository or a directory
// that could later be initialized as Git repository.
type Repository struct {
	// real indicates if this is a real repository or a non-Git
	// directory where we process .gitignore files.
	real bool

	// rootPath is the absolute path to the repository root.
	rootPath string

	// ignore contains a list of ignore patterns indexed by the
	// path prefix relative to the repository root.
	//
	// Example prefixes: ".", "foo/bar"
	//
	// Note: prefixes use the forward slash instead of the
	// OS-specific path separator. This matches Git convention.
	ignore map[string][]ignoreRules

	config *config
}

// Root returns the repository root.
func (r *Repository) Root() string {
	return r.rootPath
}

func (r *Repository) CurrentBranch() (string, error) {
	// load .git/HEAD
	head, err := LoadHead(filepath.Join(r.rootPath, ".git", "HEAD"))
	if err != nil {
		return "", err
	}
	if head == nil {
		return "", nil
	}

	// case: when a git object like commit,tag or remote branch is checked out
	if head.Type == HeadTypeSHA1 {
		return "", nil
	}
	return head.CurrentBranch()
}

func (r *Repository) LatestCommit() (string, error) {
	// load .git/HEAD
	head, err := LoadHead(filepath.Join(r.rootPath, ".git", "HEAD"))
	if err != nil {
		return "", err
	}
	if head == nil {
		// return empty string when head file does not exist
		return "", nil
	}

	// case: when a git object like commit,tag or remote branch is checked out
	if head.Type == HeadTypeSHA1 {
		return head.Content, nil
	}

	// read reference from .git/HEAD
	refPath, err := head.ReferencePath()
	if err != nil {
		return "", err
	}
	refHead, err := LoadHead(filepath.Join(r.rootPath, ".git", refPath))
	if err != nil {
		return "", err
	}
	if refHead == nil {
		// return empty string when head file does not exist
		return "", nil
	}
	if refHead.Type != HeadTypeSHA1 {
		return "", fmt.Errorf("git reference at %s was expected to be a SHA-1 commit id", refPath)
	}
	return refHead.Content, nil
}

func (r *Repository) OriginUrl() (string, error) {
	rawUrl, ok := r.config.variables["remote.origin.url"]
	if !ok {
		// return empty string if origin url is not defined
		return "", nil
	}
	originUrl, err := giturls.Parse(rawUrl)
	if err != nil {
		return "", err
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

// loadConfig loads and combines user specific and repository specific configuration files.
func (r *Repository) loadConfig() (*config, error) {
	config, err := globalGitConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load user specific gitconfig: %w", err)
	}
	err = config.loadFile(filepath.Join(r.rootPath, ".git/config"))
	if err != nil {
		return nil, fmt.Errorf("unable to load repository specific gitconfig: %w", err)
	}
	return config, nil
}

// newIgnoreFile constructs a new [ignoreRules] implementation backed by
// a file using the specified path relative to the repository root.
func (r *Repository) newIgnoreFile(relativeIgnoreFilePath string) ignoreRules {
	return newIgnoreFile(filepath.Join(r.rootPath, relativeIgnoreFilePath))
}

// getIgnoreRules returns a slice of [ignoreRules] that apply
// for the specified prefix. The prefix must be cleaned by the caller.
// It lazily initializes an entry for the specified prefix if it
// doesn't yet exist.
func (r *Repository) getIgnoreRules(prefix string) []ignoreRules {
	fs, ok := r.ignore[prefix]
	if ok {
		return fs
	}

	r.ignore[prefix] = append(r.ignore[prefix], r.newIgnoreFile(filepath.Join(prefix, gitIgnoreFileName)))
	return r.ignore[prefix]
}

// taintIgnoreRules taints all ignore rules such that the underlying files
// are checked for modification next time they are needed.
func (r *Repository) taintIgnoreRules() {
	for _, fs := range r.ignore {
		for _, f := range fs {
			f.Taint()
		}
	}
}

// Ignore computes whether to ignore the specified path.
// The specified path is relative to the repository root path.
func (r *Repository) Ignore(relPath string) (bool, error) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")

	// Retain trailing slash for directory patterns.
	// We know a trailing slash was present if the last element
	// after splitting is an empty string.
	trailingSlash := ""
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
		trailingSlash = "/"
	}

	// Walk over path prefixes to check applicable gitignore files.
	for i := range parts {
		prefix := path.Clean(strings.Join(parts[:i], "/"))
		suffix := path.Clean(strings.Join(parts[i:], "/")) + trailingSlash

		// For this prefix (e.g. ".", or "dir1/dir2") we check if the
		// suffix is matched in the respective gitignore files.
		for _, rules := range r.getIgnoreRules(prefix) {
			match, err := rules.MatchesPath(suffix)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
	}

	return false, nil
}

func NewRepository(path string) (*Repository, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	real := true
	rootPath, err := folders.FindDirWithLeaf(path, ".git")
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Cannot find `.git` directory.
		// Treat the specified path as a potential repository root.
		real = false
		rootPath = path
	}

	repo := &Repository{
		real:     real,
		rootPath: rootPath,
		ignore:   make(map[string][]ignoreRules),
	}

	config, err := repo.loadConfig()
	if err != nil {
		// Error doesn't need to be rewrapped.
		return nil, err
	}
	repo.config = config

	coreExcludesPath, err := config.coreExcludesFile()
	if err != nil {
		return nil, fmt.Errorf("unable to access core excludes file: %w", err)
	}

	// Initialize root ignore rules.
	// These are special and not lazily initialized because:
	// 1) we include a hardcoded ignore pattern
	// 2) we include a gitignore file at a non-standard path
	repo.ignore["."] = []ignoreRules{
		// Load global excludes on this machine.
		newIgnoreFile(coreExcludesPath),
		// Always ignore root .git directory.
		newStringIgnoreRules([]string{
			".git",
		}),
		// Load repository-wide excludes file.
		repo.newIgnoreFile(".git/info/excludes"),
		// Load root gitignore file.
		repo.newIgnoreFile(".gitignore"),
	}

	return repo, nil
}

func (r *Repository) addIgnoreRule(rule ignoreRules) {
	r.ignore["."] = append(r.ignore["."], rule)
}
