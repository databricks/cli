package generate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	workspace "github.com/databricks/cli/libs/workspaceapi"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_MarkFileReturnsRelativePath(t *testing.T) {
	ctx := t.Context()
	f1 := "/a/b/c"
	f2 := "/a/b/c/d"
	w := newTestWorkspaceClient(t, objectsHandler(t, []string{f1, f2}, nil))

	dir := "base/dir/doesnt/matter"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	var err error

	// Test that the path is normalized to be relative to the config directory.
	err = downloader.markFileForDownload(ctx, &f1)
	require.NoError(t, err)
	assert.Equal(t, "../source/c", f1)

	// Test that the previous path doesn't influence the next path.
	err = downloader.markFileForDownload(ctx, &f2)
	require.NoError(t, err)
	assert.Equal(t, "../source/d", f2)
}

func TestDownloader_DoesNotRecurseIntoNodeModules(t *testing.T) {
	ctx := t.Context()
	rootPath := "/workspace/app"

	// Root directory contains: app.py, src/, node_modules/; src/ contains index.js.
	// node_modules is not listed: listing it fails the test.
	w := newTestWorkspaceClient(t, objectsHandler(t,
		[]string{rootPath, "/workspace/app/app.py", "/workspace/app/src/index.js"},
		map[string][]workspace.ObjectInfo{
			rootPath: {
				{Path: "/workspace/app/app.py", ObjectType: workspace.ObjectTypeFile},
				{Path: "/workspace/app/src", ObjectType: workspace.ObjectTypeDirectory},
				{Path: "/workspace/app/node_modules", ObjectType: workspace.ObjectTypeDirectory},
			},
			"/workspace/app/src": {
				{Path: "/workspace/app/src/index.js", ObjectType: workspace.ObjectTypeFile},
			},
		},
	))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	// Execute
	err := downloader.MarkDirectoryForDownload(ctx, &rootPath)
	require.NoError(t, err)

	// Verify only 2 files were marked (not any from node_modules)
	assert.Len(t, downloader.files, 2)
	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "app.py"))
	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "src/index.js"))
}

func TestCommonDirPrefix(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{
			name:  "empty",
			paths: nil,
			want:  "",
		},
		{
			name:  "single path",
			paths: []string{"/a/b/c"},
			want:  "/a/b",
		},
		{
			name:  "shared parent",
			paths: []string{"/a/b/c", "/a/b/d"},
			want:  "/a/b",
		},
		{
			name:  "root divergence",
			paths: []string{"/x/y", "/z/w"},
			want:  "",
		},
		{
			name:  "partial dir name safety",
			paths: []string{"/a/bc/d", "/a/bd/e"},
			want:  "/a",
		},
		{
			name:  "nested shared prefix",
			paths: []string{"/Users/user/project/etl/extract", "/Users/user/project/reporting/dashboard"},
			want:  "/Users/user/project",
		},
		{
			name:  "identical paths",
			paths: []string{"/a/b/c", "/a/b/c"},
			want:  "/a/b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, commonDirPrefix(tt.paths))
		})
	}
}

// objectsHandler serves get-status for the given objects and list for the
// given directory contents; any other request fails the test.
func objectsHandler(t *testing.T, objects []string, dirs map[string][]workspace.ObjectInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Query().Get("path")
		switch r.URL.Path {
		case "/api/2.0/workspace/get-status":
			if !slices.Contains(objects, path) {
				t.Errorf("unexpected get-status path: %s", path)
				http.NotFound(w, r)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(workspace.ObjectInfo{Path: path}))
		case "/api/2.0/workspace/list":
			children, ok := dirs[path]
			if !ok {
				t.Errorf("unexpected list path: %s", path)
				http.NotFound(w, r)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"objects": children}))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}
}

func newTestWorkspaceClient(t *testing.T, handler http.HandlerFunc) *sdkclients.Clients {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			http.NotFound(w, r)
			return
		}

		handler(w, r)
	}))
	t.Cleanup(server.Close)

	w, err := sdkclients.New(t.Context(), &config.Config{
		Host:    server.URL,
		Token:   "test-token",
		Loaders: []config.Loader{config.ConfigAttributes},
	})
	require.NoError(t, err)
	return w
}

func notebookStatusHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/workspace/get-status" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		resp := workspaceStatus{
			Language:     workspace.LanguagePython,
			ObjectType:   workspace.ObjectTypeNotebook,
			ExportFormat: workspace.ExportFormatSource,
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// designerStatusHandler mimics get-status for a Lakeflow Designer file: object
// type DESIGNER_FILE and no export format reported.
func designerStatusHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/workspace/get-status" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		resp := workspaceStatus{
			ObjectType: workspace.ObjectType("DESIGNER_FILE"),
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(resp))
	}
}

