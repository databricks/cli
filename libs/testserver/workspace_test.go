package testserver_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
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
// extension. Its object info reports PYTHON whatever the metadata says, as the
// real API does. Other files keep their name.
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
	assert.Equal(t, "PYTHON", sql["language"])

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

// doRequest sends body to the test server and returns the status and response body.
func doRequest(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(out)
}

func workspaceImport(t *testing.T, baseURL string, request map[string]any) (int, string) {
	t.Helper()
	request["content"] = base64.StdEncoding.EncodeToString([]byte(request["content"].(string)))
	body, err := json.Marshal(request)
	require.NoError(t, err)
	return doRequest(t, http.MethodPost, baseURL+"/api/2.0/workspace/import", string(body))
}

// workspaceExport returns the decoded content of GET /workspace/export, or the error body.
func workspaceExport(t *testing.T, baseURL, path, format string) (int, string) {
	t.Helper()
	status, body := doRequest(t, http.MethodGet, baseURL+"/api/2.0/workspace/export?path="+path+"&format="+format, "")
	if status != 200 {
		return status, body
	}
	var resp struct {
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	content, err := base64.StdEncoding.DecodeString(resp.Content)
	require.NoError(t, err)
	return status, string(content)
}

// The real export converts a notebook to the requested format, whatever format
// it was imported in. The SOURCE text and the cell sources are what the real
// API returned on AWS for these notebooks.
func TestWorkspaceExportConvertsNotebookFormat(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	const source = "# Databricks notebook source\nprint(1)\nprint(11)\n\n# COMMAND ----------\n\nprint(2)\n"
	status, _ := workspaceImport(t, server.URL, map[string]any{"path": "/d/src", "content": source, "format": "SOURCE", "language": "PYTHON"})
	require.Equal(t, 200, status)

	status, body := workspaceExport(t, server.URL, "/d/src", "SOURCE")
	require.Equal(t, 200, status)
	assert.Equal(t, source, body)

	status, body = workspaceExport(t, server.URL, "/d/src", "JUPYTER")
	require.Equal(t, 200, status)
	var nb struct {
		Cells []struct {
			CellType string   `json:"cell_type"`
			Source   []string `json:"source"`
		} `json:"cells"`
		Metadata struct {
			LanguageInfo struct {
				Name string `json:"name"`
			} `json:"language_info"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &nb))
	require.Len(t, nb.Cells, 2)
	assert.Equal(t, []string{"print(1)\n", "print(11)"}, nb.Cells[0].Source)
	assert.Equal(t, []string{"print(2)"}, nb.Cells[1].Source)
	assert.Equal(t, "code", nb.Cells[1].CellType)
	assert.Equal(t, "python", nb.Metadata.LanguageInfo.Name)

	// A notebook with only the header exports as a notebook without cells.
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/empty", "content": "# Databricks notebook source\n", "format": "SOURCE", "language": "PYTHON"})
	require.Equal(t, 200, status)
	status, body = workspaceExport(t, server.URL, "/d/empty", "JUPYTER")
	require.Equal(t, 200, status)
	assert.Contains(t, body, `"cells": []`)

	const ipynb = `{"cells":[{"cell_type":"code","execution_count":null,"metadata":{},"outputs":[],"source":["print(1)\n","print(11)"]},{"cell_type":"markdown","metadata":{},"source":["# hi"]},{"cell_type":"code","execution_count":null,"metadata":{},"outputs":[],"source":["print(2)"]}],"metadata":{"language_info":{"name":"python"}},"nbformat":4,"nbformat_minor":2}`
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/nb", "content": ipynb, "format": "JUPYTER"})
	require.Equal(t, 200, status)

	for _, format := range []string{"SOURCE", ""} {
		status, body = workspaceExport(t, server.URL, "/d/nb", format)
		require.Equal(t, 200, status)
		assert.Equal(t, "# Databricks notebook source\nprint(1)\nprint(11)\n\n# COMMAND ----------\n\n# MAGIC %md\n# MAGIC # hi\n\n# COMMAND ----------\n\nprint(2)", body, format)
	}

	// JUPYTER export keeps the cells and reports the language in lower case.
	status, body = workspaceExport(t, server.URL, "/d/nb", "JUPYTER")
	require.Equal(t, 200, status)
	nb.Cells = nil
	require.NoError(t, json.Unmarshal([]byte(body), &nb))
	require.Len(t, nb.Cells, 3)
	assert.Equal(t, "markdown", nb.Cells[1].CellType)
	assert.Equal(t, []string{"# hi"}, nb.Cells[1].Source)
	assert.Equal(t, "python", nb.Metadata.LanguageInfo.Name)
}

// Magic cells survive a SOURCE -> JUPYTER -> SOURCE round trip, and the
// converters accept string cell sources, CRLF line endings and a missing language.
func TestWorkspaceExportNotebookCellShapes(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	const source = "# Databricks notebook source\r\nprint(1)\r\n\r\n# COMMAND ----------\r\n\r\n# MAGIC %md\r\n# MAGIC # hi\r\n\r\n# COMMAND ----------\r\n\r\n# MAGIC %sql\r\n# MAGIC SELECT 1"
	status, _ := workspaceImport(t, server.URL, map[string]any{"path": "/d/src", "content": source, "format": "SOURCE", "language": "PYTHON"})
	require.Equal(t, 200, status)
	status, ipynb := workspaceExport(t, server.URL, "/d/src", "JUPYTER")
	require.Equal(t, 200, status)
	var nb struct {
		Cells []struct {
			CellType string   `json:"cell_type"`
			Source   []string `json:"source"`
		} `json:"cells"`
	}
	require.NoError(t, json.Unmarshal([]byte(ipynb), &nb))
	require.Len(t, nb.Cells, 3)
	assert.Equal(t, "code", nb.Cells[0].CellType)
	assert.Equal(t, []string{"print(1)"}, nb.Cells[0].Source)
	assert.Equal(t, "markdown", nb.Cells[1].CellType)
	assert.Equal(t, []string{"# hi"}, nb.Cells[1].Source)
	assert.Equal(t, "code", nb.Cells[2].CellType)
	assert.Equal(t, []string{"%sql\n", "SELECT 1"}, nb.Cells[2].Source)

	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/round", "content": ipynb, "format": "JUPYTER"})
	require.Equal(t, 200, status)
	_, body := workspaceExport(t, server.URL, "/d/round", "SOURCE")
	assert.Equal(t, "# Databricks notebook source\nprint(1)\n\n# COMMAND ----------\n\n# MAGIC %md\n# MAGIC # hi\n\n# COMMAND ----------\n\n# MAGIC %sql\n# MAGIC SELECT 1", body)

	// A string cell source and no language metadata: the real API stores it as Python.
	const stringSource = `{"cells":[{"cell_type":"code","execution_count":null,"metadata":{},"outputs":[],"source":"print(1)\nprint(2)"}],"metadata":{},"nbformat":4,"nbformat_minor":2}`
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/str", "content": stringSource, "format": "JUPYTER"})
	require.Equal(t, 200, status)
	_, body = workspaceExport(t, server.URL, "/d/str", "SOURCE")
	assert.Equal(t, "# Databricks notebook source\nprint(1)\nprint(2)", body)
	assert.Equal(t, "PYTHON", getStatusBody(t, server.URL, "/d/str", false)["language"])

	// Only the first line is the header; MAGIC must be a word; HTML characters stay as they are.
	const plain = "x = 1\n# Databricks notebook source\n\n# COMMAND ----------\n\n# MAGICAL = 1 < 2"
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/plain", "content": plain, "format": "SOURCE", "language": "PYTHON"})
	require.Equal(t, 200, status)
	_, body = workspaceExport(t, server.URL, "/d/plain", "SOURCE")
	assert.Equal(t, "# Databricks notebook source\n"+plain, body)
	_, body = workspaceExport(t, server.URL, "/d/plain", "JUPYTER")
	assert.Contains(t, body, `"x = 1\n"`)
	assert.Contains(t, body, `"# MAGICAL = 1 < 2"`)

	// An empty markdown line is the bare MAGIC prefix (recorded on AWS).
	const markdown = `{"cells":[{"cell_type":"markdown","metadata":{},"source":["# hi\n","\n","more"]}],"metadata":{"language_info":{"name":"python"}},"nbformat":4,"nbformat_minor":2}`
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/md", "content": markdown, "format": "JUPYTER"})
	require.Equal(t, 200, status)
	_, body = workspaceExport(t, server.URL, "/d/md", "SOURCE")
	assert.Equal(t, "# Databricks notebook source\n# MAGIC %md\n# MAGIC # hi\n# MAGIC\n# MAGIC more", body)

	// HTML, DBC and R_MARKDOWN exports of a notebook are not modeled.
	for _, format := range []string{"HTML", "DBC", "R_MARKDOWN"} {
		status, _ = workspaceExport(t, server.URL, "/d/md", format)
		assert.Equal(t, 500, status, format)
	}

	// A body that is not a Jupyter notebook is not stored by a JUPYTER import.
	status, _ = workspaceImport(t, server.URL, map[string]any{"path": "/d/bad", "content": "hello", "format": "JUPYTER"})
	assert.Equal(t, 500, status)
	assert.Equal(t, 404, getStatus(t, server.URL, "/d/bad"))
}

// The real export adds the header line to a source notebook imported without one.
func TestWorkspaceExportAddsNotebookHeader(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	for language, header := range map[string]string{"PYTHON": "# ", "SQL": "-- ", "SCALA": "// ", "R": "# "} {
		path := "/d/" + language
		status, _ := workspaceImport(t, server.URL, map[string]any{"path": path, "content": "x = 1", "format": "SOURCE", "language": language})
		require.Equal(t, 200, status)
		_, body := workspaceExport(t, server.URL, path, "SOURCE")
		assert.Equal(t, header+"Databricks notebook source\nx = 1", body, language)
	}
}

func TestWorkspaceExportFileAsJupyterFails(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")
	require.Equal(t, 200, importFile(t, server.URL, "/d/plain.txt", "abc"))

	status, body := workspaceExport(t, server.URL, "/d/plain.txt", "JUPYTER")
	assert.Equal(t, 400, status)
	assert.Contains(t, body, "Exporting a FILE asset at /d/plain.txt in JUPYTER format is not supported.")

	status, body = workspaceExport(t, server.URL, "/d/plain.txt", "SOURCE")
	assert.Equal(t, 200, status)
	assert.Equal(t, "abc", body)
}

// Without a language, format=SOURCE (the default) means a zip archive of notebooks.
func TestWorkspaceImportSourceWithoutLanguageExpectsZip(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	var emptyZip bytes.Buffer
	require.NoError(t, zip.NewWriter(&emptyZip).Close())
	for _, content := range []string{"// Databricks notebook source\nprintln(1)", emptyZip.String()} {
		for _, format := range []string{"SOURCE", ""} {
			status, body := workspaceImport(t, server.URL, map[string]any{"path": "/d/nb", "content": content, "format": format})
			assert.Equal(t, 400, status, format)
			assert.Contains(t, body, "The zip archive contains no items.", format)
		}
	}
	assert.Equal(t, 404, getStatus(t, server.URL, "/d/nb"))

	// Extracting an archive of notebooks is not modeled, so the test server says so.
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create("a.py")
	require.NoError(t, err)
	_, err = w.Write([]byte("# Databricks notebook source\nprint(1)"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	status, _ := workspaceImport(t, server.URL, map[string]any{"path": "/d/nb", "content": archive.String()})
	assert.Equal(t, 500, status)
	assert.Equal(t, 404, getStatus(t, server.URL, "/d/nb"))
}

// The real workspace always has a root directory to list.
func TestWorkspaceListRoot(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	status, body := doRequest(t, http.MethodGet, server.URL+"/api/2.0/workspace/list?path=/", "")
	assert.Equal(t, 200, status)
	assert.Contains(t, body, `"path": "/Users"`)
}

// A notebook import without overwrite keeps an existing notebook at its path.
func TestWorkspaceImportNotebookKeepsExistingNotebook(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	const old = "# Databricks notebook source\nprint(0)"
	status, _ := workspaceImport(t, server.URL, map[string]any{"path": "/d/x.py", "content": old, "format": "AUTO"})
	require.Equal(t, 200, status)

	status, body := doRequest(t, http.MethodPost, server.URL+"/api/2.0/workspace-files/import-file/d/x.py?overwrite=false", "# Databricks notebook source\nprint(1)")
	assert.Equal(t, 400, status)
	assert.Contains(t, body, "Request failed for POST /d/x.py: RESOURCE_ALREADY_EXISTS: /d/x already exists. Please pass overwrite=true to overwrite it.")

	// The messages /workspace/import returned on AWS for AUTO and SOURCE.
	status, body = workspaceImport(t, server.URL, map[string]any{"path": "/d/x.py", "content": "# Databricks notebook source\nprint(2)", "format": "AUTO"})
	assert.Equal(t, 400, status)
	assert.Contains(t, body, "RESOURCE_ALREADY_EXISTS")
	assert.Contains(t, body, "/d/x already exists. Please pass overwrite=true to overwrite it.")
	status, body = workspaceImport(t, server.URL, map[string]any{"path": "/d/x", "content": "print(2)", "format": "SOURCE", "language": "PYTHON"})
	assert.Equal(t, 400, status)
	assert.Contains(t, body, "/d/x already exists. Please pass overwrite=true to overwrite it.")

	_, body = workspaceExport(t, server.URL, "/d/x", "SOURCE")
	assert.Equal(t, old, body)

	status, _ = doRequest(t, http.MethodPost, server.URL+"/api/2.0/workspace-files/import-file/d/x.py?overwrite=true", "# Databricks notebook source\nprint(3)")
	assert.Equal(t, 200, status)
	_, body = workspaceExport(t, server.URL, "/d/x", "SOURCE")
	assert.Equal(t, "# Databricks notebook source\nprint(3)", body)
}

// A Jupyter notebook's object info reports PYTHON, while its export uses the
// content language: language_info.name, else the Databricks metadata language,
// else Python. These pairs are what the real API returned on AWS for both
// /workspace/import with format=JUPYTER and import-file.
func TestWorkspaceJupyterNotebookLanguage(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	mkdirs(t, server.URL, "/d")

	cases := []struct {
		name, metadata, header, jupyterName string
	}{
		{"kr", `"language_info":{"name":"R"}`, "# ", "r"},
		{"kscala", `"language_info":{"name":"scala"}`, "// ", "scala"},
		{"ksql", `"language_info":{"name":"sql"}`, "-- ", "sql"},
		{"kpy", `"language_info":{"name":"python"}`, "# ", "python"},
		{"dbsql", `"application/vnd.databricks.v1+notebook":{"language":"sql"},"language_info":{"name":"python"}`, "# ", "python"},
		{"dbscala", `"application/vnd.databricks.v1+notebook":{"language":"scala"}`, "// ", "scala"},
	}
	for _, tc := range cases {
		body := `{"cells":[{"cell_type":"code","execution_count":null,"metadata":{},"outputs":[],"source":["x"]}],"metadata":{` + tc.metadata + `},"nbformat":4,"nbformat_minor":2}`
		for _, route := range []string{"import", "import-file"} {
			path := "/d/" + route + "-" + tc.name
			if route == "import" {
				status, _ := workspaceImport(t, server.URL, map[string]any{"path": path, "content": body, "format": "JUPYTER"})
				require.Equal(t, 200, status, path)
			} else {
				require.Equal(t, 200, importFile(t, server.URL, path+".ipynb", body), path)
			}

			info := getStatusBody(t, server.URL, path, false)
			assert.Equal(t, "NOTEBOOK", info["object_type"], path)
			assert.Equal(t, "PYTHON", info["language"], path)

			_, source := workspaceExport(t, server.URL, path, "SOURCE")
			assert.Equal(t, tc.header+"Databricks notebook source\nx", source, path)

			_, ipynb := workspaceExport(t, server.URL, path, "JUPYTER")
			var nb struct {
				Metadata struct {
					LanguageInfo struct {
						Name string `json:"name"`
					} `json:"language_info"`
					Databricks struct {
						Language string `json:"language"`
					} `json:"application/vnd.databricks.v1+notebook"`
				} `json:"metadata"`
			}
			require.NoError(t, json.Unmarshal([]byte(ipynb), &nb), path)
			assert.Equal(t, tc.jupyterName, nb.Metadata.LanguageInfo.Name, path)
			assert.Equal(t, tc.jupyterName, nb.Metadata.Databricks.Language, path)
		}
	}
}
