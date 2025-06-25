package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type artifactsUseDynamicVersion struct{}

// ApplyArtifactsDynamicVersion configures all artifacts to use dynamic_version when the preset is enabled.
func ApplyArtifactsDynamicVersion() bundle.Mutator {
	return &artifactsUseDynamicVersion{}
}

func (m *artifactsUseDynamicVersion) Name() string {
	return "ApplyArtifactsDynamicVersion"
}

func (m *artifactsUseDynamicVersion) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.Config.Presets.ArtifactsDynamicVersion {
		return nil
	}

	for _, a := range b.Config.Artifacts {
		if a == nil {
			continue
		}
		if a.Type != "whl" {
			// This has no effect since we only apply DynamicVersion if type is "whl" but it makes output cleaner.
			continue
		}
		a.DynamicVersion = true
	}

	return nil
}
