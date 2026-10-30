package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const memberFeatureName = "main.myschema.feat"

func offlineStore() *ml.OfflineStoreConfig {
	return &ml.OfflineStoreConfig{
		CatalogName:     "main",
		SchemaName:      "myschema",
		TableNamePrefix: "feat_off",
	}
}

func onlineStore() *ml.OnlineStoreConfig {
	return &ml.OnlineStoreConfig{
		OnlineStoreName: "store",
		CatalogName:     "main",
		SchemaName:      "myschema",
		TableNamePrefix: "feat",
	}
}

func offlineMember(featureName string) ml.MaterializedFeature {
	return ml.MaterializedFeature{
		FeatureName:         featureName,
		OfflineStoreConfig:  offlineStore(),
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
	}
}

func onlineMember(featureName, storeName string) ml.MaterializedFeature {
	store := onlineStore()
	store.OnlineStoreName = storeName

	return ml.MaterializedFeature{
		FeatureName:         featureName,
		OnlineStoreConfig:   store,
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
	}
}

func normalizeSetDiags(
	t *testing.T,
	set *resources.MaterializationSet,
) (*resources.MaterializationSet, diag.Diagnostics) {
	t.Helper()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				MaterializationSets: map[string]*resources.MaterializationSet{"set1": set},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, resourcemutator.NormalizeMaterializationSetMaterializedFeatureMembers())

	return b.Config.Resources.MaterializationSets["set1"], diags
}

func normalizeSet(t *testing.T, set *resources.MaterializationSet) *resources.MaterializationSet {
	t.Helper()

	normalized, diags := normalizeSetDiags(t, set)
	require.NoError(t, diags.Error())

	return normalized
}

func pathStrings(paths []*structpath.PathNode) []string {
	out := make([]string, len(paths))
	for i, path := range paths {
		out[i] = path.String()
	}
	return out
}

// duplicateMemberKeyError is the wording the service uses for the same rejection, so the two read
// alike wherever a caller meets it. See FeatureStoreMessages.materializationSetMemberKeysNotUnique.
func duplicateMemberKeyError(featureNames string) string {
	return "Features " + featureNames + " in materialization set 'set1' each declares more than " +
		"one materialized feature with the following same fields: feature name, online store " +
		"name, trigger is streaming mode, which is not allowed."
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersAcceptsDistinctKeys(t *testing.T) {
	set := normalizeSet(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			offlineMember(memberFeatureName),
			onlineMember(memberFeatureName, "store"),
			onlineMember(memberFeatureName, "other-store"),
			offlineMember("main.myschema.other"),
		},
	})

	assert.Len(t, set.MaterializedFeatures, 4)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersAcceptsAnEmptySet(t *testing.T) {
	set := normalizeSet(t, &resources.MaterializationSet{})

	assert.Nil(t, set.MaterializedFeatures)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersRejectsTheSameMemberTwice(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			offlineMember(memberFeatureName),
			onlineMember(memberFeatureName, "store"),
			offlineMember(memberFeatureName),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, duplicateMemberKeyError("'"+memberFeatureName+"'"), diags[0].Summary)
	assert.Equal(t,
		[]string{
			"resources.materialization_sets.set1.materialized_features[0]",
			"resources.materialization_sets.set1.materialized_features[2]",
		},
		pathStrings(diags[0].Paths))
}

// The offline destination is outside the member key, because the read never returns it, so two
// members differing only there are the same member declared twice.
func TestNormalizeMaterializationSetMaterializedFeatureMembersRejectsMembersDifferingOutsideTheKey(t *testing.T) {
	elsewhere := offlineMember(memberFeatureName)
	elsewhere.OfflineStoreConfig = &ml.OfflineStoreConfig{
		CatalogName: "main", SchemaName: "myschema", TableNamePrefix: "feat_elsewhere",
	}

	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{offlineMember(memberFeatureName), elsewhere},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+memberFeatureName+"'"), diags[0].Summary)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersNamesEveryCollidingFeatureSorted(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			onlineMember("main.myschema.other", "store"),
			onlineMember("main.myschema.other", "store"),
			offlineMember(memberFeatureName),
			offlineMember(memberFeatureName),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t,
		duplicateMemberKeyError("'"+memberFeatureName+"', 'main.myschema.other'"),
		diags[0].Summary)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersNamesAFeatureOnceForTwoCollisions(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			offlineMember(memberFeatureName),
			offlineMember(memberFeatureName),
			onlineMember(memberFeatureName, "store"),
			onlineMember(memberFeatureName, "store"),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+memberFeatureName+"'"), diags[0].Summary)
}
