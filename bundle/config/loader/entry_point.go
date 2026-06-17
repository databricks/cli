package loader

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
)

type entryPoint struct{}

// EntryPoint loads the entry point configuration.
func EntryPoint() bundle.Mutator {
	return &entryPoint{}
}

func (m *entryPoint) Name() string {
	return "EntryPoint"
}

func (m *entryPoint) Apply(ctx context.Context, b *bundle.Bundle) error {
	path, err := config.FileNames.FindInPath(b.BundleRootPath)
	if err != nil {
		return err
	}
	this, diags := config.Load(path)
	for _, d := range diags {
		if d.Severity != diag.Error {
			logdiag.LogDiag(ctx, d)
		}
	}
	if err := diags.Error(); err != nil {
		return err
	}
	return b.Config.Merge(this)
}
