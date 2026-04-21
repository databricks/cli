package mutator

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type selectTarget struct {
	name string
}

// SelectTarget merges the named target into the root configuration and
// records the selection under u.Target + u.Config.Ucm.Target.
func SelectTarget(name string) ucm.Mutator {
	return &selectTarget{name: name}
}

func (m *selectTarget) Name() string {
	return fmt.Sprintf("SelectTarget(%s)", m.name)
}

func (m *selectTarget) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	if u.Config.Targets == nil {
		return diag.Errorf("no targets defined")
	}

	target, ok := u.Config.Targets[m.name]
	if !ok {
		available := slices.Collect(maps.Keys(u.Config.Targets))
		return diag.Errorf("%s: no such target. Available targets: %s", m.name, strings.Join(available, ", "))
	}

	if err := u.Config.MergeTargetOverrides(m.name); err != nil {
		return diag.FromErr(fmt.Errorf("failed to perform target override for target=%s: %w", m.name, err))
	}

	u.Target = target
	u.Config.Ucm.Target = m.name

	// Drop the raw targets block from the merged tree so it doesn't appear
	// twice in validate/summary output.
	u.Config.Targets = nil
	return nil
}
