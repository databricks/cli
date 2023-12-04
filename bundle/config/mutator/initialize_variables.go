package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
)

type initializeVariables struct{}

// InitializeVariables initializes nil variables to their corresponding zero values.
func InitializeVariables() bundle.Mutator {
	return &initializeVariables{}
}

func (m *initializeVariables) Name() string {
	return "InitializeVariables"
}

func (m *initializeVariables) Apply(ctx context.Context, b *bundle.Bundle) error {
	vars := b.Config.Variables
	for k, v := range vars {
		if v == nil {
			vars[k] = &variable.Variable{}
		}
	}

	return nil
}
