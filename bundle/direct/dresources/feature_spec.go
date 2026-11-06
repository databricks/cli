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

// PrepareState sorts the fields in FeatureSpec that are order-free lists, so a reorder in config is
// not a recreate. The FeatureSpec with the order-free list fields sorted is what gets sent and
// stored in the backend.
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
	// Returns the FeatureSpec with its order-free list fields sorted, so a feature spec created
	// outside the bundle also has them sorted.
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

// KeyedSlices compares the fields in FeatureSpec that consist of a list of key-value pairs by key
// instead of position, since their order means nothing. The other list fields in FeatureSpec are
// still compared positionally.
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

// sortOrderFreeLists sorts a FeatureSpec's ExcludeColumns and ExtraPipRequirements. Both are
// treated as sets by the server but stored as string lists, which cannot be compared by a key
// defined in KeyedSlices. Each field is replaced with the sorted list rather than sorted in place,
// so the bundle config is not mutated.
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
