package whl

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type fromLibraries struct{}

func DefineArtifactsFromLibraries() bundle.Mutator {
	return &fromLibraries{}
}

func (m *fromLibraries) Name() string {
	return "artifacts.whl.DefineArtifactsFromLibraries"
}

func (*fromLibraries) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Artifacts) != 0 {
		log.Debugf(ctx, "Skipping defining artifacts from libraries because artifacts section is explicitly defined")
		return nil
	}

	tasks := libraries.FindAllWheelTasksWithLocalLibraries(b)
	for _, task := range tasks {
		for _, lib := range task.Libraries {
			matches, err := filepath.Glob(filepath.Join(b.Path, lib.Whl))
			// File referenced from libraries section does not exists, skipping
			if err != nil {
				continue
			}

			for _, match := range matches {
				name := filepath.Base(match)
				if b.Config.Artifacts == nil {
					b.Config.Artifacts = make(map[string]*config.Artifact)
				}

				log.Debugf(ctx, "Adding an artifact block for %s", match)
				b.Config.Artifacts[name] = &config.Artifact{
					Files: []config.ArtifactFile{
						{Source: match},
					},
					Type: config.ArtifactPythonWheel,
				}
			}
		}
	}

	return nil
}
