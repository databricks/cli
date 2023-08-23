package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type setRunAs struct {
}

func SetRunAs() bundle.Mutator {
	return &setRunAs{}
}

func (m *setRunAs) Name() string {
	return "SetRunAs"
}

func (m *setRunAs) Apply(_ context.Context, b *bundle.Bundle) error {
	runAs := b.Config.RunAs
	if runAs == nil {
		return nil
	}

	for i := range b.Config.Resources.Jobs {
		job := b.Config.Resources.Jobs[i]
		if job.RunAs != nil {
			continue
		}
		job.RunAs = &jobs.JobRunAs{
			ServicePrincipalName: runAs.ServicePrincipalName,
			UserName:             runAs.UserName,
		}
	}

	for i := range b.Config.Resources.Pipelines {
		pipeline := b.Config.Resources.Pipelines[i]
		pipeline.Permissions = append(pipeline.Permissions, resources.Permission{
			Level:                "IS_OWNER",
			ServicePrincipalName: runAs.ServicePrincipalName,
			UserName:             runAs.UserName,
		})
	}

	return nil
}
