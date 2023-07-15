package mutator

import (
	"context"
	"fmt"
	"os"

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
	if b.Config.Bundle.Mode != config.Development {
		if b.Config.Bundle.ComputeID != "" {
			return fmt.Errorf("cannot override compute for an environment that does not use 'mode: development'")
		}
		return nil
	}
	if os.Getenv("DATABRICKS_CLUSTER_ID") != "" {
		b.Config.Bundle.ComputeID = os.Getenv("DATABRICKS_CLUSTER_ID")
	}

	if b.Config.Bundle.ComputeID == "" {
		return nil
	}

	r := b.Config.Resources
	for i := range r.Jobs {
		overrideJobCompute(r.Jobs[i], b.Config.Bundle.ComputeID)
	}

	return nil
}
