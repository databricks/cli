package mutator

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type selectDefaultTarget struct{}

// SelectDefaultTarget picks the default target and merges it in. One target
// means that target is the default; multiple targets requires exactly one to
// carry `default: true`.
func SelectDefaultTarget() ucm.Mutator {
	return &selectDefaultTarget{}
}

func (m *selectDefaultTarget) Name() string { return "SelectDefaultTarget" }

func (m *selectDefaultTarget) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	if len(u.Config.Targets) == 0 {
		return diag.Errorf("no targets defined")
	}

	names := slices.Collect(maps.Keys(u.Config.Targets))
	if len(names) == 1 {
		ucm.ApplyContext(ctx, u, SelectTarget(names[0]))
		return nil
	}

	var defaults []string
	for name, t := range u.Config.Targets {
		if t != nil && t.Default {
			defaults = append(defaults, name)
		}
	}

	if len(defaults) > 1 {
		return diag.Errorf("multiple targets are marked as default (%s)", strings.Join(defaults, ", "))
	}
	if len(defaults) == 0 {
		return diag.Errorf("please specify target")
	}

	ucm.ApplyContext(ctx, u, SelectTarget(defaults[0]))
	return nil
}
