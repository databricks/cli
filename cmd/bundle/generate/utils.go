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

func saveConfigToFile(ctx context.Context, data any, filename string) error {
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

func downloadNotebookAndReplaceTaskPath(ctx context.Context, task *jobs.Task, w *databricks.WorkspaceClient, outputDir string) error {
	if task.NotebookTask == nil {
		return nil
	}

	info, err := w.Workspace.GetStatusByPath(ctx, task.NotebookTask.NotebookPath)
	if err != nil {
		return err
	}

	ext := notebook.GetExtensionByLanguage(info)

	reader, err := w.Workspace.Download(ctx, task.NotebookTask.NotebookPath)
	if err != nil {
		return err
	}

	filename := path.Base(task.NotebookTask.NotebookPath) + ext
	targetPath := filepath.Join(outputDir, filename)
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}

	task.NotebookTask.NotebookPath = strings.Join([]string{".", filename}, string(filepath.Separator))

	cmdio.LogString(ctx, fmt.Sprintf("Notebook successfully saved to %s", targetPath))
	return nil
}
