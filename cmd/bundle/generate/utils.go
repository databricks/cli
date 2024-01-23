package generate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"golang.org/x/sync/errgroup"
)

type downloader struct {
	files     map[string]string
	w         *databricks.WorkspaceClient
	sourceDir string
	configDir string
}

func (n *downloader) MarkTaskForDownload(ctx context.Context, task *jobs.Task) error {
	if task.NotebookTask == nil {
		return nil
	}

	return n.markNotebookForDownload(ctx, &task.NotebookTask.NotebookPath)
}

func (n *downloader) MarkPipelineLibraryForDownload(ctx context.Context, lib *pipelines.PipelineLibrary) error {
	if lib.Notebook != nil {
		return n.markNotebookForDownload(ctx, &lib.Notebook.Path)
	}

	if lib.File != nil {
		return n.markFileForDownload(ctx, &lib.File.Path)
	}

	return nil
}

func (n *downloader) markFileForDownload(ctx context.Context, filePath *string) error {
	_, err := n.w.Workspace.GetStatusByPath(ctx, *filePath)
	if err != nil {
		return err
	}

	filename := path.Base(*filePath)
	targetPath := filepath.Join(n.sourceDir, filename)

	n.files[targetPath] = *filePath

	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*filePath = rel
	return nil
}

func (n *downloader) markNotebookForDownload(ctx context.Context, notebookPath *string) error {
	info, err := n.w.Workspace.GetStatusByPath(ctx, *notebookPath)
	if err != nil {
		return err
	}

	ext := notebook.GetExtensionByLanguage(info)

	filename := path.Base(*notebookPath) + ext
	targetPath := filepath.Join(n.sourceDir, filename)

	n.files[targetPath] = *notebookPath

	// Update the notebook path to be relative to the config dir
	rel, err := filepath.Rel(n.configDir, targetPath)
	if err != nil {
		return err
	}

	*notebookPath = rel
	return nil
}

func (n *downloader) FlushToDisk(ctx context.Context, force bool) error {
	err := os.MkdirAll(n.sourceDir, 0755)
	if err != nil {
		return err
	}

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
	for k, v := range n.files {
		targetPath := k
		filePath := v
		errs.Go(func() error {
			reader, err := n.w.Workspace.Download(errCtx, filePath)
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

			cmdio.LogString(errCtx, fmt.Sprintf("File successfully saved to %s", targetPath))
			return reader.Close()
		})
	}

	return errs.Wait()
}

func newDownloader(w *databricks.WorkspaceClient, sourceDir string, configDir string) *downloader {
	return &downloader{
		files:     make(map[string]string),
		w:         w,
		sourceDir: sourceDir,
		configDir: configDir,
	}
}
