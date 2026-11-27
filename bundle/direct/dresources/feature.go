package dresources

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// https://docs.databricks.com/api/workspace/featureengineering
// Terraform: https://github.com/databricks/terraform-provider-databricks/blob/main/internal/providers/pluginfw/products/featureengineering/
type ResourceFeature struct {
	client *databricks.WorkspaceClient
}

func (*ResourceFeature) New(client *databricks.WorkspaceClient) *ResourceFeature {
	return &ResourceFeature{client: client}
}

func (*ResourceFeature) PrepareState(input *resources.Feature) *ml.Feature {
	return &input.Feature
}

func (r *ResourceFeature) DoRead(ctx context.Context, id string) (*ml.Feature, error) {
	return r.client.FeatureEngineering.GetFeature(ctx, ml.GetFeatureRequest{FullName: id})
}

func (r *ResourceFeature) DoCreate(ctx context.Context, config *ml.Feature) (string, *ml.Feature, error) {
	create := func(ctx context.Context) (*ml.Feature, error) {
		return r.client.FeatureEngineering.CreateFeature(ctx, ml.CreateFeatureRequest{Feature: *config})
	}
	start := time.Now()
	response, err := create(ctx)
	if err != nil {
		// The feature is keyed by full_name, which the bundle chooses and the API keeps unique. UC
		// stores the name lowercased, which is also the id a successful create returns.
		fullName := strings.ToLower(config.FullName)
		response, err = recoverCreate(ctx, createRecovery[ml.Feature, *ml.Feature]{
			name:  "feature " + fullName,
			start: start,
			get: func(ctx context.Context) (*ml.Feature, error) {
				return r.client.FeatureEngineering.GetFeature(ctx, ml.GetFeatureRequest{FullName: fullName})
			},
			createTime:      func(found *ml.Feature) (time.Time, bool) { return sdkTime(found.CreatedAt) },
			alreadyExists:   isAlreadyExists,
			createdByCaller: nil,
			use:             useFound[ml.Feature],
			create:          repeatCreate(create),
		}, err)
		if err != nil {
			return "", nil, err
		}
	}
	return response.FullName, response, nil
}

// The update endpoint rejects a mask containing anything else, so "*" cannot be used. Every other
// field is excluded at plan level in configs/features.yml and features.generated.yml.
const featureUpdateMask = "description"

func (r *ResourceFeature) DoUpdate(ctx context.Context, id string, config *ml.Feature, _ *PlanEntry) (*ml.Feature, error) {
	// The whole feature goes in the body, with the mask limiting what is applied: full_name,
	// function and source have no omitempty, so a body built from just the updatable fields would
	// send them blanked out. Description is forced so clearing it sends "" rather than omitting it.
	update := *config
	update.FullName = id
	update.ForceSendFields = append(slices.Clone(config.ForceSendFields), "Description")

	return r.client.FeatureEngineering.UpdateFeature(ctx, ml.UpdateFeatureRequest{
		FullName:   id,
		Feature:    update,
		UpdateMask: featureUpdateMask,
	})
}

func (r *ResourceFeature) DoDelete(ctx context.Context, id string, _ *ml.Feature) error {
	return r.client.FeatureEngineering.DeleteFeature(ctx, ml.DeleteFeatureRequest{FullName: id})
}
