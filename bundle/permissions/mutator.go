package permissions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

const CAN_MANAGE = "CAN_MANAGE"
const CAN_VIEW = "CAN_VIEW"
const CAN_RUN = "CAN_RUN"

var allowedLevels = []string{CAN_MANAGE, CAN_VIEW, CAN_RUN}
var levelsMap = map[string](map[string]string){
	"jobs": {
		CAN_MANAGE: "CAN_MANAGE",
		CAN_VIEW:   "CAN_VIEW",
		CAN_RUN:    "CAN_MANAGE_RUN",
	},
	"pipelines": {
		CAN_MANAGE: "CAN_MANAGE",
		CAN_VIEW:   "CAN_VIEW",
		CAN_RUN:    "CAN_RUN",
	},
	"mlflow_experiments": {
		CAN_MANAGE: "CAN_MANAGE",
		CAN_VIEW:   "CAN_READ",
	},
	"mlflow_models": {
		CAN_MANAGE: "CAN_MANAGE",
		CAN_VIEW:   "CAN_READ",
	},
	"model_serving_endpoints": {
		CAN_MANAGE: "CAN_MANAGE",
		CAN_VIEW:   "CAN_VIEW",
		CAN_RUN:    "CAN_QUERY",
	},
}

type bundlePermissions struct{}

func ApplyBundlePermissions() bundle.Mutator {
	return &bundlePermissions{}
}

func (m *bundlePermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := validate(b)
	if err != nil {
		return diag.FromErr(err)
	}

	applyForJobs(ctx, b)
	applyForPipelines(ctx, b)
	applyForMlModels(ctx, b)
	applyForMlExperiments(ctx, b)
	applyForModelServiceEndpoints(ctx, b)

	return nil
}

func validate(b *bundle.Bundle) error {
	for _, p := range b.Config.Permissions {
		if !slices.Contains(allowedLevels, p.Level) {
			return fmt.Errorf("invalid permission level: %s, allowed values: [%s]", p.Level, strings.Join(allowedLevels, ", "))
		}
	}

	return nil
}

func applyForJobs(ctx context.Context, b *bundle.Bundle) {
	for key, job := range b.Config.Resources.Jobs {
		job.Permissions = append(job.Permissions, convert(
			ctx,
			b.Config.Permissions,
			job.Permissions,
			key,
			levelsMap["jobs"],
		)...)
	}
}

func applyForPipelines(ctx context.Context, b *bundle.Bundle) {
	for key, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Permissions = append(pipeline.Permissions, convert(
			ctx,
			b.Config.Permissions,
			pipeline.Permissions,
			key,
			levelsMap["pipelines"],
		)...)
	}
}

func applyForMlExperiments(ctx context.Context, b *bundle.Bundle) {
	for key, experiment := range b.Config.Resources.Experiments {
		experiment.Permissions = append(experiment.Permissions, convert(
			ctx,
			b.Config.Permissions,
			experiment.Permissions,
			key,
			levelsMap["mlflow_experiments"],
		)...)
	}
}

func applyForMlModels(ctx context.Context, b *bundle.Bundle) {
	for key, model := range b.Config.Resources.Models {
		model.Permissions = append(model.Permissions, convert(
			ctx,
			b.Config.Permissions,
			model.Permissions,
			key,
			levelsMap["mlflow_models"],
		)...)
	}
}

func applyForModelServiceEndpoints(ctx context.Context, b *bundle.Bundle) {
	for key, model := range b.Config.Resources.ModelServingEndpoints {
		model.Permissions = append(model.Permissions, convert(
			ctx,
			b.Config.Permissions,
			model.Permissions,
			key,
			levelsMap["model_serving_endpoints"],
		)...)
	}
}

func (m *bundlePermissions) Name() string {
	return "ApplyBundlePermissions"
}
