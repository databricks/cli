package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/zip"
)

var errNotFound = errors.New("not found")

type RepositoryNotFoundError struct {
	url string
}

func (err RepositoryNotFoundError) Error() string {
	return fmt.Sprintf("repository not found: %s", err.url)
}

func (err RepositoryNotFoundError) Is(other error) bool {
	return other == errNotFound
}

type CloneOptions struct {
	// Name of the organization or profile with the repository
	Organization   string
	RepositoryName string

	// Git service provider. Eg: github, gitlab
	Provider string

	// Branch or tag name to clone
	Reference string

	// Path to clone into. The repository is cloned as ${RepositoryName}-${Reference}
	// in this target directory.
	TargetDir string
}

func (opts CloneOptions) repoUrl() string {
	return fmt.Sprintf(`https://github.com/%s/%s`, opts.Organization, opts.RepositoryName)
}

func (opts CloneOptions) zipUrl() string {
	return fmt.Sprintf(`%s/archive/%s.zip`, opts.repoUrl(), opts.Reference)
}

// TODO: test this holds true for alternate branches and tags
func (opts CloneOptions) destination() string {
	return filepath.Join(opts.TargetDir, opts.RepositoryName+"-"+opts.Reference)
}

func download(ctx context.Context, url string, dest string) error {
	// Get request to download the ZIP archive
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return RepositoryNotFoundError{url}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download ZIP archive: %s. %s", url, resp.Status)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, resp.Body)
	return err
}

// TODO: check stdin / stdout works properly with git clone and requesting an ID/password
func clonePrivate(ctx context.Context, opts CloneOptions) error {
	// TODO: test that the branch --branch flag works with tags
	cmd := exec.CommandContext(ctx, "git", "clone", opts.repoUrl(), opts.destination(), "--branch", opts.Reference)
	return cmd.Run()
}

func clonePublic(ctx context.Context, opts CloneOptions) error {
	zipDst := filepath.Join(opts.TargetDir, opts.RepositoryName+".zip")

	// Download public repository from github as a ZIP file
	err := download(ctx, opts.zipUrl(), zipDst)
	if err != nil {
		return err
	}

	// Decompress the ZIP file
	err = zip.Extract(zipDst, opts.TargetDir)
	if err != nil {
		return err
	}

	// Remove the ZIP file post extraction
	return os.Remove(zipDst)
}

func Clone(ctx context.Context, opts CloneOptions) error {
	if opts.Provider != "github" {
		return fmt.Errorf("git provider not supported: %s", opts.Provider)
	}

	// First we try to clone the repository as a public URL, as that does not
	// require the git CLI
	err := clonePublic(ctx, opts)

	// If a public repository was not found, we defer to the git CLI
	if errors.Is(err, errNotFound) {

		return clonePrivate(ctx, opts)
	}
	return err
}
