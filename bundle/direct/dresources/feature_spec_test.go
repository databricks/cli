package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureSpecPrepareStateSortsOrderFreeListsWithoutMutatingConfig(t *testing.T) {
	input := &resources.FeatureSpec{
		FeatureSpec: ml.FeatureSpec{
			Name:                 "main.default.spec",
			ExcludeColumns:       []string{"b", "a"},
			ExtraPipRequirements: &ml.PipRequirements{Requirements: []string{"scipy", "numpy"}},
		},
	}

	state := (&ResourceFeatureSpec{}).PrepareState(input)

	assert.Equal(t, []string{"a", "b"}, state.ExcludeColumns)
	assert.Equal(t, []string{"numpy", "scipy"}, state.ExtraPipRequirements.Requirements)
	assert.Equal(t, []string{"b", "a"}, input.ExcludeColumns)
	assert.Equal(t, []string{"scipy", "numpy"}, input.ExtraPipRequirements.Requirements)
}

func TestFeatureSpecPrepareStateKeepsEmptyListsEmpty(t *testing.T) {
	input := &resources.FeatureSpec{
		FeatureSpec: ml.FeatureSpec{
			Name:                 "main.default.spec",
			ExtraPipRequirements: &ml.PipRequirements{Requirements: []string{}},
		},
	}

	state := (&ResourceFeatureSpec{}).PrepareState(input)

	assert.Nil(t, state.ExcludeColumns)
	assert.NotNil(t, state.ExtraPipRequirements.Requirements)
	assert.Empty(t, state.ExtraPipRequirements.Requirements)
}

func TestFeatureSpecDoReadSortsOrderFreeLists(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceFeatureSpec{}).New(m.WorkspaceClient)

	m.GetMockFeatureEngineeringAPI().EXPECT().
		GetFeatureSpec(ctx, ml.GetFeatureSpecRequest{Name: "main.default.spec"}).
		Return(&ml.FeatureSpec{
			Name:                 "main.default.spec",
			ExcludeColumns:       []string{"b", "a"},
			ExtraPipRequirements: &ml.PipRequirements{Requirements: []string{"scipy", "numpy"}},
		}, nil).
		Once()

	remote, err := r.DoRead(ctx, "main.default.spec")
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b"}, remote.ExcludeColumns)
	assert.Equal(t, []string{"numpy", "scipy"}, remote.ExtraPipRequirements.Requirements)
}

func featureSpecWithLookupAndFunction(keys []string, renames []ml.RenameOutput, defaults []ml.DefaultValue, bindings []ml.FeatureFunctionInputBinding) *ml.FeatureSpec {
	return &ml.FeatureSpec{
		Name: "main.default.spec",
		Features: []ml.Features{
			{FeatureLookup: &ml.FeatureLookup{
				Table:         "main.default.users",
				LookupKeys:    keys,
				FeatureNames:  []string{"age", "country"},
				RenameOutputs: renames,
				DefaultValues: defaults,
			}},
			{FeatureFunction: &ml.FeatureFunction{
				Udf:           "main.default.score",
				InputBindings: bindings,
				Output:        "score",
			}},
		},
	}
}

func TestFeatureSpecKeyedSlicesIgnoreOrderOfMapLikeLists(t *testing.T) {
	keys := (&ResourceFeatureSpec{}).KeyedSlices()

	config := featureSpecWithLookupAndFunction(
		[]string{"user_id"},
		[]ml.RenameOutput{{Feature: "age", Output: "user_age"}, {Feature: "country", Output: "user_country"}},
		[]ml.DefaultValue{{Feature: "user_age", DefaultValue: "0"}, {Feature: "user_country", DefaultValue: "US"}},
		[]ml.FeatureFunctionInputBinding{{Parameter: "x", BoundFeature: "user_age"}, {Parameter: "y", BoundFeature: "user_country"}},
	)
	remote := featureSpecWithLookupAndFunction(
		[]string{"user_id"},
		[]ml.RenameOutput{{Feature: "country", Output: "user_country"}, {Feature: "age", Output: "user_age"}},
		[]ml.DefaultValue{{Feature: "user_country", DefaultValue: "US"}, {Feature: "user_age", DefaultValue: "0"}},
		[]ml.FeatureFunctionInputBinding{{Parameter: "y", BoundFeature: "user_country"}, {Parameter: "x", BoundFeature: "user_age"}},
	)

	changes, err := structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.Empty(t, changes)

	// A real edit inside a keyed list is still a change.
	remote.Features[0].FeatureLookup.RenameOutputs[0].Output = "renamed_country"
	changes, err = structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.NotEmpty(t, changes)
}

func TestFeatureSpecKeyedSlicesKeepPositionalListsOrdered(t *testing.T) {
	keys := (&ResourceFeatureSpec{}).KeyedSlices()

	// Lookup keys must follow the feature table's primary key order.
	config := featureSpecWithLookupAndFunction([]string{"a", "b"}, nil, nil, nil)
	remote := featureSpecWithLookupAndFunction([]string{"b", "a"}, nil, nil, nil)
	changes, err := structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.NotEmpty(t, changes)

	// The order of features sets the output column order.
	remote = featureSpecWithLookupAndFunction([]string{"a", "b"}, nil, nil, nil)
	remote.Features[0], remote.Features[1] = remote.Features[1], remote.Features[0]
	changes, err = structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.NotEmpty(t, changes)
}
