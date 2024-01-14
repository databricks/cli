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
	"gopkg.in/yaml.v3"
)

type notebookDownloader struct {
	notebooks map[string]io.ReadCloser
	w         *databricks.WorkspaceClient
	outputDir string
}

func saveConfigToFile(ctx context.Context, data any, filename string, force bool) error {
	// check that file exists
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", filename)
		}
		if !force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", filename)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = encode(data, file)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Job configuration successfully saved to %s", filename))
	return nil
}

func encode(data any, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(data)
}

func (n *notebookDownloader) DownloadInMemory(ctx context.Context, task *jobs.Task) error {
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

	// if not yet downloaded, download first
	if _, ok := n.notebooks[targetPath]; !ok {
		reader, err := n.w.Workspace.Download(ctx, task.NotebookTask.NotebookPath)
		if err != nil {
			return err
		}
		n.notebooks[targetPath] = reader
	}

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

	for targetPath, reader := range n.notebooks {
		file, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(file, reader)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Notebook successfully saved to %s", targetPath))
	}

	return nil
}

func (n *notebookDownloader) Close() error {
	for _, reader := range n.notebooks {
		err := reader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func newNotebookDownloader(w *databricks.WorkspaceClient, outputDir string) *notebookDownloader {
	return &notebookDownloader{
		notebooks: make(map[string]io.ReadCloser),
		w:         w,
		outputDir: outputDir,
	}
}
