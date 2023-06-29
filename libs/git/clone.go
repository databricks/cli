package git

import (
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

func githubZipUrl(org string, name string, ref string) string {
	return fmt.Sprintf(`%s/archive/%s.zip`, githubUrl(org, name), ref)
}

func githubUrl(org string, name string) string {
	return fmt.Sprintf(`https://github.com/%s/%s`, org, name)
}

type RepositoryNotFoundError struct {
	url string
}

func (err RepositoryNotFoundError) Error() string {
	return fmt.Sprintf("repository not found: %s", err.url)
}

func (err RepositoryNotFoundError) Is(other error) bool {
	return other == errNotFound
}

// TODO: pass context to these the get requests
func download(url string, dest string) error {
	resp, err := http.Get(url)
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

func clonePrivate(org string, repoName string, targetDir string) error {
	zipUrl := githubUrl(org, repoName)

	// We append repoName-main to targetDir to be symmetric to clonePublic
	targetDir = filepath.Join(targetDir, repoName+"-main")

	// TODO: pass context to the command execution
	cmd := exec.Command("git", "clone", zipUrl, targetDir, "--branch", "main")
	return cmd.Run()
}

func clonePublic(org string, repoName string, targetDir string) error {
	zipDst := filepath.Join(targetDir, repoName+".zip")
	zipUrl := githubZipUrl(org, repoName, "main")

	// Download public repository from github as a ZIP file
	err := download(zipUrl, zipDst)
	if err != nil {
		return err
	}

	// Decompress the ZIP file
	err = zip.Extract(zipDst, targetDir)
	if err != nil {
		return err
	}

	// Remove the ZIP file
	return os.Remove(zipDst)
}

func Clone(org, repoName string, targetDir string) error {
	// First we try to clone the repository as a public URL, as that does not
	// require the git CLI
	err := clonePublic(org, repoName, targetDir)
	if err != nil && !errors.Is(err, errNotFound) {
		return err
	}

	// Since a public repository was not found, we defer to the git CLI
	return clonePrivate(org, repoName, targetDir)
}
