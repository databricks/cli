package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/notebook"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type translatePaths struct {
	seen     map[string]string
	filePath string
}

// TranslatePaths converts paths to local notebook files into paths in the workspace file system.
func TranslatePaths() bundle.Mutator {
	return &translatePaths{}
}

func (m *translatePaths) Name() string {
	return "TranslatePaths"
}

func (m *translatePaths) rewritePath(b *bundle.Bundle, p *string, fn func(literal, relPath, absPath string) (string, error)) error {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(*p) {
		return nil
	}

	// Reuse value if this path has been rewritten before.
	relPath := path.Clean(*p)
	if interp, ok := m.seen[relPath]; ok {
		*p = interp
		return nil
	}

	// Convert local path into workspace path via specified function.
	absPath := filepath.Join(b.Config.Path, relPath)
	interp, err := fn(*p, relPath, absPath)
	if err != nil {
		return err
	}

	*p = interp
	m.seen[relPath] = interp
	return nil
}

func (m *translatePaths) translateNotebookPath(literal, relPath, absPath string) (string, error) {
	nb, _, err := notebook.Detect(absPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("notebook %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a notebook: %w", relPath, err)
	}
	if !nb {
		return "", fmt.Errorf("file at %s is not a notebook", relPath)
	}

	// Upon import, notebooks are stripped of their extension.
	withoutExt := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// We have a notebook on our hands! It will be available under the file path.
	return path.Join(m.filePath, withoutExt), nil
}

func (m *translatePaths) translateFilePath(literal, relPath, absPath string) (string, error) {
	_, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to access %s: %w", relPath, err)
	}

	// The file will be available under the file path.
	return path.Join(m.filePath, relPath), nil
}

func (m *translatePaths) translateJobTask(b *bundle.Bundle, task *jobs.JobTaskSettings) error {
	var err error

	if task.NotebookTask != nil {
		err = m.rewritePath(b, &task.NotebookTask.NotebookPath, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if task.SparkPythonTask != nil {
		err = m.rewritePath(b, &task.SparkPythonTask.PythonFile, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) translatePipelineLibrary(b *bundle.Bundle, library *pipelines.PipelineLibrary) error {
	var err error

	if library.Notebook != nil {
		err = m.rewritePath(b, &library.Notebook.Path, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if library.File != nil {
		err = m.rewritePath(b, &library.File.Path, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)
	m.filePath = b.Config.Workspace.FilePath

	for _, job := range b.Config.Resources.Jobs {
		for i := 0; i < len(job.Tasks); i++ {
			err := m.translateJobTask(b, &job.Tasks[i])
			if err != nil {
				return nil, err
			}
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			err := m.translatePipelineLibrary(b, &pipeline.Libraries[i])
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
