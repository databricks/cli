package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
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

func (m *processInclude) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	this, err := config.Load(m.fullPath)
	if err != nil {
		return nil, err
	}
	configDir := filepath.Dir(m.relPath)
	return []bundle.Mutator{LoadGitConfig(configDir)}, b.Config.Merge(this)
}