func TestDownloader_MarkTasksForDownload_DesignerNotebook(t *testing.T) {
	ctx := t.Context()
	w := newTestWorkspaceClient(t, designerStatusHandler(t))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("designer_task"),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: new("/Users/user/project/My Pipeline.designer.ipynb"),
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)

	// The ".designer.ipynb" suffix must be preserved, not stripped to ".designer".
	assert.Equal(t, "../source/My Pipeline.designer.ipynb", *tasks[0].NotebookTask.NotebookPath)
	require.Len(t, downloader.files, 1)
	f := downloader.files[filepath.Join(sourceDir, "My Pipeline.designer.ipynb")]
	assert.Equal(t, "/Users/user/project/My Pipeline.designer.ipynb", f.path)
	// Designer files round-trip as Jupyter notebooks even though get-status
	// reports no export format.
	assert.Equal(t, workspace.ExportFormatJupyter, f.format)
}

func TestDownloader_MarkTasksForDownload_PreservesStructure(t *testing.T) {
	w := newTestWorkspaceClient(t, notebookStatusHandler(t))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("extract_task"),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: new("/Users/user/project/etl/extract"),
			},
		},
		{
			TaskKey: new("dashboard_task"),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: new("/Users/user/project/reporting/dashboard"),
			},
		},
	}

	err := downloader.MarkTasksForDownload(t.Context(), tasks)
	require.NoError(t, err)

	assert.Equal(t, "../source/etl/extract.py", *tasks[0].NotebookTask.NotebookPath)
	assert.Equal(t, "../source/reporting/dashboard.py", *tasks[1].NotebookTask.NotebookPath)
	assert.Len(t, downloader.files, 2)
}

func TestDownloader_MarkTasksForDownload_SingleNotebook(t *testing.T) {
	ctx := t.Context()
	w := newTestWorkspaceClient(t, notebookStatusHandler(t))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("task1"),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: new("/Users/user/project/notebook"),
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)

	// Single notebook: basePath = path.Dir => same as old behavior.
	assert.Equal(t, "../source/notebook.py", *tasks[0].NotebookTask.NotebookPath)
	assert.Len(t, downloader.files, 1)
}

func TestDownloader_MarkTasksForDownload_SparkPythonFile(t *testing.T) {
	ctx := t.Context()
	pythonFile := "/Users/user/project/etl/job.py"
	w := newTestWorkspaceClient(t, objectsHandler(t, []string{pythonFile}, nil))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir, WithSparkPythonFiles())

	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("spark_python_task"),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: new(pythonFile),
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)

	assert.Equal(t, "../source/job.py", *tasks[0].SparkPythonTask.PythonFile)
	require.Len(t, downloader.files, 1)
	f := downloader.files[filepath.Join(sourceDir, "job.py")]
	assert.Equal(t, pythonFile, f.path)
	assert.Equal(t, workspace.ExportFormatSource, f.format)
}

func TestDownloader_MarkTasksForDownload_SparkPythonFileSkipped(t *testing.T) {
	ctx := t.Context()
	// Cloud URIs and Git-sourced files are not workspace paths, so no
	// get-status/download requests should be made for them.
	w := newTestWorkspaceClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})

	downloader := NewDownloader(w, "source", "config", WithSparkPythonFiles())

	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("cloud_uri_task"),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: new("dbfs:/FileStore/job.py"),
			},
		},
		{
			TaskKey: new("git_task"),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: new("etl/job.py"),
				Source:     jobs.Source_Git,
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)
	assert.Empty(t, downloader.files)
	assert.Equal(t, "dbfs:/FileStore/job.py", *tasks[0].SparkPythonTask.PythonFile)
	assert.Equal(t, "etl/job.py", *tasks[1].SparkPythonTask.PythonFile)
}

func TestDownloader_MarkTasksForDownload_SparkPythonFileDisabledByDefault(t *testing.T) {
	ctx := t.Context()
	// Without WithSparkPythonFiles, workspace files referenced by a
	// spark_python_task are left untouched, so no requests are made.
	w := newTestWorkspaceClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})

	downloader := NewDownloader(w, "source", "config")

	pythonFile := "/Users/user/project/etl/job.py"
	tasks := []jobs.TaskSettings{
		{
			TaskKey: new("spark_python_task"),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: new(pythonFile),
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)
	assert.Empty(t, downloader.files)
	assert.Equal(t, pythonFile, *tasks[0].SparkPythonTask.PythonFile)
}

func TestDownloader_MarkTasksForDownload_NoNotebooks(t *testing.T) {
	ctx := t.Context()
	w := newTestWorkspaceClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})

	downloader := NewDownloader(w, "source", "config")

	tasks := []jobs.TaskSettings{
		{TaskKey: new("spark_task")},
		{TaskKey: new("python_wheel_task")},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)
	assert.Empty(t, downloader.files)
}

