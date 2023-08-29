package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type fnTemplateData func(task *jobs.Task) (map[string]any, error)
type fnCleanUp func(task *jobs.Task)
type fnTasks func(b *bundle.Bundle) []*jobs.Task

type trampoline struct {
	name         string
	getTasks     fnTasks
	templateData fnTemplateData
	cleanUp      fnCleanUp
	template     string
}

func NewTrampoline(
	name string,
	tasks fnTasks,
	templateData fnTemplateData,
	cleanUp fnCleanUp,
	template string,
) *trampoline {
	return &trampoline{name, tasks, templateData, cleanUp, template}
}

func (m *trampoline) Name() string {
	return fmt.Sprintf("trampoline(%s)", m.name)
}

func (m *trampoline) Apply(ctx context.Context, b *bundle.Bundle) error {
	tasks := m.getTasks(b)
	for _, task := range tasks {
		err := m.generateNotebookWrapper(b, task)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *trampoline) generateNotebookWrapper(b *bundle.Bundle, task *jobs.Task) error {
	internalDir, err := b.InternalDir()
	if err != nil {
		return err
	}

	notebookName := fmt.Sprintf("notebook_%s", task.TaskKey)
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

	data, err := m.templateData(task)
	if err != nil {
		return err
	}

	t, err := template.New(notebookName).Parse(m.template)
	if err != nil {
		return err
	}

	internalDirRel, err := filepath.Rel(b.Config.Path, internalDir)
	if err != nil {
		return err
	}

	m.cleanUp(task)
	remotePath := path.Join(b.Config.Workspace.FilesPath, filepath.ToSlash(internalDirRel), notebookName)

	task.NotebookTask = &jobs.NotebookTask{
		NotebookPath: remotePath,
	}

	return t.Execute(f, data)
}
