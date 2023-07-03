package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

type overrideCompute struct{}

func OverrideCompute() bundle.Mutator {
	return &overrideCompute{}
}

func (m *overrideCompute) Name() string {
	return "OverrideCompute"
}

func overrideJobCompute(j *resources.Job, compute string) {
	for i := range j.Tasks {
		task := &j.Tasks[i]
		if task.NewCluster != nil {
			task.NewCluster = nil
			task.ExistingClusterId = compute
		} else if task.ExistingClusterId != "" {
			task.ExistingClusterId = compute
		}
	}
}

func (m *overrideCompute) Apply(ctx context.Context, b *bundle.Bundle) error {
	if b.Config.Bundle.Compute == "" {
		return nil
	}
	if b.Config.Bundle.Mode != config.Development {
		return fmt.Errorf("cannot override compute for an environment that does not use 'mode: debug'")
	}

	r := b.Config.Resources
	for i := range r.Jobs {
		overrideJobCompute(r.Jobs[i], b.Config.Bundle.Compute)
	}

	return nil
}
