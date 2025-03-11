package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type definePlaceholderTarget struct {
	name string
}

const PlaceholderTargetName = "PLACEHOLDER_TARGET"

// DefinePlaceholderTarget adds a target named "PLACEHOLDER_TARGET"
// to the configuration if none have been defined.
//
// We do this because downstream mutators like [SelectDefaultTarget]
// and [SelectTarget] expect at least one target to be defined.
func DefinePlaceholderTarget() bundle.Mutator {
	return &definePlaceholderTarget{
		name: PlaceholderTargetName,
	}
}

func (m *definePlaceholderTarget) Name() string {
	return fmt.Sprintf("DefineDefaultTarget(%s)", m.name)
}

func (m *definePlaceholderTarget) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Nothing to do if the configuration has at least 1 target.
	if len(b.Config.Targets) > 0 {
		return nil
	}

	// Define default target.
	b.Config.Targets = make(map[string]*config.Target)
	b.Config.Targets[m.name] = &config.Target{}
	return nil
}
