package testserver_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func importFile(t *testing.T, baseURL, path, body string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/workspace-files/import-file/"+strings.TrimPrefix(path, "/")+"?overwrite=true", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func mkdirs(t *testing.T, baseURL, path string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/workspace/mkdirs", strings.NewReader(`{"path":"`+path+`"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
}

func workspaceDelete(t *testing.T, baseURL, path string, recursive bool) int {
	t.Helper()
	body, err := json.Marshal(map[string]any{"path": path, "recursive": recursive})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/workspace/delete", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func getStatus(t *testing.T, baseURL, path string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/2.0/workspace/get-status?path="+path, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// The real import API returns a 404 when the parent directory does not exist; it
// does not create it. Callers get "mkdir -p" semantics only by calling mkdirs first.
func TestWorkspaceImportRejectsMissingParent(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	assert.Equal(t, 404, importFile(t, server.URL, "/test-dir/file.py", "content"))

	mkdirs(t, server.URL, "/test-dir")
	assert.Equal(t, 200, importFile(t, server.URL, "/test-dir/file.py", "content"))
}

// A non-recursive delete only removes a directory that is truly empty, which is what
// lets a caller clean up a parent directory without touching one still in use.
func TestWorkspaceDeleteNonRecursiveRequiresEmptyDirectory(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	// A subdirectory keeps the parent.
	mkdirs(t, server.URL, "/a/b/c")
	assert.Equal(t, 400, workspaceDelete(t, server.URL, "/a/b", false))
	assert.Equal(t, 200, getStatus(t, server.URL, "/a/b"))

	// So does a file.
	mkdirs(t, server.URL, "/f/dir")
	require.Equal(t, 200, importFile(t, server.URL, "/f/dir/file.py", "content"))
	assert.Equal(t, 400, workspaceDelete(t, server.URL, "/f/dir", false))
	assert.Equal(t, 200, getStatus(t, server.URL, "/f/dir"))

	// Emptied out, both go.
	assert.Equal(t, 200, workspaceDelete(t, server.URL, "/a/b/c", false))
	assert.Equal(t, 200, workspaceDelete(t, server.URL, "/a/b", false))
	assert.Equal(t, 404, getStatus(t, server.URL, "/a/b"))
}

// format=AUTO matches the extension case-insensitively, so ".R" — the
// conventional spelling for R sources — is detected as a notebook just like
// ".r". Notebooks are stored with the extension stripped.
func TestWorkspaceImportDetectsNotebookExtensionCaseInsensitively(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	mkdirs(t, server.URL, "/test-dir")

	for _, name := range []string{"lower.r", "upper.R"} {
		require.Equal(t, 200, importFile(t, server.URL, "/test-dir/"+name, "# Databricks notebook source\n"), name)
	}

	// Both land at the extension-stripped path, which only happens for notebooks.
	assert.Equal(t, 200, getStatus(t, server.URL, "/test-dir/lower"))
	assert.Equal(t, 200, getStatus(t, server.URL, "/test-dir/upper"))
}

// import-file turns a valid Jupyter notebook into a notebook without the .ipynb
// extension, in the language its metadata names. Other files keep their name.
func TestWorkspaceImportFileJupyter(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	mkdirs(t, server.URL, "/test-dir")

	const kernelOnly = `{"cells":[],"metadata":{"language_info":{"name":"python"}},"nbformat":4,"nbformat_minor":2}`
	const databricksSQL = `{"cells":[],"metadata":{"application/vnd.databricks.v1+notebook":{"language":"sql"},"language_info":{"name":"python"}},"nbformat":4,"nbformat_minor":2}`
	files := map[string]string{
		"kernel.ipynb":     kernelOnly,
		"sql.ipynb":        databricksSQL,
		"broken.ipynb":     `{"cells": [`,
		"d.designer.ipynb": kernelOnly,
		"q.lwquery.ipynb":  kernelOnly,
	}
	for name, body := range files {
		require.Equal(t, 200, importFile(t, server.URL, "/test-dir/"+name, body), name)
	}

	kernel := getStatusBody(t, server.URL, "/test-dir/kernel", false)
	assert.Equal(t, "NOTEBOOK", kernel["object_type"])
	assert.Equal(t, "PYTHON", kernel["language"])

	sql := getStatusBody(t, server.URL, "/test-dir/sql", false)
	assert.Equal(t, "NOTEBOOK", sql["object_type"])
	assert.Equal(t, "SQL", sql["language"])

	for _, name := range []string{"broken.ipynb", "d.designer.ipynb", "q.lwquery.ipynb"} {
		assert.Equal(t, "FILE", getStatusBody(t, server.URL, "/test-dir/"+name, false)["object_type"], name)
	}
	assert.Equal(t, 404, getStatus(t, server.URL, "/test-dir/broken"))
}

// The Files API stores a notebook as uploaded, so it reads back at its own path.
func TestFilesUploadKeepsNotebookAsFile(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	const nb = `{"cells":[],"metadata":{"language_info":{"name":"python"}},"nbformat":4,"nbformat_minor":2}`
	url := server.URL + "/api/2.0/fs/files/Volumes/main/default/vol/n.ipynb"
	do := func(method, body string) (int, string) {
		req, _ := http.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		out, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(out)
	}

	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/2.0/fs/directories/Volumes/main/default/vol", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	status, _ := do(http.MethodPut, nb)
	require.Equal(t, 200, status)
	status, _ = do(http.MethodHead, "")
	assert.Equal(t, 200, status)
	status, body := do(http.MethodGet, "")
	assert.Equal(t, 200, status)
	assert.Equal(t, nb, body)
}

// mkdirs creates all intermediate directories, matching "mkdir -p".
func TestWorkspaceMkdirsRecursive(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	mkdirs(t, server.URL, "/a/b/c")

	for _, dir := range []string{"/a", "/a/b", "/a/b/c"} {
		assert.Equal(t, 200, getStatus(t, server.URL, dir), dir)
	}
}

func createRepo(t *testing.T, baseURL, path string) {
	t.Helper()
	body := `{"url":"https://github.com/databricks/cli","provider":"gitHub","path":"` + path + `"}`
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/2.0/repos", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
}

func getStatusBody(t *testing.T, baseURL, path string, returnGitInfo bool) map[string]any {
	t.Helper()
	url := baseURL + "/api/2.0/workspace/get-status?path=" + path
	if returnGitInfo {
		url += "&return_git_info=true"
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

// How a Git folder is described on the wire: the object type, directory_info and
// the presence of git_info. The acceptance tests read the resolved metadata
// rather than the response, so these fields are only pinned here.

// A standard Git folder keeps the REPO object type and reports the full metadata,
// and only when it is asked for.
func TestWorkspaceGetStatusGitInfoForRepo(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	createRepo(t, server.URL, "/Repos/me/myrepo")

	assert.NotContains(t, getStatusBody(t, server.URL, "/Repos/me/myrepo", false), "git_info")

	body := getStatusBody(t, server.URL, "/Repos/me/myrepo", true)
	assert.Equal(t, "REPO", body["object_type"])
	assert.NotContains(t, body, "directory_info")

	gitInfo, ok := body["git_info"].(map[string]any)
	require.True(t, ok, "expected git_info in %v", body)
	assert.Equal(t, "main", gitInfo["branch"])
	assert.NotEmpty(t, gitInfo["head_commit_id"])
	assert.Equal(t, "https://github.com/databricks/cli", gitInfo["url"])
}

// A Git folder with Git CLI access is a DIRECTORY that marks itself through
// directory_info, and reports only the id and path.
func TestWorkspaceGetStatusGitInfoForGitCliFolder(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	createRepo(t, server.URL, "/Workspace/Users/me/gitfolder")
	body := getStatusBody(t, server.URL, "/Workspace/Users/me/gitfolder", true)

	assert.Equal(t, "DIRECTORY", body["object_type"])
	dirInfo, ok := body["directory_info"].(map[string]any)
	require.True(t, ok, "expected directory_info in %v", body)
	assert.Equal(t, true, dirInfo["is_git_folder"])

	gitInfo, ok := body["git_info"].(map[string]any)
	require.True(t, ok, "expected git_info in %v", body)
	assert.Equal(t, "/Users/me/gitfolder", gitInfo["path"])
	assert.NotEmpty(t, gitInfo["id"])
	assert.NotContains(t, gitInfo, "branch")
	assert.NotContains(t, gitInfo, "head_commit_id")
	assert.NotContains(t, gitInfo, "url")
}
