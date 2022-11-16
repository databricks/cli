package mutator

import (
	"fmt"

	"github.com/databricks/bricks/bundle/config"
)

type processInclude struct {
	fullPath string
	relPath  string
}

func ProcessInclude(fullPath, relPath string) Mutator {
	return &processInclude{
		fullPath: fullPath,
		relPath:  relPath,
	}
}

func (m *processInclude) Name() string {
	return fmt.Sprintf("ProcessInclude(%s)", m.relPath)
}

func (m *processInclude) Apply(root *config.Root) ([]Mutator, error) {
	this, err := config.Load(m.fullPath)
	if err != nil {
		return nil, err
	}
	return nil, root.Merge(this)
}
