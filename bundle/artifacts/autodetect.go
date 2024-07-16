package artifacts

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

func DetectPackages() bundle.Mutator {
	return &autodetect{}
}

type autodetect struct {
}

func (m *autodetect) Name() string {
	return "artifacts.DetectPackages"
}

func (m *autodetect) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// If artifacts section explicitly defined, do not try to auto detect packages
	if b.Config.Artifacts != nil {
		log.Debugf(ctx, "artifacts block is defined, skipping auto-detecting")
		return nil
	}

	return bundle.Apply(ctx, b, bundle.Seq(
		whl.DetectPackage(),
		whl.DefineArtifactsFromLibraries(),
	))
}
