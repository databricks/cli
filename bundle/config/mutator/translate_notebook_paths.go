package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/git"
	"github.com/databricks/bricks/libs/notebook"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type translateNotebookPaths struct {
	seen map[string]string

	repository *git.Repository
}

// TranslateNotebookPaths converts paths to local notebook files into paths in the workspace file system.
func TranslateNotebookPaths() bundle.Mutator {
	return &translateNotebookPaths{}
}

func (m *translateNotebookPaths) Name() string {
	return "TranslateNotebookPaths"
}

func (m *translateNotebookPaths) rewritePath(b *bundle.Bundle, relativeToRepositoryRoot bool, p *string) error {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(*p) {
		return nil
	}

	relPath := path.Clean(*p)
	if interp, ok := m.seen[relPath]; ok {
		*p = interp
		return nil
	}

	absPath := filepath.Join(b.Config.Path, relPath)
	nb, _, err := notebook.Detect(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("notebook %s not found: %w", *p, err)
	}
	if err != nil {
		return fmt.Errorf("unable to determine if %s is a notebook: %w", relPath, err)
	}
	if !nb {
		return fmt.Errorf("file at %s is not a notebook", relPath)
	}

	// Upon import, notebooks are stripped of their extension.
	// The extension is needed in both branches below.
	ext := filepath.Ext(relPath)

	// We either rewrite the path to be relative to the Git repository root
	// if the job definition uses a Git source, or we rewrite the path to
	// an absolute workspace path under the location where notebooks are uploaded.
	if relativeToRepositoryRoot {
		gitRelPath, err := filepath.Rel(m.repository.Root(), absPath)
		if err != nil {
			return fmt.Errorf("unable to derive path relative to repository root: %w", err)
		}
		*p = filepath.ToSlash(strings.TrimSuffix(gitRelPath, ext))
	} else {
		*p = "${workspace.file_path.workspace}/" + filepath.ToSlash(strings.TrimSuffix(relPath, ext))
	}

	m.seen[relPath] = *p
	return nil
}

func (m *translateNotebookPaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)

	for _, job := range b.Config.Resources.Jobs {
		// If the job has a Git source defined, rewrite paths to be relative to repository root.
		relativeToGitRepository := job.GitSource != nil && job.GitSource.GitUrl != ""
		if relativeToGitRepository && m.repository == nil {
			repository, err := b.GitRepository()
			if err != nil {
				return nil, err
			}

			m.repository = repository
		}

		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			if task.NotebookTask == nil {
				continue
			}

			err := m.rewritePath(b, relativeToGitRepository, &task.NotebookTask.NotebookPath)
			if err != nil {
				return nil, err
			}

			if relativeToGitRepository {
				task.NotebookTask.Source = jobs.NotebookTaskSourceGit
			} else {
				task.NotebookTask.Source = jobs.NotebookTaskSourceWorkspace
			}
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			if library.Notebook == nil {
				continue
			}

			err := m.rewritePath(b, false, &library.Notebook.Path)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
