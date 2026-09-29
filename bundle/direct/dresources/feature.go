package dresources

import (
	"context"
	"slices"
	"strings"

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
	response, err := r.client.FeatureEngineering.CreateFeature(ctx, ml.CreateFeatureRequest{Feature: *config})
	if err != nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

// The update endpoint rejects a mask containing anything else, so "*" cannot be used. Every other
// field is excluded at plan level in configs/features.yml and features.generated.yml.
var featureUpdatableFields = []string{"description"}

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
		UpdateMask: strings.Join(featureUpdatableFields, ","),
	})
}

func (r *ResourceFeature) DoDelete(ctx context.Context, id string, _ *ml.Feature) error {
	return r.client.FeatureEngineering.DeleteFeature(ctx, ml.DeleteFeatureRequest{FullName: id})
}
