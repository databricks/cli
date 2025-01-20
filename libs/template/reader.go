package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

type Reader interface {
	// FS returns a file system that contains the template
	// definition files.
	FS(ctx context.Context) (fs.FS, error)

	// Cleanup releases any resources associated with the reader
	// like cleaning up temporary directories.
	Cleanup(ctx context.Context)
}

type builtinReader struct {
	name string
}

func (r *builtinReader) FS(ctx context.Context) (fs.FS, error) {
	builtin, err := builtin()
	if err != nil {
		return nil, err
	}

	for _, entry := range builtin {
		if entry.Name == r.name {
			return entry.FS, nil
		}
	}

	return nil, fmt.Errorf("builtin template %s not found", r.name)
}

func (r *builtinReader) Cleanup(ctx context.Context) {}

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
	// Calling FS twice will lead to two downloaded copies of the git repo.
	// In the future if you need to call FS twice, consider adding some caching
	// logic here to avoid multiple downloads.
	if r.tmpRepoDir != "" {
		return nil, errors.New("FS called twice on git reader")
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

	err = r.cloneFunc(ctx, r.gitUrl, r.ref, repoDir)
	close(promptSpinner)
	if err != nil {
		return nil, err
	}

	return os.DirFS(filepath.Join(repoDir, r.templateDir)), nil
}

func (r *gitReader) Cleanup(ctx context.Context) {
	if r.tmpRepoDir == "" {
		return
	}

	// Cleanup is best effort. Only log errors.
	err := os.RemoveAll(r.tmpRepoDir)
	if err != nil {
		log.Debugf(ctx, "Error cleaning up tmp directory %s for git template reader for URL %s: %s", r.tmpRepoDir, r.gitUrl, err)
	}
}

type localReader struct {
	// Path on the local filesystem that contains the template
	path string
}

func (r *localReader) FS(ctx context.Context) (fs.FS, error) {
	return os.DirFS(r.path), nil
}

func (r *localReader) Cleanup(ctx context.Context) {}
