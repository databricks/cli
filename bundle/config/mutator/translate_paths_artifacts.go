package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

func selectArtifactPath(resource interface{}, m *translatePaths) *selector {
	artifact, ok := resource.(*config.Artifact)
	if !ok {
		return nil
	}

	return &selector{
		&artifact.Path,
		"artifacts.path",
		m.translateToBundleRootRelativePath,
	}
}

func getArtifactsTransformers(m *translatePaths, b *bundle.Bundle) ([]*transformer, error) {
	var transformers []*transformer = make([]*transformer, 0)

	for key, artifact := range b.Config.Artifacts {
		dir, err := artifact.ConfigFileDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to determine directory for artifact %s: %w", key, err)
		}

		transformers = addTransformerForResource(transformers, m, artifact, dir)
	}

	return transformers, nil
}
