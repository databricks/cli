package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
)

type overrideImmutableFolder struct{}

// OverrideImmutableFolder sets bundle.deployment.immutable_folder to true
// if the __TEST_DATABRICKS_IMMUTABLE_FOLDER environment variable is non-empty.
// This allows running the acceptance test suite against the immutable folder
// code path without modifying any databricks.yml files.
func OverrideImmutableFolder() bundle.Mutator {
	return &overrideImmutableFolder{}
}

func (m *overrideImmutableFolder) Name() string {
	return "OverrideImmutableFolder"
}

func (m *overrideImmutableFolder) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if env.Get(ctx, "__TEST_DATABRICKS_IMMUTABLE_FOLDER") == "" {
		return nil
	}
	if b.Config.Experimental == nil {
		b.Config.Experimental = &config.Experimental{}
	}
	b.Config.Experimental.ImmutableFolder = true
	return nil
}
