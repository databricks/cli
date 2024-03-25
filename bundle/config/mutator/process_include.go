package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type processInclude struct {
	fullPath string
	relPath  string
}

// ProcessInclude loads the configuration at [fullPath] and merges it into the configuration.
func ProcessInclude(fullPath, relPath string) bundle.Mutator {
	return &processInclude{
		fullPath: fullPath,
		relPath:  relPath,
	}
}

func (m *processInclude) Name() string {
	return fmt.Sprintf("ProcessInclude(%s)", m.relPath)
}

func (m *processInclude) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	this, err := config.Load(m.fullPath)
	if err != nil {
		return diag.FromErr(err)
	}
	// TODO: Return actual warnings.
	err = b.Config.Merge(this)
	return diag.FromErr(err)
}
