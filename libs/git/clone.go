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

	"github.com/databricks/cli/libs/cmdio"
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

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func clonePrivate(ctx context.Context, opts CloneOptions) error {
	cmd := exec.CommandContext(ctx, "git", "clone", opts.repoUrl(), opts.destination(), "--branch", opts.Reference)

	// Redirect exec command output
	cmd.Stderr = cmdio.Err(ctx)
	cmd.Stdout = cmdio.Out(ctx)
	cmd.Stdin = cmdio.In(ctx)

	// start git clone
	err := cmd.Start()
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("please install git CLI to download private templates: %w", err)
	}
	if err != nil {
		return err
	}

	// wait for git clone to complete
	return cmd.Wait()
}

func clonePublic(ctx context.Context, opts CloneOptions) error {
	tmpDir := os.TempDir()
	defer os.Remove(tmpDir)

	zipDst := filepath.Join(tmpDir, opts.RepositoryName+".zip")

	// Download public repository from github as a ZIP file
	err := download(ctx, opts.zipUrl(), zipDst)
	if err != nil {
		return err
	}
	defer os.Remove(zipDst)

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
