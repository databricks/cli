package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type selectTarget struct {
	name string
}

// SelectTarget merges the specified target into the root configuration.
// After merging, it removes the 'Targets' section from the configuration.
func SelectTarget(name string) bundle.Mutator {
	return &selectTarget{
		name: name,
	}
}

func (m *selectTarget) Name() string {
	return fmt.Sprintf("SelectTarget(%s)", m.name)
}

func (m *selectTarget) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	return diag.FromErr(b.SelectTarget(m.name))
}
