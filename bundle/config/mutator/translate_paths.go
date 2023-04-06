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
	seen map[string]string
}

// TranslatePaths converts paths to local notebook files into paths in the workspace file system.
func TranslatePaths() bundle.Mutator {
	return &translatePaths{}
}

func (m *translatePaths) Name() string {
	return "TranslatePaths"
}

func (m *translatePaths) rewritePath(
	relativeTo string,
	b *bundle.Bundle,
	p *string,
	fn func(b *bundle.Bundle, literal, relative string) (string, error),
) error {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(filepath.ToSlash(*p)) {
		return nil
	}

	// Reuse value if this path has been rewritten before.
	relativePath := path.Join(relativeTo, filepath.ToSlash(*p))
	if interp, ok := m.seen[relativePath]; ok {
		*p = interp
		return nil
	}

	// Convert local path into workspace path via specified function.
	interp, err := fn(b, *p, relativePath)
	if err != nil {
		return err
	}

	*p = interp
	m.seen[relativePath] = interp
	return nil
}

func (m *translatePaths) translateNotebookPath(b *bundle.Bundle, literalPath, relativePath string) (string, error) {
	nb, _, err := notebook.Detect(filepath.Join(b.Config.Path, relativePath))
	if os.IsNotExist(err) {
		return "", fmt.Errorf("notebook %s not found", literalPath)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a notebook: %w", relativePath, err)
	}
	if !nb {
		return "", fmt.Errorf("file at %s is not a notebook", relativePath)
	}

	// Upon import, notebooks are stripped of their extension.
	withoutExt := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

	// We have a notebook on our hands! It will be available under the file path.
	return path.Join(b.Config.Workspace.FilePath.Workspace, withoutExt), nil
}

func (m *translatePaths) translateFilePath(b *bundle.Bundle, literalPath, relativePath string) (string, error) {
	_, err := os.Stat(filepath.Join(b.Config.Path, relativePath))
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file %s not found", literalPath)
	}
	if err != nil {
		return "", fmt.Errorf("unable to access %s: %w", relativePath, err)
	}

	// The file will be available under the file path.
	return path.Join(b.Config.Workspace.FilePath.Workspace, relativePath), nil
}

func (m *translatePaths) translateJobTask(relativeTo string, b *bundle.Bundle, task *jobs.JobTaskSettings) error {
	var err error

	if task.NotebookTask != nil {
		err = m.rewritePath(relativeTo, b, &task.NotebookTask.NotebookPath, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if task.SparkPythonTask != nil {
		err = m.rewritePath(relativeTo, b, &task.SparkPythonTask.PythonFile, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) translatePipelineLibrary(relativeTo string, b *bundle.Bundle, library *pipelines.PipelineLibrary) error {
	var err error

	if library.Notebook != nil {
		err = m.rewritePath(relativeTo, b, &library.Notebook.Path, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if library.File != nil {
		err = m.rewritePath(relativeTo, b, &library.File.Path, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)

	for _, job := range b.Config.Resources.Jobs {
		for i := 0; i < len(job.Tasks); i++ {
			err := m.translateJobTask(job.ConfigFileDirectory(), b, &job.Tasks[i])
			if err != nil {
				return nil, err
			}
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			err := m.translatePipelineLibrary(pipeline.ConfigFileDirectory(), b, &pipeline.Libraries[i])
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
