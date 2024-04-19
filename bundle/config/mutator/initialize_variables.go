package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
)

type initializeVariables struct{}

// InitializeVariables initializes nil variables to their corresponding zero values.
func InitializeVariables() bundle.Mutator {
	return &initializeVariables{}
}

func (m *initializeVariables) Name() string {
	return "InitializeVariables"
}

func (m *initializeVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	vars := b.Config.Variables
	for k, v := range vars {
		if v == nil {
			vars[k] = &variable.Variable{}
		}
	}

	return nil
}