func TestDownloader_FlushToDisk(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	contents := map[string]string{
		"/Users/user/project/notebook": "# Databricks notebook source\nprint(1)",
		"/Users/user/project/utils.py": "def helper(): pass",
	}
	w := newTestWorkspaceClient(t, func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/workspace/export" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		content, ok := contents[r.URL.Query().Get("path")]
		if !ok {
			t.Fatalf("unexpected export path: %s", r.URL.Query().Get("path"))
		}
		_, err := rw.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	})

	sourceDir := t.TempDir()
	downloader := NewDownloader(w, sourceDir, "config")
	downloader.files[filepath.Join(sourceDir, "notebook.py")] = exportFile{
		path:   "/Users/user/project/notebook",
		format: workspace.ExportFormatSource,
	}
	downloader.files[filepath.Join(sourceDir, "utils.py")] = exportFile{
		path:   "/Users/user/project/utils.py",
		format: workspace.ExportFormatSource,
	}

	err := downloader.FlushToDisk(ctx, false)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(sourceDir, "notebook.py"))
	require.NoError(t, err)
	assert.Equal(t, contents["/Users/user/project/notebook"], string(data))

	data, err = os.ReadFile(filepath.Join(sourceDir, "utils.py"))
	require.NoError(t, err)
	assert.Equal(t, contents["/Users/user/project/utils.py"], string(data))
}

type fakeWriteCloser struct {
	bytes.Buffer
	closeErr error
}

func (f *fakeWriteCloser) Close() error { return f.closeErr }

func TestWriteAndClose(t *testing.T) {
	closeErr := errors.New("close failed")
	readErr := errors.New("read failed")

	tests := []struct {
		name     string
		src      io.Reader
		closeErr error
		wantErr  error
		wantData string
	}{
		{
			name:     "success",
			src:      strings.NewReader("data"),
			wantData: "data",
		},
		{
			name:     "close error is returned",
			src:      strings.NewReader("data"),
			closeErr: closeErr,
			wantErr:  closeErr,
			wantData: "data",
		},
		{
			name:     "copy error takes precedence over close error",
			src:      iotest.ErrReader(readErr),
			closeErr: closeErr,
			wantErr:  readErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &fakeWriteCloser{closeErr: tt.closeErr}
			err := writeAndClose(dst, tt.src)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.wantData, dst.String())
		})
	}
}

func TestDownloader_MarkDirectoryForDownload_NotebookGetsExtension(t *testing.T) {
	ctx := t.Context()

	rootPath := "/pipeline/root"
	notebookPath := "/pipeline/root/ExploratoryNotebook"
	filePath := "/pipeline/root/utils.py"

	type listResponse struct {
		Objects []workspace.ObjectInfo `json:"objects"`
	}

	w := newTestWorkspaceClient(t, func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/2.0/workspace/list":
			err := json.NewEncoder(rw).Encode(listResponse{
				Objects: []workspace.ObjectInfo{
					{Path: notebookPath, ObjectType: workspace.ObjectTypeNotebook},
					{Path: filePath, ObjectType: workspace.ObjectTypeFile},
				},
			})
			assert.NoError(t, err)
		case "/api/2.0/workspace/get-status":
			path := r.URL.Query().Get("path")
			switch path {
			case rootPath:
				err := json.NewEncoder(rw).Encode(workspace.ObjectInfo{Path: rootPath})
				assert.NoError(t, err)
			case notebookPath:
				err := json.NewEncoder(rw).Encode(workspaceStatus{
					Language:     workspace.LanguagePython,
					ObjectType:   workspace.ObjectTypeNotebook,
					ExportFormat: workspace.ExportFormatSource,
				})
				assert.NoError(t, err)
			case filePath:
				err := json.NewEncoder(rw).Encode(workspace.ObjectInfo{Path: filePath})
				assert.NoError(t, err)
			default:
				t.Fatalf("unexpected get-status path: %s", path)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	err := downloader.MarkDirectoryForDownload(ctx, &rootPath)
	require.NoError(t, err)

	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "ExploratoryNotebook.py"))
	assert.NotContains(t, downloader.files, filepath.Join(sourceDir, "ExploratoryNotebook"))
	assert.Contains(t, downloader.files, filepath.Join(sourceDir, "utils.py"))
	assert.Len(t, downloader.files, 2)
}

func TestDownloader_CleanupOldFiles(t *testing.T) {
	ctx := t.Context()
	sourceDir := t.TempDir()

	oldExtract := filepath.Join(sourceDir, "extract.py")
	oldDashboard := filepath.Join(sourceDir, "dashboard.py")
	unrelated := filepath.Join(sourceDir, "utils.py")
	require.NoError(t, os.WriteFile(oldExtract, []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(oldDashboard, []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(unrelated, []byte("keep"), 0o644))

	downloader := NewDownloader(nil, sourceDir, "config")
	downloader.files[filepath.Join(sourceDir, "etl", "extract.py")] = exportFile{}
	downloader.files[filepath.Join(sourceDir, "reporting", "dashboard.py")] = exportFile{}

	downloader.CleanupOldFiles(ctx)

	assert.NoFileExists(t, oldExtract)
	assert.NoFileExists(t, oldDashboard)
	assert.FileExists(t, unrelated)
}
