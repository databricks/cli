package mutator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type TaskWithJobKey struct {
	Task   *jobs.Task
	JobKey string
}

type TrampolineFunctions interface {
	GetTemplateData(b *bundle.Bundle, task *jobs.Task) (map[string]any, error)
	GetTasks(b *bundle.Bundle) []TaskWithJobKey
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

func GetTasksWithJobKeyBy(b *bundle.Bundle, filter func(*jobs.Task) bool) []TaskWithJobKey {
	tasks := make([]TaskWithJobKey, 0)
	for k := range b.Config.Resources.Jobs {
		for i := range b.Config.Resources.Jobs[k].Tasks {
			t := &b.Config.Resources.Jobs[k].Tasks[i]
			if filter(t) {
				tasks = append(tasks, TaskWithJobKey{
					JobKey: k,
					Task:   t,
				})
			}
		}
	}
	return tasks
}

func (m *trampoline) Name() string {
	return fmt.Sprintf("trampoline(%s)", m.name)
}

func (m *trampoline) Apply(ctx context.Context, b *bundle.Bundle) error {
	tasks := m.functions.GetTasks(b)
	for _, task := range tasks {
		log.Default().Printf("%s, %s task", task.Task.TaskKey, task.Task.NotebookTask.NotebookPath)
		err := m.generateNotebookWrapper(b, task)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *trampoline) generateNotebookWrapper(b *bundle.Bundle, task TaskWithJobKey) error {
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
