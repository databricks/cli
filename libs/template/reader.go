package template

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

type Reader interface {
	// FS returns a file system that contains the template
	// definition files. This function is NOT thread safe.
	FS(ctx context.Context) (fs.FS, error)

	// Cleanup releases any resources associated with the reader
	// like cleaning up temporary directories.
	Cleanup()
}

type builtinReader struct {
	name string
}

func (r *builtinReader) FS(ctx context.Context) (fs.FS, error) {
	builtin, err := builtin()
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

	if templateFS == nil {
		return nil, fmt.Errorf("builtin template %s not found", r.name)
	}

	return templateFS, nil
}

func (r *builtinReader) Cleanup() {}

type gitReader struct {
	gitUrl string
	// tag or branch to checkout
	ref string
	// subdirectory within the repository that contains the template
	templateDir string
	// temporary directory where the repository is cloned
	tmpRepoDir string

	// Function to clone the repository. This is a function pointer to allow
	// mocking in tests.
	cloneFunc func(ctx context.Context, url, reference, targetPath string) error
}

// Computes the repo name from the repo URL. Treats the last non empty word
// when splitting at '/' as the repo name. For example: for url git@github.com:databricks/cli.git
// the name would be "cli.git"
func repoName(url string) string {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	return parts[len(parts)-1]
}

func (r *gitReader) FS(ctx context.Context) (fs.FS, error) {
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

	err = r.cloneFunc(ctx, r.gitUrl, r.ref, repoDir)
	close(promptSpinner)
	if err != nil {
		return nil, err
	}

	return os.DirFS(filepath.Join(repoDir, r.templateDir)), nil
}

func (r *gitReader) Cleanup() {
	if r.tmpRepoDir == "" {
		return
	}

	// Cleanup is best effort. Ignore errors.
	os.RemoveAll(r.tmpRepoDir)
}

type localReader struct {
	// Path on the local filesystem that contains the template
	path string
}

func (r *localReader) FS(ctx context.Context) (fs.FS, error) {
	return os.DirFS(r.path), nil
}

func (r *localReader) Cleanup() {}
