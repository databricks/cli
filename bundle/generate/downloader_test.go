package generate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_MarkFileReturnsRelativePath(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir/doesnt/matter"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	var err error

	// Test that the path is normalized to be relative to the config directory.
	f1 := "/a/b/c"
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f1).Return(&workspace.ObjectInfo{
		Path: f1,
	}, nil)
	err = downloader.markFileForDownload(ctx, &f1)
	require.NoError(t, err)
	assert.Equal(t, "../source/c", f1)

	// Test that the previous path doesn't influence the next path.
	f2 := "/a/b/c/d"
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(ctx, f2).Return(&workspace.ObjectInfo{
		Path: f2,
	}, nil)
	err = downloader.markFileForDownload(ctx, &f2)
	require.NoError(t, err)
	assert.Equal(t, "../source/d", f2)
}

func TestDownloader_DoesNotRecurseIntoNodeModules(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(m.WorkspaceClient, sourceDir, configDir)

	rootPath := "/workspace/app"

	// Mock the root directory listing
	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, rootPath).
		Return(&workspace.ObjectInfo{Path: rootPath}, nil)

	// Root directory contains: app.py, src/, node_modules/
	m.GetMockWorkspaceAPI().EXPECT().
		ListAll(ctx, workspace.ListWorkspaceRequest{Path: rootPath}).
		Return([]workspace.ObjectInfo{
			{Path: "/workspace/app/app.py", ObjectType: workspace.ObjectTypeFile},
			{Path: "/workspace/app/src", ObjectType: workspace.ObjectTypeDirectory},
			{Path: "/workspace/app/node_modules", ObjectType: workspace.ObjectTypeDirectory},
		}, nil)

	// src/ directory contains: index.js
	m.GetMockWorkspaceAPI().EXPECT().
		ListAll(ctx, workspace.ListWorkspaceRequest{Path: "/workspace/app/src"}).
		Return([]workspace.ObjectInfo{
			{Path: "/workspace/app/src/index.js", ObjectType: workspace.ObjectTypeFile},
		}, nil)

	// We should NOT list node_modules directory - this is the key assertion
	// If this expectation is not met, the test will fail

	// Mock file downloads to make markFileForDownload work
	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, "/workspace/app/app.py").
		Return(&workspace.ObjectInfo{Path: "/workspace/app/app.py"}, nil)

	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(ctx, "/workspace/app/src/index.js").
		Return(&workspace.ObjectInfo{Path: "/workspace/app/src/index.js"}, nil)

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

func newTestWorkspaceClient(t *testing.T, handler http.HandlerFunc) *databricks.WorkspaceClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			http.NotFound(w, r)
			return
		}

		handler(w, r)
	}))
	t.Cleanup(server.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "test-token",
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

func TestDownloader_MarkTasksForDownload_PreservesStructure(t *testing.T) {
	w := newTestWorkspaceClient(t, notebookStatusHandler(t))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	tasks := []jobs.Task{
		{
			TaskKey: "extract_task",
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: "/Users/user/project/etl/extract",
			},
		},
		{
			TaskKey: "dashboard_task",
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: "/Users/user/project/reporting/dashboard",
			},
		},
	}

	err := downloader.MarkTasksForDownload(t.Context(), tasks)
	require.NoError(t, err)

	assert.Equal(t, "../source/etl/extract.py", tasks[0].NotebookTask.NotebookPath)
	assert.Equal(t, "../source/reporting/dashboard.py", tasks[1].NotebookTask.NotebookPath)
	assert.Len(t, downloader.files, 2)
}

func TestDownloader_MarkTasksForDownload_SingleNotebook(t *testing.T) {
	ctx := t.Context()
	w := newTestWorkspaceClient(t, notebookStatusHandler(t))

	dir := "base/dir"
	sourceDir := filepath.Join(dir, "source")
	configDir := filepath.Join(dir, "config")
	downloader := NewDownloader(w, sourceDir, configDir)

	tasks := []jobs.Task{
		{
			TaskKey: "task1",
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: "/Users/user/project/notebook",
			},
		},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)

	// Single notebook: basePath = path.Dir => same as old behavior.
	assert.Equal(t, "../source/notebook.py", tasks[0].NotebookTask.NotebookPath)
	assert.Len(t, downloader.files, 1)
}

func TestDownloader_MarkTasksForDownload_NoNotebooks(t *testing.T) {
	ctx := t.Context()
	w := newTestWorkspaceClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})

	downloader := NewDownloader(w, "source", "config")

	tasks := []jobs.Task{
		{TaskKey: "spark_task"},
		{TaskKey: "python_wheel_task"},
	}

	err := downloader.MarkTasksForDownload(ctx, tasks)
	require.NoError(t, err)
	assert.Empty(t, downloader.files)
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
