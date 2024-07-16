package artifacts

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

var inferMutators map[config.ArtifactType]mutatorFactory = map[config.ArtifactType]mutatorFactory{
	config.ArtifactPythonWheel: whl.InferBuildCommand,
}

func getInferMutator(t config.ArtifactType, name string) bundle.Mutator {
	mutatorFactory, ok := inferMutators[t]
	if !ok {
		return nil
	}

	return mutatorFactory(name)
}

func InferMissingProperties() bundle.Mutator {
	return &all{
		name: "infer",
		fn:   inferArtifactByName,
	}
}

func inferArtifactByName(name string) (bundle.Mutator, error) {
	return &infer{name}, nil
}

type infer struct {
	name string
}

func (m *infer) Name() string {
	return fmt.Sprintf("artifacts.Infer(%s)", m.name)
}

func (m *infer) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	// only try to infer command if it's not already defined
	// and there is no explicitly files defined which means
	// that the package is built outside of bundle cycles
	// manually by customer
	if artifact.BuildCommand != "" || len(artifact.Files) > 0 {
		return nil
	}

	inferMutator := getInferMutator(artifact.Type, m.name)
	if inferMutator != nil {
		return bundle.Apply(ctx, b, inferMutator)
	}

	return nil
}
