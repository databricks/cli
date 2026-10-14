package generate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/workspaceapi"
	"golang.org/x/sync/errgroup"
)

type exportFile struct {
	path   string
	format workspaceapi.ExportFormat
}

type Downloader struct {
	files     map[string]exportFile
	w         *sdkclients.Clients
	sourceDir string
	configDir string
	basePath  string

	// downloadSparkPythonFiles controls whether workspace files referenced by a
	// spark_python_task are downloaded. It is opt-in because a Python file often
	// imports sibling files that are not captured, so downloading only the entry
	// point can produce a job that fails at runtime with missing imports.
	downloadSparkPythonFiles bool
}

// DownloaderOption configures a Downloader.
type DownloaderOption func(*Downloader)

// WithSparkPythonFiles enables downloading workspace files referenced by a
// spark_python_task.
func WithSparkPythonFiles() DownloaderOption {
	return func(n *Downloader) {
		n.downloadSparkPythonFiles = true
	}
}

func (n *Downloader) MarkTaskForDownload(ctx context.Context, task *jobs.TaskSettings) error {
	if task.NotebookTask != nil && task.NotebookTask.NotebookPath != nil {
		return n.markNotebookForDownload(ctx, task.NotebookTask.NotebookPath)
	}

	if n.downloadSparkPythonFiles && task.SparkPythonTask != nil && isWorkspaceFileTask(task.SparkPythonTask.Source, utils.Deref(task.SparkPythonTask.PythonFile)) {
		return n.markFileForDownload(ctx, task.SparkPythonTask.PythonFile)
	}

	return nil
}

// isWorkspaceFileTask reports whether a task file lives in the Databricks
// workspace and should be downloaded. Files sourced from Git (source: GIT) are
// left to the Git repository, and python_file also accepts cloud URIs
// (dbfs:/, s3:/, adls:/, gcs:/) which are not workspace paths; per the Jobs API,
// workspace files are absolute and begin with "/".
func isWorkspaceFileTask(source jobs.Source, filePath string) bool {
	return source != jobs.Source_Git && strings.HasPrefix(filePath, "/")
}

func (n *Downloader) MarkPipelineLibraryForDownload(ctx context.Context, lib *pipelines.PipelineLibrary) error {
	if lib.Notebook != nil && lib.Notebook.Path != nil {
		return n.markNotebookForDownload(ctx, lib.Notebook.Path)
	}

	if lib.File != nil && lib.File.Path != nil {
		return n.markFileForDownload(ctx, lib.File.Path)
	}

	return nil
}

func (n *Downloader) markFileForDownload(ctx context.Context, filePath *string) error {
	_, err := n.w.Workspace.GetStatusByPath(ctx, *filePath)
	if err != nil {
		return err
	}

	relPath := n.relativePath(*filePath)
	targetPath := filepath.Join(n.sourceDir, relPath)

	n.files[targetPath] = exportFile{
		path:   *filePath,
		format: workspaceapi.ExportFormatSource,
	}

	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*filePath = filepath.ToSlash(rel)
	return nil
}

func (n *Downloader) MarkDirectoryForDownload(ctx context.Context, dirPath *string) error {
	_, err := n.w.Workspace.GetStatusByPath(ctx, *dirPath)
	if err != nil {
		return err
	}

	if n.basePath == "" {
		// Set the base path for relative path calculations
		n.basePath = *dirPath
	}

	objects, err := n.recursiveListWithExclusions(ctx, *dirPath)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		switch obj.ObjectType {
		case workspaceapi.ObjectTypeDirectory:
			continue
		case workspaceapi.ObjectTypeNotebook:
			if err := n.markNotebookForDownload(ctx, &obj.Path); err != nil {
				return err
			}
		default:
			if err := n.markFileForDownload(ctx, &obj.Path); err != nil {
				return err
			}
		}
	}

	rel, err := filepath.Rel(n.configDir, n.sourceDir)
	if err != nil {
		return err
	}

	*dirPath = filepath.ToSlash(rel)
	return nil
}

// recursiveListWithExclusions recursively lists all files in a directory,
// but skips recursing into directories that should be excluded (like node_modules).
func (n *Downloader) recursiveListWithExclusions(ctx context.Context, dirPath string) ([]workspaceapi.ObjectInfo, error) {
	var result []workspaceapi.ObjectInfo

	objects, err := n.w.Workspace.ListAll(ctx, workspaceapi.ListWorkspaceRequest{
		Path: dirPath,
	})
	if err != nil {
		return nil, err
	}

	for _, obj := range objects {
		if obj.ObjectType == workspaceapi.ObjectTypeDirectory {
			if path.Base(obj.Path) == "node_modules" {
				continue
			}

			subObjects, err := n.recursiveListWithExclusions(ctx, obj.Path)
			if err != nil {
				return nil, err
			}
			result = append(result, subObjects...)
		} else {
			result = append(result, obj)
		}
	}

	return result, nil
}

type workspaceStatus struct {
	Language     workspaceapi.Language     `json:"language,omitempty"`
	ObjectType   workspaceapi.ObjectType   `json:"object_type,omitempty"`
	ExportFormat workspaceapi.ExportFormat `json:"repos_export_format,omitempty"`
}

