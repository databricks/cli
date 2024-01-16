package generate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

type notebookDownloader struct {
	notebooks map[string]string
	w         *databricks.WorkspaceClient
	outputDir string
}

func (n *notebookDownloader) MarkForDownload(ctx context.Context, task *jobs.Task) error {
	if task.NotebookTask == nil {
		return nil
	}

	info, err := n.w.Workspace.GetStatusByPath(ctx, task.NotebookTask.NotebookPath)
	if err != nil {
		return err
	}

	ext := notebook.GetExtensionByLanguage(info)

	filename := path.Base(task.NotebookTask.NotebookPath) + ext
	targetPath := filepath.Join(n.outputDir, filename)

	n.notebooks[targetPath] = task.NotebookTask.NotebookPath

	task.NotebookTask.NotebookPath = strings.Join([]string{".", filename}, string(filepath.Separator))
	return nil
}

func (n *notebookDownloader) FlushToDisk(ctx context.Context, force bool) error {
	err := os.MkdirAll(n.outputDir, 0755)
	if err != nil {
		return err
	}

	// First check that all files can be written
	for targetPath := range n.notebooks {
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
	for k, v := range n.notebooks {
		targetPath := k
		notebookPath := v
		errs.Go(func() error {
			reader, err := n.w.Workspace.Download(ctx, notebookPath)
			if err != nil {
				return err
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(file, reader)
			if err != nil {
				return err
			}

			cmdio.LogString(errCtx, fmt.Sprintf("Notebook successfully saved to %s", targetPath))
			return reader.Close()
		})
	}

	return errs.Wait()
}

func newNotebookDownloader(w *databricks.WorkspaceClient, outputDir string) *notebookDownloader {
	return &notebookDownloader{
		notebooks: make(map[string]string),
		w:         w,
		outputDir: outputDir,
	}
}
