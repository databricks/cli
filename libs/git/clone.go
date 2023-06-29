package git

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/zip"
)

func githubZipUrl(org string, name string, ref string) string {
	return fmt.Sprintf(`https://github.com/%s/%s/archive/%s.zip`, org, name, ref)
}

func download(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("repository not found: %s", url)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download ZIP archive: %s. %s", url, resp.Status)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func Clone(org string, repoName string, targetDir string) error {
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
