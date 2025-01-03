package template

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
)

// TODO: Add tests for all these readers.
type Reader interface {
	// FS returns a file system that contains the template
	// definition files. This function is NOT thread safe.
	FS(ctx context.Context) (fs.FS, error)

	// Close releases any resources associated with the reader
	// like cleaning up temporary directories.
	Close() error

	Name() string
}

type builtinReader struct {
	name     string
	fsCached fs.FS
}

func (r *builtinReader) FS(ctx context.Context) (fs.FS, error) {
	// If the FS has already been loaded, return it.
	if r.fsCached != nil {
		return r.fsCached, nil
	}

	builtin, err := Builtin()
	if err != nil {
		return nil, err
	}

	var templateFS fs.FS
	for _, entry := range builtin {
		if entry.Name == r.name {
			templateFS = entry.FS
			break
		}
	}

	r.fsCached = templateFS
	return r.fsCached, nil
}

func (r *builtinReader) Close() error {
	return nil
}

func (r *builtinReader) Name() string {
	return r.name
}

type gitReader struct {
	name string
	// URL of the git repository that contains the template
	gitUrl string
	// tag or branch to checkout
	ref string
	// subdirectory within the repository that contains the template
	templateDir string
	// temporary directory where the repository is cloned
	tmpRepoDir string

	fsCached fs.FS
}

// Computes the repo name from the repo URL. Treats the last non empty word
// when splitting at '/' as the repo name. For example: for url git@github.com:databricks/cli.git
// the name would be "cli.git"
func repoName(url string) string {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	return parts[len(parts)-1]
}

var gitUrlPrefixes = []string{
	"https://",
	"git@",
}

// TODO: Copy over tests for this function.
func IsGitRepoUrl(url string) bool {
	result := false
	for _, prefix := range gitUrlPrefixes {
		if strings.HasPrefix(url, prefix) {
			result = true
			break
		}
	}
	return result
}

// TODO: Can I remove the name from here and other readers?
func NewGitReader(name, gitUrl, ref, templateDir string) Reader {
	return &gitReader{
		name:        name,
		gitUrl:      gitUrl,
		ref:         ref,
		templateDir: templateDir,
	}
}

// TODO: Test the idempotency of this function as well.
func (r *gitReader) FS(ctx context.Context) (fs.FS, error) {
	// If the FS has already been loaded, return it.
	if r.fsCached != nil {
		return r.fsCached, nil
	}

	// Create a temporary directory with the name of the repository.  The '*'
	// character is replaced by a random string in the generated temporary directory.
	repoDir, err := os.MkdirTemp("", repoName(r.gitUrl)+"-*")
	if err != nil {
		return nil, err
	}
	r.tmpRepoDir = repoDir

	// start the spinner
	promptSpinner := cmdio.Spinner(ctx)
	promptSpinner <- "Downloading the template\n"

	err = git.Clone(ctx, r.gitUrl, r.ref, repoDir)
	close(promptSpinner)
	if err != nil {
		return nil, err
	}

	r.fsCached = os.DirFS(filepath.Join(repoDir, r.templateDir))
	return r.fsCached, nil
}

func (r *gitReader) Close() error {
	if r.tmpRepoDir == "" {
		return nil
	}

	return os.RemoveAll(r.tmpRepoDir)
}

func (r *gitReader) Name() string {
	return r.name
}

type localReader struct {
	name string
	// Path on the local filesystem that contains the template
	path string

	fsCached fs.FS
}

func NewLocalReader(name, path string) Reader {
	return &localReader{
		name: name,
		path: path,
	}
}

func (r *localReader) FS(ctx context.Context) (fs.FS, error) {
	// If the FS has already been loaded, return it.
	if r.fsCached != nil {
		return r.fsCached, nil
	}

	r.fsCached = os.DirFS(r.path)
	return r.fsCached, nil
}

func (r *localReader) Close() error {
	return nil
}

func (r *localReader) Name() string {
	return r.name
}

type failReader struct{}

func (r *failReader) FS(ctx context.Context) (fs.FS, error) {
	return nil, fmt.Errorf("this is a placeholder reader that always fails. Please configure a real reader.")
}

func (r *failReader) Close() error {
	return fmt.Errorf("this is a placeholder reader that always fails. Please configure a real reader.")
}

func (r *failReader) Name() string {
	return "failReader"
}
