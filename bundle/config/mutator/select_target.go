package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type selectTarget struct {
	name string
}

// SelectTarget merges the specified target into the root configuration.
// It runs while the configuration is loading.
func SelectTarget(name string) bundle.Mutator {
	return &selectTarget{
		name: name,
	}
}

func (m *selectTarget) Name() string {
	return fmt.Sprintf("SelectTarget(%s)", m.name)
}

// targetNames returns the names of the targets of the loading configuration
// (nil if there are none).
func targetNames(b *bundle.Bundle) []string {
	targets, ok := b.Config.Value().Get("targets").AsMap()
	if !ok {
		return nil
	}
	names := make([]string, 0, targets.Len())
	for _, p := range targets.Pairs() {
		names = append(names, p.Key.MustString())
	}
	return names
}

func (m *selectTarget) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if k := b.Config.Value().Get("targets").Kind(); k == dyn.KindInvalid || k == dyn.KindNil {
		return diag.Errorf("no targets defined")
	}

	// Get specified target
	if !b.Config.HasTarget(m.name) {
		return diag.Errorf("%s: no such target. Available targets: %s", m.name, strings.Join(targetNames(b), ", "))
	}

	var target *config.Target
	if err := b.Config.DecodeLoading(&target, "targets", m.name); err != nil {
		return diag.FromErr(err)
	}

	// Merge specified target into root configuration structure.
	err := b.Config.MergeTargetOverrides(m.name)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to perform target override for target=%s: %w", m.name, err))
	}

	// Store specified target in configuration for reference.
	b.Target = target
	b.Metrics.TargetCount = int64(len(targetNames(b)))

	err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		v, err := dyn.SetByPath(v, dyn.NewPath(dyn.Key("bundle"), dyn.Key("target")), dyn.V(m.name))
		if err != nil {
			return v, err
		}

		// We need to keep environment in sync for backwards compatibility.
		v, err = dyn.SetByPath(v, dyn.NewPath(dyn.Key("bundle"), dyn.Key("environment")), dyn.V(m.name))
		if err != nil {
			return v, err
		}

		// Cleanup the original targets and environments sections since they
		// show up in the JSON output of the 'summary' and 'validate' commands.
		return dyn.DropKeys(v, []string{"targets", "environments"})
	})
	return diag.FromErr(err)
}
