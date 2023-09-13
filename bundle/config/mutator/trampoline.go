package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/databricks/cli/bundle"
	jobs_utils "github.com/databricks/cli/libs/jobs"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type TrampolineFunctions interface {
	GetTemplateData(b *bundle.Bundle, task *jobs.Task) (map[string]any, error)
	GetTasks(b *bundle.Bundle) []jobs_utils.TaskWithJobKey
	GetTemplate(b *bundle.Bundle, task *jobs.Task) (string, error)
	CleanUp(task *jobs.Task) error
}
type trampoline struct {
	name      string
	functions TrampolineFunctions
}

func NewTrampoline(
	name string,
	functions TrampolineFunctions,
) *trampoline {
	return &trampoline{name, functions}
}

func (m *trampoline) Name() string {
	return fmt.Sprintf("trampoline(%s)", m.name)
}

func (m *trampoline) Apply(ctx context.Context, b *bundle.Bundle) error {
	tasks := m.functions.GetTasks(b)
	for _, task := range tasks {
		err := m.generateNotebookWrapper(b, task)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *trampoline) generateNotebookWrapper(b *bundle.Bundle, task jobs_utils.TaskWithJobKey) error {
	internalDir, err := b.InternalDir()
	if err != nil {
		return err
	}

	notebookName := fmt.Sprintf("notebook_%s_%s_%s", m.name, task.JobKey, task.Task.TaskKey)
	localNotebookPath := filepath.Join(internalDir, notebookName+".py")

	err = os.MkdirAll(filepath.Dir(localNotebookPath), 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(localNotebookPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := m.functions.GetTemplateData(b, task.Task)
	if err != nil {
		return err
	}

	templateString, err := m.functions.GetTemplate(b, task.Task)
	if err != nil {
		return err
	}
	t, err := template.New(notebookName).Parse(templateString)
	if err != nil {
		return err
	}

	internalDirRel, err := filepath.Rel(b.Config.Path, internalDir)
	if err != nil {
		return err
	}

	err = m.functions.CleanUp(task.Task)
	if err != nil {
		return err
	}
	remotePath := path.Join(b.Config.Workspace.FilesPath, filepath.ToSlash(internalDirRel), notebookName)

	task.Task.NotebookTask = &jobs.NotebookTask{
		NotebookPath: remotePath,
	}

	return t.Execute(f, data)
}
