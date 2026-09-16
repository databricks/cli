package acceptance_test

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type downloadRoundTripper func(*http.Request) (*http.Response, error)

func (f downloadRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDownloadCLIReusesCachedVersions(t *testing.T) {
	execName := "databricks"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create(execName)
	require.NoError(t, err)
	_, err = io.WriteString(entry, "cached CLI")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	downloads := 0
	originalClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = originalClient })
	http.DefaultClient = &http.Client{Transport: downloadRoundTripper(func(r *http.Request) (*http.Response, error) {
		downloads++
		assert.Equal(t, "github.com", r.URL.Host)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archive.Bytes())),
			Header:     make(http.Header),
		}, nil
	})}

	root := t.TempDir()
	for _, tc := range []struct {
		name      string
		version   string
		downloads int
	}{
		{"first", "1.8.0", 1},
		{"same_version", "1.8.0", 1},
		{"different_version", "1.9.0", 2},
		{"first_version_again", "1.8.0", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buildDir := getBuildDir(t, root, runtime.GOOS, runtime.GOARCH)
			path := DownloadCLI(t, buildDir, tc.version)
			assert.Equal(t, filepath.Join(buildDir, tc.version, execName), path)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "cached CLI", string(data))
			assert.Equal(t, tc.downloads, downloads)
		})
	}
}
