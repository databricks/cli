package sandbox_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutableFromArchive(t *testing.T) {
	for _, name := range []string{"databricks", "../databricks", "other"} {
		t.Run(name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			file, err := writer.Create(name)
			require.NoError(t, err)
			_, err = file.Write([]byte("executable"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			executable, err := sandbox.ExecutableFromArchiveForTest(&buffer)
			if name != "databricks" {
				require.ErrorContains(t, err, "does not contain databricks")
				return
			}
			require.NoError(t, err)
			defer executable.Close()
			data, err := io.ReadAll(executable)
			require.NoError(t, err)
			assert.Equal(t, "executable", string(data))
		})
	}
	_, err := sandbox.ExecutableFromArchiveForTest(bytes.NewBufferString("invalid zip"))
	require.ErrorContains(t, err, "open Linux CLI archive")
}

func TestUploadExecutableValidatesArchitectureAndChmod(t *testing.T) {
	for _, tc := range []struct {
		architecture string
		exit         int
		errorText    string
	}{
		{"riscv64", 0, "unsupported sandbox architecture"},
		{"x86_64", 2, "make sandbox SSH executable runnable"},
		{"aarch64", 0, ""},
	} {
		t.Run(tc.architecture, func(t *testing.T) {
			directory := t.TempDir()
			for _, architecture := range []string{"amd64", "arm64"} {
				filename := "databricks_cli_" + build.GetInfo().Version + "_linux_" + architecture + ".zip"
				if build.GetInfo().IsDevelopment() {
					filename = "databricks_cli_linux_" + architecture + ".zip"
				}
				file, err := os.Create(filepath.Join(directory, filename))
				require.NoError(t, err)
				archive := zip.NewWriter(file)
				entry, err := archive.Create("databricks")
				require.NoError(t, err)
				_, err = entry.Write([]byte("executable"))
				require.NoError(t, err)
				require.NoError(t, archive.Close())
				require.NoError(t, file.Close())
			}
			uploads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/databricks-config" {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				if r.Method == http.MethodPut {
					uploads++
					assert.Equal(t, "/api/2.0/sandbox-files/sandboxes/test/content", r.URL.Path)
					assert.Equal(t, "/home/sandbox-agent/session/databricks", r.URL.Query().Get("path"))
					data, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.Equal(t, "executable", string(data))
					w.WriteHeader(http.StatusCreated)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "EXECUTE_COMMAND_STATUS_COMPLETED", "exit_code": tc.exit, "stderr": "chmod failed"})
			}))
			defer server.Close()
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)
			_, err = sandbox.UploadExecutableForTest(t.Context(), w, "sandboxes/test", &sandbox.RemoteSetupForTest{Architecture: tc.architecture, SessionDir: "/home/sandbox-agent/session"}, directory)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
			} else {
				require.NoError(t, err)
			}
			if tc.architecture == "riscv64" {
				assert.Zero(t, uploads)
			} else {
				assert.Equal(t, 1, uploads)
			}
		})
	}
}

func TestLinuxExecutableUsesCurrentLinuxBuild(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires a Linux executable")
	}
	reader, err := sandbox.LinuxExecutableForTest(t.Context(), runtime.GOARCH, "")
	require.NoError(t, err)
	defer reader.Close()
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	expected, err := os.ReadFile(executable)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestLinuxExecutableRequiresDevelopmentArchives(t *testing.T) {
	if !build.GetInfo().IsDevelopment() {
		t.Skip("requires a development build")
	}
	reader, err := sandbox.LinuxExecutableForTest(t.Context(), "different-architecture", "")
	require.ErrorContains(t, err, "requires --releases-dir")
	assert.Nil(t, reader)
}
