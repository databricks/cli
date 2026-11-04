package dresources

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// https://docs.databricks.com/api/workspace/featureengineering
// The API has no update: every field is immutable, so any change recreates the feature spec.
type ResourceFeatureSpec struct {
	client *databricks.WorkspaceClient
}

func (*ResourceFeatureSpec) New(client *databricks.WorkspaceClient) *ResourceFeatureSpec {
	return &ResourceFeatureSpec{client: client}
}

// PrepareState sorts the order-free string lists, so a reorder in config is not a recreate. The
// sorted lists are what gets sent and stored; DoRead sorts the same lists on the way back.
func (*ResourceFeatureSpec) PrepareState(input *resources.FeatureSpec) *ml.FeatureSpec {
	state := input.FeatureSpec
	sortOrderFreeLists(&state)
	return &state
}

func (r *ResourceFeatureSpec) DoRead(ctx context.Context, id string) (*ml.FeatureSpec, error) {
	featureSpec, err := r.client.FeatureEngineering.GetFeatureSpec(ctx, ml.GetFeatureSpecRequest{Name: id})
	if err != nil {
		return nil, err
	}
	// A feature spec created outside the bundle may hold these lists in any order.
	sortOrderFreeLists(featureSpec)
	return featureSpec, nil
}

func (r *ResourceFeatureSpec) DoCreate(ctx context.Context, config *ml.FeatureSpec) (string, *ml.FeatureSpec, error) {
	response, err := r.client.FeatureEngineering.CreateFeatureSpec(ctx, ml.CreateFeatureSpecRequest{FeatureSpec: *config})
	if err != nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

func (r *ResourceFeatureSpec) DoDelete(ctx context.Context, id string, _ *ml.FeatureSpec) error {
	return r.client.FeatureEngineering.DeleteFeatureSpec(ctx, ml.DeleteFeatureSpecRequest{Name: id})
}

// KeyedSlices compares the map-like pair lists by key, since their order means nothing. The other
// lists stay positional: features sets the output column order, and lookup keys must follow the
// feature table's primary key order.
func (*ResourceFeatureSpec) KeyedSlices() map[string]any {
	return map[string]any{
		"features[*].feature_lookup.rename_outputs":   renameOutputKey,
		"features[*].feature_lookup.default_values":   defaultValueKey,
		"features[*].feature_function.input_bindings": featureFunctionInputBindingKey,
	}
}

func renameOutputKey(x ml.RenameOutput) (string, string) {
	return "feature", x.Feature
}

func defaultValueKey(x ml.DefaultValue) (string, string) {
	return "feature", x.Feature
}

func featureFunctionInputBindingKey(x ml.FeatureFunctionInputBinding) (string, string) {
	return "parameter", x.Parameter
}

// sortOrderFreeLists sorts exclude_columns and the extra pip requirements. Both are sets to the
// server, but as string lists they cannot be compared by key (KeyedSlices needs struct elements).
// It replaces the slices rather than sorting them in place, so the bundle config is not mutated.
func sortOrderFreeLists(featureSpec *ml.FeatureSpec) {
	featureSpec.ExcludeColumns = sortedClone(featureSpec.ExcludeColumns)
	if featureSpec.ExtraPipRequirements != nil {
		pipRequirements := *featureSpec.ExtraPipRequirements
		pipRequirements.Requirements = sortedClone(pipRequirements.Requirements)
		featureSpec.ExtraPipRequirements = &pipRequirements
	}
}

// sortedClone keeps an empty list empty rather than nil, so the request body is unchanged.
func sortedClone(values []string) []string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return sorted
}