func (n *Downloader) markNotebookForDownload(ctx context.Context, notebookPath *string) error {
	apiClient, err := apiclient.New(n.w.Config)
	if err != nil {
		return err
	}

	var stat workspaceStatus
	err = apiClient.Do(
		ctx,
		http.MethodGet,
		"/api/2.0/workspace/get-status",
		auth.WorkspaceIDHeaders(n.w.Config),
		nil,
		map[string]string{
			"path":               *notebookPath,
			"return_export_info": "true",
		},
		&stat,
	)
	if err != nil {
		return err
	}

	relPath := n.relativePath(*notebookPath)

	relPath = notebook.StripExtension(relPath)

	format := stat.ExportFormat
	if fixed, ok := notebook.FixedExportFormat(stat.ObjectType); ok {
		// These object types carry their full extension in the workspace path
		// (preserved above) and report no export format, so we use a fixed one.
		format = fixed
	} else {
		ext := notebook.GetExtensionByLanguage(&workspaceapi.ObjectInfo{
			Language:   stat.Language,
			ObjectType: stat.ObjectType,
		})
		if format == workspaceapi.ExportFormatJupyter {
			ext = notebook.ExtensionJupyter
		}
		relPath += ext
	}

	targetPath := filepath.Join(n.sourceDir, relPath)

	n.files[targetPath] = exportFile{
		path:   *notebookPath,
		format: format,
	}

	// Update the notebook path to be relative to the config dir
	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*notebookPath = filepath.ToSlash(rel)
	return nil
}

func (n *Downloader) MarkTasksForDownload(ctx context.Context, tasks []jobs.TaskSettings) error {
	var paths []string
	for _, task := range tasks {
		if task.NotebookTask != nil && task.NotebookTask.NotebookPath != nil {
			paths = append(paths, *task.NotebookTask.NotebookPath)
		}
		if n.downloadSparkPythonFiles && task.SparkPythonTask != nil && isWorkspaceFileTask(task.SparkPythonTask.Source, utils.Deref(task.SparkPythonTask.PythonFile)) {
			paths = append(paths, *task.SparkPythonTask.PythonFile)
		}
	}
	if len(paths) > 0 {
		n.basePath = commonDirPrefix(paths)
	}
	for i := range tasks {
		if err := n.MarkTaskForDownload(ctx, &tasks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (n *Downloader) CleanupOldFiles(ctx context.Context) {
	for targetPath := range n.files {
		rel, err := filepath.Rel(n.sourceDir, targetPath)
		if err != nil {
			continue
		}
		if filepath.Base(rel) == rel {
			continue
		}
		oldPath := filepath.Join(n.sourceDir, filepath.Base(rel))
		if _, isNewFile := n.files[oldPath]; isNewFile {
			continue
		}
		if err := os.Remove(oldPath); err == nil {
			log.Infof(ctx, "Removed previously generated file %s", filepath.ToSlash(oldPath))
		}
	}
}

// commonDirPrefix returns the longest common directory-aligned prefix of the given paths.
func commonDirPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return path.Dir(paths[0])
	}

	prefix := paths[0]
	for _, p := range paths[1:] {
		for !strings.HasPrefix(p, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}

	// Truncate to last '/' to ensure directory alignment.
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		prefix = prefix[:i]
	} else {
		prefix = ""
	}
	return prefix
}

func (n *Downloader) relativePath(fullPath string) string {
	basePath := path.Dir(fullPath)
	if n.basePath != "" {
		basePath = n.basePath
	}

	// Remove the base path prefix
	relPath := strings.TrimPrefix(fullPath, basePath)
	if relPath[0] == '/' {
		relPath = relPath[1:]
	}

	return relPath
}

func (n *Downloader) FlushToDisk(ctx context.Context, force bool) error {
	// First check that all files can be written
	for targetPath := range n.files {
		info, err := os.Stat(targetPath)
		if err == nil {
			if info.IsDir() {
				return fmt.Errorf("%s is a directory", filepath.ToSlash(targetPath))
			}
			if !force {
				return fmt.Errorf("%s already exists. Use --force to overwrite", filepath.ToSlash(targetPath))
			}
		}
	}

	errs, errCtx := errgroup.WithContext(ctx)
	for targetPath, exportFile := range n.files {
		// Create parent directories if they don't exist
		dir := filepath.Dir(targetPath)
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
		errs.Go(func() error {
			reader, err := n.w.Workspace.Download(errCtx, exportFile.path, workspaceapi.DownloadFormat(exportFile.format))
			if err != nil {
				return err
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			err = writeAndClose(file, reader)
			if err != nil {
				return err
			}

			cmdio.LogString(errCtx, "File successfully saved to "+filepath.ToSlash(targetPath))
			return reader.Close()
		})
	}

	return errs.Wait()
}

// writeAndClose copies src into dst and closes dst. A copy error takes
// precedence; otherwise the Close error is returned because a failed Close
// can mean buffered writes were lost and the file is truncated.
func writeAndClose(dst io.WriteCloser, src io.Reader) error {
	_, err := io.Copy(dst, src)
	cerr := dst.Close()
	if err == nil {
		err = cerr
	}
	return err
}

func NewDownloader(w *sdkclients.Clients, sourceDir, configDir string, opts ...DownloaderOption) *Downloader {
	n := &Downloader{
		files:     make(map[string]exportFile),
		w:         w,
		sourceDir: sourceDir,
		configDir: configDir,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}
