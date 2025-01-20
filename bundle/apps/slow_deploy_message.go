package apps

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

type slowDeployMessage struct{}

// TODO: needs to be removed when when no_compute option becomes available in TF provider and used in DABs
// See https://github.com/databricks/cli/pull/2144
func (v *slowDeployMessage) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Resources.Apps) > 0 {
		cmdio.LogString(ctx, "Note: Databricks apps included in this bundle may increase initial deployment time due to compute provisioning.")
	}

	return nil
}

func (v *slowDeployMessage) Name() string {
	return "apps.SlowDeployMessage"
}

func SlowDeployMessage() bundle.Mutator {
	return &slowDeployMessage{}
}
