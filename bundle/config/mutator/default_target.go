package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type defineDefaultTarget struct {
	name string
}

// DefineDefaultTarget adds a target named "default"
// to the configuration if none have been defined.
func DefineDefaultTarget() bundle.Mutator {
	return &defineDefaultTarget{
		name: "default",
	}
}

func (m *defineDefaultTarget) Name() string {
	return fmt.Sprintf("DefineDefaultTarget(%s)", m.name)
}

func (m *defineDefaultTarget) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Nothing to do if the configuration has at least 1 target.
	if len(b.Config.Targets) > 0 {
		return nil
	}

	// Define default target.
	b.Config.Targets = make(map[string]*config.Target)
	b.Config.Targets[m.name] = &config.Target{}
	return nil
}
