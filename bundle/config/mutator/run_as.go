package mutator

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type setRunAs struct {
}

// SetRunAs mutator is used to go over defined resources such as Jobs and DLT Pipelines
// And set correct execution identity ("run_as" for a job or "is_owner" permission for DLT)
// if top-level "run-as" section is defined in the configuration.
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
		pipeline.Permissions = slices.DeleteFunc(pipeline.Permissions, func(p resources.Permission) bool {
			return (runAs.ServicePrincipalName != "" && p.ServicePrincipalName == runAs.ServicePrincipalName) ||
				(runAs.UserName != "" && p.UserName == runAs.UserName)
		})
		pipeline.Permissions = append(pipeline.Permissions, resources.Permission{
			Level:                "IS_OWNER",
			ServicePrincipalName: runAs.ServicePrincipalName,
			UserName:             runAs.UserName,
		})
	}

	return nil
}
