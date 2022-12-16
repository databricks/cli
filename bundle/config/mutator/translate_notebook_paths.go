package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
)

type translateNotebookPaths struct {
	seen map[string]string
}

// TranslateNotebookPaths converts paths to local notebook files into references to artifacts.
func TranslateNotebookPaths() bundle.Mutator {
	return &translateNotebookPaths{}
}

func (m *translateNotebookPaths) Name() string {
	return "TranslateNotebookPaths"
}

var nonWord = regexp.MustCompile(`[^\w]`)

func (m *translateNotebookPaths) rewritePath(b *bundle.Bundle, p *string) {
	relPath := path.Clean(*p)
	absPath := filepath.Join(b.Config.Path, relPath)

	// This is opportunistic. If we can't stat, continue.
	_, err := os.Stat(absPath)
	if err != nil {
		return
	}

	// Define artifact for this notebook.
	id := nonWord.ReplaceAllString(relPath, "_")
	if v, ok := m.seen[id]; ok {
		*p = v
		return
	}

	b.Config.Artifacts[id] = &config.Artifact{
		Notebook: &config.NotebookArtifact{
			Path: relPath,
		},
	}

	interp := fmt.Sprintf("${artifacts.%s.notebook.remote_path}", id)
	*p = interp
	m.seen[id] = interp
}

func (m *translateNotebookPaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	m.seen = make(map[string]string)

	if b.Config.Artifacts == nil {
		b.Config.Artifacts = make(map[string]*config.Artifact)
	}

	for _, job := range b.Config.Resources.Jobs {
		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			if task.NotebookTask == nil {
				continue
			}

			m.rewritePath(b, &task.NotebookTask.NotebookPath)
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			if library.Notebook == nil {
				continue
			}

			m.rewritePath(b, &library.Notebook.Path)
		}
	}

	return nil, nil
}
