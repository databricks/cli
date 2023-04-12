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

// rewritePath converts a given relative path to a stable remote workspace path.
//
// It takes these arguments:
//   - The argument `dir` is the directory relative to which the given relative path is.
//   - The given relative path is both passed and written back through `*p`.
//   - The argument `fn` is a function that performs the actual rewriting logic.
//     This logic is different between regular files or notebooks.
//
// The function returns an error if it is impossible to rewrite the given relative path.
func (m *translatePaths) rewritePath(
	dir string,
	b *bundle.Bundle,
	p *string,
	fn func(literal, localPath, remotePath string) (string, error),
) error {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(filepath.ToSlash(*p)) {
		return nil
	}

	// Local path is relative to the directory the resource was defined in.
	localPath := filepath.Join(dir, filepath.FromSlash(*p))
	if interp, ok := m.seen[localPath]; ok {
		*p = interp
		return nil
	}

	// Remote path must be relative to the bundle root.
	remotePath, err := filepath.Rel(b.Config.Path, localPath)
	if err != nil {
		return err
	}
	if strings.HasPrefix(remotePath, "..") {
		return fmt.Errorf("path %s is not contained in bundle root path", localPath)
	}

	// Prefix remote path with its remote root path.
	remotePath = path.Join(b.Config.Workspace.FilesPath, filepath.ToSlash(remotePath))

	// Convert local path into workspace path via specified function.
	interp, err := fn(*p, localPath, filepath.ToSlash(remotePath))
	if err != nil {
		return err
	}

	*p = interp
	m.seen[localPath] = interp
	return nil
}

func (m *translatePaths) translateNotebookPath(literal, localPath, remotePath string) (string, error) {
	nb, _, err := notebook.Detect(localPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("notebook %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a notebook: %w", localPath, err)
	}
	if !nb {
		return "", fmt.Errorf("file at %s is not a notebook", localPath)
	}

	// Upon import, notebooks are stripped of their extension.
	return strings.TrimSuffix(remotePath, filepath.Ext(localPath)), nil
}

func (m *translatePaths) translateFilePath(literal, localPath, remotePath string) (string, error) {
	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to access %s: %w", localPath, err)
	}

	return remotePath, nil
}

func (m *translatePaths) translateJobTask(dir string, b *bundle.Bundle, task *jobs.JobTaskSettings) error {
	var err error

	if task.NotebookTask != nil {
		err = m.rewritePath(dir, b, &task.NotebookTask.NotebookPath, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if task.SparkPythonTask != nil {
		err = m.rewritePath(dir, b, &task.SparkPythonTask.PythonFile, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) translatePipelineLibrary(dir string, b *bundle.Bundle, library *pipelines.PipelineLibrary) error {
	var err error

	if library.Notebook != nil {
		err = m.rewritePath(dir, b, &library.Notebook.Path, m.translateNotebookPath)
		if err != nil {
			return err
		}
	}

	if library.File != nil {
		err = m.rewritePath(dir, b, &library.File.Path, m.translateFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *translatePaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)

	for key, job := range b.Config.Resources.Jobs {
		dir, err := job.ConfigFileDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
		}

		for i := 0; i < len(job.Tasks); i++ {
			err := m.translateJobTask(dir, b, &job.Tasks[i])
			if err != nil {
				return nil, err
			}
		}
	}

	for key, pipeline := range b.Config.Resources.Pipelines {
		dir, err := pipeline.ConfigFileDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
		}

		for i := 0; i < len(pipeline.Libraries); i++ {
			err := m.translatePipelineLibrary(dir, b, &pipeline.Libraries[i])
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
