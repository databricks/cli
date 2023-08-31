package python

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// go:embed trampoline_data/notebook.py
var notebookTrampolineData string

// go:embed trampoline_data/python.py
var pyTrampolineData string

func TransforNotebookTask() bundle.Mutator {
	return mutator.NewTrampoline(
		"python_notebook",
		&notebookTrampoline{},
		getTemplate,
	)
}

type notebookTrampoline struct{}

func (n *notebookTrampoline) GetTasks(b *bundle.Bundle) []mutator.TaskWithJobKey {
	return mutator.GetTasksWithJobKeyBy(b, func(task *jobs.Task) bool {
		return task.NotebookTask != nil &&
			task.NotebookTask.Source == jobs.SourceWorkspace &&
			(strings.HasSuffix(task.NotebookTask.NotebookPath, ".ipynb") ||
				strings.HasSuffix(task.NotebookTask.NotebookPath, ".py"))
	})
}

func (n *notebookTrampoline) CleanUp(task *jobs.Task) error {
	return nil
}

func getTemplate(task *jobs.Task) (string, error) {
	if task.NotebookTask == nil {
		return "", fmt.Errorf("nil notebook path")
	}

	if task.NotebookTask.Source != jobs.SourceWorkspace {
		return "", fmt.Errorf("source must be workspace")
	}

	bytesData, err := os.ReadFile(task.NotebookTask.NotebookPath)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(bytesData))
	if strings.HasSuffix(task.NotebookTask.NotebookPath, ".ipynb") {
		return getIpynbTemplate(s)
	}

	lines := strings.Split(s, "\n")
	if strings.HasPrefix(lines[0], "# Databricks notebook source") {
		return getDbnbTemplate(strings.Join(lines[1:], "\n"))
	}

	//TODO return getPyTemplate(s), nil
	return s, nil
}

func getDbnbTemplate(s string) (string, error) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "# Databricks notebook source"))
	return fmt.Sprintf(`# Databricks notebook source
%s
# Command ----------
%s
`, notebookTrampolineData, s), nil
}

func getIpynbTemplate(s string) (string, error) {
	var data map[string]any
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return "", err
	}

	if data["cells"] == nil {
		data["cells"] = []any{}
	}

	data["cells"] = append([]any{
		map[string]any{
			"cell_type": "code",
			"source":    []string{notebookTrampolineData},
		},
	}, data["cells"].([]any)...)

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (n *notebookTrampoline) GetTemplateData(b *bundle.Bundle, task *jobs.Task) (map[string]any, error) {
	return map[string]any{
		"ProjectRoot": b.Config.Workspace.FilesPath,
		"SourceFile":  task.NotebookTask.NotebookPath,
	}, nil
}
