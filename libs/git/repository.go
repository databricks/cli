package git

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/vfs"
)

const gitIgnoreFileName = ".gitignore"

var GitDirectoryName = ".git"

// Repository represents a Git repository or a directory
// that could later be initialized as Git repository.
type Repository struct {
	// real indicates if this is a real repository or a non-Git
	// directory where we process .gitignore files.
	real bool

	// root is the absolute path to the repository root.
	root vfs.Path

	// ignore contains a list of ignore patterns indexed by the
	// path prefix relative to the repository root.
	//
	// Example prefixes: ".", "foo/bar"
	//
	// Note: prefixes use the forward slash instead of the
	// OS-specific path separator. This matches Git convention.
	ignore map[string][]ignoreRules

	// config contains a merged view of the user specific and the repository
	// specific git configuration loaded from .git/config files.
	//
	// Also see: https://git-scm.com/docs/git-config.
	config *config
}

// Root returns the absolute path to the repository root.
func (r *Repository) Root() string {
	return r.root.Native()
}

func (r *Repository) CurrentBranch() (string, error) {
	// load .git/HEAD
	ref, err := LoadReferenceFile(r.root, path.Join(GitDirectoryName, "HEAD"))
	if err != nil {
		return "", err
	}
	if ref == nil {
		return "", nil
	}

	// case: when a git object like commit,tag or remote branch is checked out
	if ref.Type == ReferenceTypeSHA1 {
		return "", nil
	}
	return ref.CurrentBranch()
}

func (r *Repository) LatestCommit() (string, error) {
	// load .git/HEAD
	ref, err := LoadReferenceFile(r.root, path.Join(GitDirectoryName, "HEAD"))
	if err != nil {
		return "", err
	}
	if ref == nil {
		// return empty string when head file does not exist
		return "", nil
	}

	// case: when a git object like commit,tag or remote branch is checked out
	if ref.Type == ReferenceTypeSHA1 {
		return ref.Content, nil
	}

	// read reference from .git/HEAD
	branchHeadPath, err := ref.ResolvePath()
	if err != nil {
		return "", err
	}
	branchHeadRef, err := LoadReferenceFile(r.root, path.Join(GitDirectoryName, branchHeadPath))
	if err != nil {
		return "", err
	}
	if branchHeadRef == nil {
		// return empty string when head file does not exist
		return "", nil
	}
	if branchHeadRef.Type != ReferenceTypeSHA1 {
		return "", fmt.Errorf("git reference at %s was expected to be a SHA-1 commit id", branchHeadPath)
	}
	return branchHeadRef.Content, nil
}

// return origin url if it's defined, otherwise an empty string
func (r *Repository) OriginUrl() string {
	rawUrl := r.config.variables["remote.origin.url"]

	// Remove username and password from the URL.
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		// Git supports https URLs and non standard URLs like "ssh://" or "file://".
		// Parsing these URLs is not supported by the Go standard library. In case
		// of an error, we return the raw URL. This is okay because for ssh URLs
		// because passwords cannot be included in the URL.
		return rawUrl
	}
	// Setting User to nil removes the username and password from the URL when
	// .String() is called.
	// See: https://pkg.go.dev/net/url#URL.String
	parsedUrl.User = nil
	return parsedUrl.String()
}

// loadConfig loads and combines user specific and repository specific configuration files.
func (r *Repository) loadConfig() error {
	config, err := globalGitConfig()
	if err != nil {
		return fmt.Errorf("unable to load user specific gitconfig: %w", err)
	}
	err = config.loadFile(r.root, ".git/config")
	if err != nil {
		return fmt.Errorf("unable to load repository specific gitconfig: %w", err)
	}
	r.config = config
	return nil
}

// newIgnoreFile constructs a new [ignoreRules] implementation backed by
// a file using the specified path relative to the repository root.
func (r *Repository) newIgnoreFile(relativeIgnoreFilePath string) ignoreRules {
	return newIgnoreFile(r.root, relativeIgnoreFilePath)
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

	r.ignore[prefix] = append(r.ignore[prefix], r.newIgnoreFile(path.Join(prefix, gitIgnoreFileName)))
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
	parts := strings.Split(relPath, "/")

	// Retain trailing slash for directory patterns.
	// We know a trailing slash was present if the last element
	// after splitting is an empty string.
	trailingSlash := ""
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
		trailingSlash = "/"
	}

	// Never ignore the root path (an unnamed path)
	if len(parts) == 1 && parts[0] == "." {
		return false, nil
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

func NewRepository(path vfs.Path) (*Repository, error) {
	real := true
	rootPath, err := vfs.FindLeafInTree(path, GitDirectoryName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		// Cannot find `.git` directory.
		// Treat the specified path as a potential repository root.
		real = false
		rootPath = path
	}

	repo := &Repository{
		real:   real,
		root:   rootPath,
		ignore: make(map[string][]ignoreRules),
	}

	err = repo.loadConfig()
	if err != nil {
		// Error doesn't need to be rewrapped.
		return nil, err
	}

	coreExcludesPath, err := repo.config.coreExcludesFile()
	if err != nil {
		return nil, fmt.Errorf("unable to access core excludes file: %w", err)
	}

	// Load global excludes on this machine.
	// This is by definition a local path so we create a new [vfs.Path] instance.
	coreExcludes := newStringIgnoreRules([]string{})
	if coreExcludesPath != "" {
		dir := filepath.Dir(coreExcludesPath)
		base := filepath.Base(coreExcludesPath)
		coreExcludes = newIgnoreFile(vfs.MustNew(dir), base)
	}

	// Initialize root ignore rules.
	// These are special and not lazily initialized because:
	// 1) we include a hardcoded ignore pattern
	// 2) we include a gitignore file at a non-standard path
	repo.ignore["."] = []ignoreRules{
		coreExcludes,
		// Always ignore root .git directory.
		newStringIgnoreRules([]string{
			".git",
		}),
		// Load repository-wide exclude file.
		repo.newIgnoreFile(".git/info/exclude"),
		// Load root gitignore file.
		repo.newIgnoreFile(".gitignore"),
	}

	return repo, nil
}

func (r *Repository) addIgnoreRule(rule ignoreRules) {
	r.ignore["."] = append(r.ignore["."], rule)
}
