package trampoline

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type TaskWithJobKey struct {
	Task   *jobs.Task
	JobKey string
}

type TrampolineFunctions interface {
	GetTemplateData(task *jobs.Task) (map[string]any, error)
	GetTasks(b *bundle.Bundle) []TaskWithJobKey
	CleanUp(task *jobs.Task) error
}

type trampoline struct {
	name      string
	functions TrampolineFunctions
	template  string
}

func NewTrampoline(
	name string,
	functions TrampolineFunctions,
	template string,
) *trampoline {
	return &trampoline{name, functions, template}
}

func (m *trampoline) Name() string {
	return fmt.Sprintf("trampoline(%s)", m.name)
}

func (m *trampoline) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tasks := m.functions.GetTasks(b)
	for _, task := range tasks {
		err := m.generateNotebookWrapper(ctx, b, task)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func (m *trampoline) generateNotebookWrapper(ctx context.Context, b *bundle.Bundle, task TaskWithJobKey) error {
	internalDir, err := b.InternalDir(ctx)
	if err != nil {
		return err
	}

	notebookName := fmt.Sprintf("notebook_%s_%s", task.JobKey, task.Task.TaskKey)
	localNotebookPath := filepath.Join(internalDir, notebookName+".py")

	err = os.MkdirAll(filepath.Dir(localNotebookPath), 0o755)
	if err != nil {
		return err
	}

	f, err := os.Create(localNotebookPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := m.functions.GetTemplateData(task.Task)
	if err != nil {
		return err
	}

	t, err := template.New(notebookName).Parse(m.template)
	if err != nil {
		return err
	}

	internalDirRel, err := filepath.Rel(b.SyncRootPath, internalDir)
	if err != nil {
		return err
	}

	err = m.functions.CleanUp(task.Task)
	if err != nil {
		return err
	}
	remotePath := path.Join(b.Config.Workspace.FilePath, filepath.ToSlash(internalDirRel), notebookName)

	task.Task.NotebookTask = &jobs.NotebookTask{
		NotebookPath: remotePath,
	}

	return t.Execute(f, data)
}
