package python

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

//go:embed trampoline_data/notebook.py
var notebookTrampolineData string

//go:embed trampoline_data/python.py
var pyTrampolineData string

func TransforNotebookTask() bundle.Mutator {
	return mutator.NewTrampoline(
		"python_notebook",
		&notebookTrampoline{},
	)
}

type notebookTrampoline struct{}

func localNotebookPath(b *bundle.Bundle, task *jobs.Task) (string, error) {
	remotePath := task.NotebookTask.NotebookPath
	relRemotePath, err := filepath.Rel(b.Config.Workspace.FilesPath, remotePath)
	if err != nil {
		return "", err
	}
	localPath := filepath.Join(b.Config.Path, filepath.FromSlash(relRemotePath))
	_, err = os.Stat(fmt.Sprintf("%s.ipynb", localPath))
	if err == nil {
		return fmt.Sprintf("%s.ipynb", localPath), nil
	}

	_, err = os.Stat(fmt.Sprintf("%s.py", localPath))
	if err == nil {
		return fmt.Sprintf("%s.py", localPath), nil
	}
	return "", fmt.Errorf("notebook %s not found", localPath)
}

func (n *notebookTrampoline) GetTasks(b *bundle.Bundle) []mutator.TaskWithJobKey {
	return mutator.GetTasksWithJobKeyBy(b, func(task *jobs.Task) bool {
		if task.NotebookTask == nil ||
			task.NotebookTask.Source == jobs.SourceGit {
			return false
		}
		localPath, err := localNotebookPath(b, task)
		if err != nil {
			return false
		}
		return strings.HasSuffix(localPath, ".ipynb") || strings.HasSuffix(localPath, ".py")
	})
}

func (n *notebookTrampoline) CleanUp(task *jobs.Task) error {
	return nil
}

func (n *notebookTrampoline) GetTemplate(b *bundle.Bundle, task *jobs.Task) (string, error) {
	if task.NotebookTask == nil {
		return "", fmt.Errorf("nil notebook path")
	}

	if task.NotebookTask.Source == jobs.SourceGit {
		return "", fmt.Errorf("source must be workspace %s", task.NotebookTask.Source)
	}
	localPath, err := localNotebookPath(b, task)
	if err != nil {
		return "", err
	}

	bytesData, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(bytesData))
	if strings.HasSuffix(localPath, ".ipynb") {
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
