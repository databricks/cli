package sandbox_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sandboxcmd "github.com/databricks/cli/cmd/sandbox"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const filesRoute = "/api/2.0/sandbox-files/sandboxes/test-sandbox"

func sandboxContext(t *testing.T, server *testserver.Server) context.Context {
	t.Helper()
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	state, err := json.Marshal(map[string]any{
		"sandboxes": map[string]any{server.URL: []map[string]string{{"id": "test-sandbox", "name": "my-sandbox"}}},
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".databricks"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".databricks", "sandbox.json"), state, 0o600))
	return cmdctx.SetWorkspaceClient(ctx, w)
}

func runFsCommand(t *testing.T, ctx context.Context, name string, args ...string) (string, error) {
	t.Helper()
	ctx, stdout := cmdio.NewTestContextWithStdout(ctx)
	cmd, _, err := sandboxcmd.New().Find([]string{"fs", name})
	require.NoError(t, err)
	cmd.SetContext(ctx)
	require.NoError(t, cmd.ParseFlags(args))
	err = cmd.RunE(cmd, cmd.Flags().Args())
	return stdout.String(), err
}

func TestSandboxFsCopySkipsAndOverwrites(t *testing.T) {
	server := testserver.New(t)
	ctx := sandboxContext(t, server)
	contents := "original"
	server.Handle("GET", filesRoute+"/paths:describe", func(req testserver.Request) any {
		name := req.URL.Query().Get("path")
		fileType := sandbox.SandboxFileTypeSandboxFileTypeFile
		if name == "/remote" {
			fileType = sandbox.SandboxFileTypeSandboxFileTypeDirectory
		}
		return sandbox.GetMetadataResponse{File: &sandbox.SandboxFileInfo{Path: name, FileType: fileType}}
	})
	var overwrites []string
	server.Handle("PUT", filesRoute+"/content", func(req testserver.Request) any {
		assert.Equal(t, "/remote/file", req.URL.Query().Get("path"))
		overwrite := req.URL.Query().Get("overwrite")
		overwrites = append(overwrites, overwrite)
		if overwrite == "false" {
			return testserver.Response{StatusCode: 412, Body: map[string]string{
				"error_code": "RESOURCE_ALREADY_EXISTS", "message": "already exists",
			}}
		}
		contents = string(req.Body)
		return testserver.Response{StatusCode: 204}
	})
	local := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(local, []byte("replacement"), 0o600))
	output, err := runFsCommand(t, ctx, "cp", local, "my-sandbox:/remote/file")
	require.NoError(t, err)
	assert.Contains(t, output, "my-sandbox:/remote/file (skipped; already exists)")
	assert.Equal(t, "original", contents)
	output, err = runFsCommand(t, ctx, "cp", local, "my-sandbox:/remote/file", "--overwrite")
	require.NoError(t, err)
	assert.NotContains(t, output, "skipped")
	assert.Equal(t, "replacement", contents)
	assert.Equal(t, []string{"false", "true"}, overwrites)
}

func TestSandboxFsRecursiveDownload(t *testing.T) {
	server := testserver.New(t)
	ctx := sandboxContext(t, server)
	server.Handle("GET", filesRoute+"/paths:describe", func(req testserver.Request) any {
		return sandbox.GetMetadataResponse{File: &sandbox.SandboxFileInfo{
			Path: "/remote", FileType: sandbox.SandboxFileTypeSandboxFileTypeDirectory,
		}}
	})
	server.Handle("GET", filesRoute+"/directory-entries", func(req testserver.Request) any {
		if req.URL.Query().Get("path") == "/remote" {
			return sandbox.ListDirectoryContentsResponse{Contents: []sandbox.SandboxFileInfo{
				{Path: "/remote/file", FileType: sandbox.SandboxFileTypeSandboxFileTypeFile},
				{Path: "/remote/sub", FileType: sandbox.SandboxFileTypeSandboxFileTypeDirectory},
			}}
		}
		return sandbox.ListDirectoryContentsResponse{Contents: []sandbox.SandboxFileInfo{
			{Path: "/remote/sub/nested", FileType: sandbox.SandboxFileTypeSandboxFileTypeFile},
		}}
	})
	server.Handle("GET", filesRoute+"/content", func(req testserver.Request) any {
		return []byte(req.URL.Query().Get("path"))
	})
	target := filepath.Join(t.TempDir(), "output")
	_, err := runFsCommand(t, ctx, "cp", "-r", "my-sandbox:/remote", target)
	require.NoError(t, err)
	for _, name := range []string{"file", "sub/nested"} {
		contents, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(name)))
		require.NoError(t, err)
		assert.Equal(t, "/remote/"+name, string(contents))
	}
}

func TestSandboxFsCatAndLs(t *testing.T) {
	server := testserver.New(t)
	ctx := sandboxContext(t, server)
	server.Handle("GET", filesRoute+"/content", func(req testserver.Request) any {
		return []byte("contents")
	})
	server.Handle("GET", filesRoute+"/directory-entries", func(req testserver.Request) any {
		return sandbox.ListDirectoryContentsResponse{Contents: []sandbox.SandboxFileInfo{
			{Path: "/tmp/z", FileType: sandbox.SandboxFileTypeSandboxFileTypeFile},
			{Path: "/tmp/a", FileType: sandbox.SandboxFileTypeSandboxFileTypeDirectory},
		}}
	})
	output, err := runFsCommand(t, ctx, "cat", "my-sandbox:/tmp/z")
	require.NoError(t, err)
	assert.Equal(t, "contents", output)
	output, err = runFsCommand(t, ctx, "ls", "my-sandbox:/tmp", "--absolute")
	require.NoError(t, err)
	assert.Equal(t, "my-sandbox:/tmp/a\nmy-sandbox:/tmp/z\n", output)
}

func TestSandboxFsRejectsRelativeRemotePaths(t *testing.T) {
	for _, input := range []string{"my-sandbox:relative", ":/tmp/file"} {
		t.Run(input, func(t *testing.T) {
			_, err := runFsCommand(t, t.Context(), "cat", input)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "sandbox")
		})
	}
}
