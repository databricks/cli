package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

func transformArtifactPath(resource any, dir string) *transformer {
	artifact, ok := resource.(*config.Artifact)
	if !ok {
		return nil
	}

	return &transformer{
		dir,
		&artifact.Path,
		"artifacts.path",
		translateNoOp,
	}
}

func applyArtifactTransformers(m *translatePaths, b *bundle.Bundle) error {
	artifactTransformers := []transformFunc{
		transformArtifactPath,
	}

	for key, artifact := range b.Config.Artifacts {
		dir, err := artifact.ConfigFileDirectory()
		if err != nil {
			return fmt.Errorf("unable to determine directory for artifact %s: %w", key, err)
		}

		err = m.applyTransformers(artifactTransformers, b, artifact, dir)
		if err != nil {
			return err
		}
	}

	return nil
}
