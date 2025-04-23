package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"golang.org/x/exp/maps"
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
	if b.Config.Targets == nil {
		return diag.Errorf("no targets defined")
	}

	// Get specified target
	target, ok := b.Config.Targets[m.name]
	if !ok {
		return diag.Errorf("%s: no such target. Available targets: %s", m.name, strings.Join(maps.Keys(b.Config.Targets), ", "))
	}

	// Merge specified target into root configuration structure.
	err := b.Config.MergeTargetOverrides(m.name)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to perform target override for target=%s: %w", m.name, err))
	}

	// Store specified target in configuration for reference.
	b.Target = target
	b.Config.Bundle.Target = m.name

	// We do this for backward compatibility.
	// TODO: remove when Environments section is not supported anymore.
	b.Config.Bundle.Environment = b.Config.Bundle.Target

	// Record number of targets in the bundle. This is then used downstream during
	// telemetry upload. This value is always >= 1 since if no targets are defined
	// in YAML, we create a "default" placeholder target upstream.
	b.Metrics.TargetCount = int64(len(b.Config.Targets))

	// Cleanup the original targets and environments sections since they
	// show up in the JSON output of the 'summary' and 'validate' commands.
	b.Config.Targets = nil
	b.Config.Environments = nil

	return nil
}
