package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"golang.org/x/exp/maps"
)

type selectDefaultTarget struct{}

// SelectDefaultTarget merges the default target into the root configuration.
func SelectDefaultTarget() bundle.Mutator {
	return &selectDefaultTarget{}
}

func (m *selectDefaultTarget) Name() string {
	return "SelectDefaultTarget"
}

func (m *selectDefaultTarget) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Targets) == 0 {
		return diag.Errorf("no targets defined")
	}

	// One target means there's only one default.
	names := maps.Keys(b.Config.Targets)
	if len(names) == 1 {
		bundle.ApplyContext(ctx, b, SelectTarget(names[0]))
		return nil
	}

	// Multiple targets means we look for the `default` flag.
	var defaults []string
	for name, env := range b.Config.Targets {
		if env != nil && env.Default {
			defaults = append(defaults, name)
		}
	}

	// It is invalid to have multiple targets with the `default` flag set.
	if len(defaults) > 1 {
		return diag.Errorf("multiple targets are marked as default (%s)", strings.Join(defaults, ", "))
	}

	// If no target has the `default` flag set, ask the user to specify one.
	if len(defaults) == 0 {
		return diag.Errorf("please specify target")
	}

	// One default remaining.
	bundle.ApplyContext(ctx, b, SelectTarget(defaults[0]))
	return nil
}
