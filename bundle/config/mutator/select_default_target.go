package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
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
		return bundle.Apply(ctx, b, SelectTarget(names[0]))
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

	// Still no default? Then use development mode as a fallback.
	// We support this as an optional fallback because it's a common
	// pattern to have a single development environment, and it
	// helps make databricks.yml even more concise.
	if len(defaults) == 0 {
		for name, env := range b.Config.Targets {
			if env != nil && env.Mode == config.Development {
				defaults = append(defaults, name)
			}
		}
	}

	// If no target has the `default` flag set, ask the user to specify one.
	if len(defaults) == 0 {
		return diag.Errorf("please specify target")
	}

	// One default remaining.
	return bundle.Apply(ctx, b, SelectTarget(defaults[0]))
}
