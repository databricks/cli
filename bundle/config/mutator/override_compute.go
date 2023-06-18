package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

type overrideCompute struct {
	compute string
}

func OverrideCompute(compute string) bundle.Mutator {
	return &overrideCompute{compute: compute}
}

func (m *overrideCompute) Name() string {
	return "OverrideCompute"
}

func (m *overrideCompute) overrideJobCompute(j *resources.Job) {
	for i := range j.Tasks {
		task := &j.Tasks[i]
		if task.NewCluster != nil {
			task.NewCluster = nil
			task.ExistingClusterId = m.compute
		} else if task.ExistingClusterId != "" {
			task.ExistingClusterId = m.compute
		}
	}
}

func (m *overrideCompute) Apply(ctx context.Context, b *bundle.Bundle) error {
	if m.compute == "" {
		return nil
	}
	if b.Config.Bundle.Mode != config.Debug {
		return fmt.Errorf("cannot override compute for an environment that does not use 'mode: debug'")
	}

	r := b.Config.Resources
	for i := range r.Jobs {
		m.overrideJobCompute(r.Jobs[i])
	}

	return nil
}
