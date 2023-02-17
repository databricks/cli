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
)

type translateNotebookPaths struct {
	seen map[string]string
}

// TranslateNotebookPaths converts paths to local notebook files into workspace paths.
func TranslateNotebookPaths() bundle.Mutator {
	return &translateNotebookPaths{}
}

func (m *translateNotebookPaths) Name() string {
	return "TranslateNotebookPaths"
}

func (m *translateNotebookPaths) rewritePath(b *bundle.Bundle, p *string) error {
	relPath := path.Clean(*p)
	if interp, ok := m.seen[relPath]; ok {
		*p = interp
		return nil
	}

	absPath := filepath.Join(b.Config.Path, relPath)
	nb, _, err := notebook.Detect(absPath)
	if err != nil {
		// Ignore if this file doesn't exist. Maybe it's an absolute workspace path?
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to determine if %s is a notebook: %w", relPath, err)
	}

	if !nb {
		return fmt.Errorf("file at %s is not a notebook", relPath)
	}

	// Upon import, notebooks are stripped of their extension.
	withoutExt := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// We have a notebook on our hands! It will be available under the file path.
	interp := fmt.Sprintf("${workspace.file_path.workspace}/%s", withoutExt)
	*p = interp
	m.seen[relPath] = interp
	return nil
}

func (m *translateNotebookPaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)

	for _, job := range b.Config.Resources.Jobs {
		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			if task.NotebookTask == nil {
				continue
			}

			err := m.rewritePath(b, &task.NotebookTask.NotebookPath)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			if library.Notebook == nil {
				continue
			}

			err := m.rewritePath(b, &library.Notebook.Path)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
