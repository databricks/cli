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
}

// TranslateNotebookPaths converts paths to local notebook files into references to artifacts.
func TranslateNotebookPaths() bundle.Mutator {
	return &translateNotebookPaths{}
}

func (m *translateNotebookPaths) Name() string {
	return "TranslateNotebookPaths"
}

var nonWord = regexp.MustCompile(`[^\w]`)

func (m *translateNotebookPaths) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	var seen = make(map[string]string)

	if b.Config.Artifacts == nil {
		b.Config.Artifacts = make(map[string]*config.Artifact)
	}

	for _, job := range b.Config.Resources.Jobs {
		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			if task.NotebookTask == nil {
				continue
			}

			relPath := path.Clean(task.NotebookTask.NotebookPath)
			absPath := filepath.Join(b.Config.Path, relPath)

			// This is opportunistic. If we can't stat, continue.
			_, err := os.Stat(absPath)
			if err != nil {
				continue
			}

			// Define artifact for this notebook.
			id := nonWord.ReplaceAllString(relPath, "_")
			if v, ok := seen[id]; ok {
				task.NotebookTask.NotebookPath = v
				continue
			}

			b.Config.Artifacts[id] = &config.Artifact{
				Notebook: &config.NotebookArtifact{
					Path: relPath,
				},
			}

			interp := fmt.Sprintf("${artifacts.%s.notebook.remote_path}", id)
			task.NotebookTask.NotebookPath = interp
			seen[id] = interp
		}
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			if library.Notebook == nil {
				continue
			}

			relPath := path.Clean(library.Notebook.Path)
			absPath := filepath.Join(b.Config.Path, relPath)

			// This is opportunistic. If we can't stat, continue.
			_, err := os.Stat(absPath)
			if err != nil {
				continue
			}

			// Define artifact for this notebook.
			id := nonWord.ReplaceAllString(relPath, "_")
			if v, ok := seen[id]; ok {
				library.Notebook.Path = v
				continue
			}

			b.Config.Artifacts[id] = &config.Artifact{
				Notebook: &config.NotebookArtifact{
					Path: relPath,
				},
			}

			interp := fmt.Sprintf("${artifacts.%s.notebook.remote_path}", id)
			library.Notebook.Path = interp
			seen[id] = interp
		}
	}

	return nil, nil
}
