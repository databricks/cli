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

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/sync/errgroup"

	"github.com/databricks/databricks-sdk-go/client"
)

type exportFile struct {
	path   string
	format workspace.ExportFormat
}

type Downloader struct {
	files     map[string]exportFile
	w         *databricks.WorkspaceClient
	sourceDir string
	configDir string
	basePath  string
}

func (n *Downloader) MarkTaskForDownload(ctx context.Context, task *jobs.Task) error {
	if task.NotebookTask == nil {
		return nil
	}

	return n.markNotebookForDownload(ctx, &task.NotebookTask.NotebookPath)
}

func (n *Downloader) MarkPipelineLibraryForDownload(ctx context.Context, lib *pipelines.PipelineLibrary) error {
	if lib.Notebook != nil {
		return n.markNotebookForDownload(ctx, &lib.Notebook.Path)
	}

	if lib.File != nil {
		return n.markFileForDownload(ctx, &lib.File.Path)
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
		format: workspace.ExportFormatSource,
	}

	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*filePath = rel
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

	objects, err := n.w.Workspace.RecursiveList(ctx, *dirPath)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if obj.ObjectType == workspace.ObjectTypeDirectory {
			continue
		}

		err := n.markFileForDownload(ctx, &obj.Path)
		if err != nil {
			return err
		}
	}

	rel, err := filepath.Rel(n.configDir, n.sourceDir)
	if err != nil {
		return err
	}

	*dirPath = rel
	return nil
}

type workspaceStatus struct {
	Language     workspace.Language     `json:"language,omitempty"`
	ObjectType   workspace.ObjectType   `json:"object_type,omitempty"`
	ExportFormat workspace.ExportFormat `json:"repos_export_format,omitempty"`
}

func (n *Downloader) markNotebookForDownload(ctx context.Context, notebookPath *string) error {
	apiClient, err := client.New(n.w.Config)
	if err != nil {
		return err
	}

	var stat workspaceStatus
	err = apiClient.Do(
		ctx,
		http.MethodGet,
		"/api/2.0/workspace/get-status",
		nil,
		nil,
		map[string]string{
			"path":               *notebookPath,
			"return_export_info": "true",
		},
		&stat,
	)

	relPath := n.relativePath(*notebookPath)
	// If the path has any extension, strip it
	ext := path.Ext(relPath)
	if ext != "" {
		relPath = strings.TrimSuffix(relPath, ext)
	}

	ext = notebook.GetExtensionByLanguage(&workspace.ObjectInfo{
		Language:   stat.Language,
		ObjectType: stat.ObjectType,
	})

	if stat.ExportFormat == workspace.ExportFormatJupyter {
		ext = ".ipynb"
	}

	relPath = relPath + ext
	targetPath := filepath.Join(n.sourceDir, relPath)

	n.files[targetPath] = exportFile{
		path:   *notebookPath,
		format: stat.ExportFormat,
	}

	// Update the notebook path to be relative to the config dir
	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*notebookPath = rel
	return nil
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
				return fmt.Errorf("%s is a directory", targetPath)
			}
			if !force {
				return fmt.Errorf("%s already exists. Use --force to overwrite", targetPath)
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
			reader, err := n.w.Workspace.Download(errCtx, exportFile.path, workspace.DownloadFormat(exportFile.format))
			if err != nil {
				return err
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, reader)
			if err != nil {
				return err
			}

			cmdio.LogString(errCtx, "File successfully saved to "+targetPath)
			return reader.Close()
		})
	}

	return errs.Wait()
}

func NewDownloader(w *databricks.WorkspaceClient, sourceDir, configDir string) *Downloader {
	return &Downloader{
		files:     make(map[string]exportFile),
		w:         w,
		sourceDir: sourceDir,
		configDir: configDir,
	}
}
