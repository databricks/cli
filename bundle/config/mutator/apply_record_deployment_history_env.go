package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
)

type applyRecordDeploymentHistoryEnv struct{}

// ApplyRecordDeploymentHistoryEnv enables the experimental record_deployment_history
// setting when the DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY environment variable is
// set. This lets the feature be toggled per invocation without editing the bundle
// configuration; it never disables a setting that the configuration already enabled.
func ApplyRecordDeploymentHistoryEnv() bundle.Mutator {
	return &applyRecordDeploymentHistoryEnv{}
}

func (m *applyRecordDeploymentHistoryEnv) Name() string {
	return "ApplyRecordDeploymentHistoryEnv"
}

func (m *applyRecordDeploymentHistoryEnv) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if _, ok := env.RecordDeploymentHistory(ctx); !ok {
		return nil
	}

	if b.Config.Experimental == nil {
		b.Config.Experimental = &config.Experimental{}
	}
	b.Config.Experimental.RecordDeploymentHistory = true
	return nil
}
