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

const (
	shorthandFeatureName = "main.myschema.feat"
	shorthandCronExpr    = "0 0 * * * ?"
)

func shorthandOfflineStore() *ml.OfflineStoreConfig {
	return &ml.OfflineStoreConfig{
		CatalogName:     "main",
		SchemaName:      "myschema",
		TableNamePrefix: "feat_off",
	}
}

func shorthandOnlineStore() *ml.OnlineStoreConfig {
	return &ml.OnlineStoreConfig{
		OnlineStoreName: "store",
		CatalogName:     "main",
		SchemaName:      "myschema",
		TableNamePrefix: "feat",
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

func TestNormalizeMaterializationSetMaterializedFeatureMembersSawtooth(t *testing.T) {
	set := normalizeSet(t, &resources.MaterializationSet{
		MaterializationSet: ml.MaterializationSet{},
		SawtoothMaterializedFeatures: []resources.SawtoothMaterializedFeatures{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
			StreamingMode:       &ml.StreamingMode{Mode: ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm},
		}},
	})

	require.Len(t, set.MaterializedFeatures, 3)

	offlineBatch := set.MaterializedFeatures[0]
	assert.Equal(t, shorthandOfflineStore(), offlineBatch.OfflineStoreConfig)
	assert.Nil(t, offlineBatch.OnlineStoreConfig)
	assert.Equal(t, ml.CronScheduleModeDerived, offlineBatch.CronScheduleTrigger.Mode)
	assert.Nil(t, offlineBatch.StreamingMode)

	onlineBatch := set.MaterializedFeatures[1]
	assert.Nil(t, onlineBatch.OfflineStoreConfig)
	assert.Equal(t, shorthandOnlineStore(), onlineBatch.OnlineStoreConfig)
	assert.Equal(t, ml.CronScheduleModeDerived, onlineBatch.CronScheduleTrigger.Mode)
	assert.Nil(t, onlineBatch.StreamingMode)

	onlineStreaming := set.MaterializedFeatures[2]
	assert.Nil(t, onlineStreaming.OfflineStoreConfig)
	assert.Equal(t, shorthandOnlineStore(), onlineStreaming.OnlineStoreConfig)
	assert.Nil(t, onlineStreaming.CronScheduleTrigger)
	assert.Equal(t, ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm, onlineStreaming.StreamingMode.Mode)

	for _, member := range set.MaterializedFeatures {
		assert.Equal(t, shorthandFeatureName, member.FeatureName)
	}
	assert.Nil(t, set.SawtoothMaterializedFeatures)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersBatchOfflineAndOnline(t *testing.T) {
	trigger := &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr}
	set := normalizeSet(t, &resources.MaterializationSet{
		BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: trigger,
		}},
	})

	require.Len(t, set.MaterializedFeatures, 2)

	offline := set.MaterializedFeatures[0]
	assert.Equal(t, shorthandOfflineStore(), offline.OfflineStoreConfig)
	assert.Nil(t, offline.OnlineStoreConfig)

	online := set.MaterializedFeatures[1]
	assert.Nil(t, online.OfflineStoreConfig)
	assert.Equal(t, shorthandOnlineStore(), online.OnlineStoreConfig)

	// The trigger is not split for a batch pair: both members refresh together.
	assert.Equal(t, trigger, offline.CronScheduleTrigger)
	assert.Equal(t, trigger, online.CronScheduleTrigger)
	assert.Nil(t, set.BatchOfflineAndOnlineMaterializedFeatures)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersCarriesSharedFields(t *testing.T) {
	notifications := &ml.MaterializationNotifications{
		OnFailure: &ml.MaterializationFailureNotification{EmailAddresses: []string{"a@example.com"}},
	}
	entry := ml.MaterializedFeature{
		FeatureName:           shorthandFeatureName,
		Tags:                  map[string]string{"team": "feature-store"},
		BudgetPolicyId:        "budget-1",
		Notifications:         notifications,
		PipelineScheduleState: ml.MaterializedFeaturePipelineScheduleStatePaused,
		OfflineStoreConfig:    shorthandOfflineStore(),
		OnlineStoreConfig:     shorthandOnlineStore(),
		CronScheduleTrigger:   &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
		StreamingMode:         &ml.StreamingMode{Mode: ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm},
	}

	for name, set := range map[string]*resources.MaterializationSet{
		"sawtooth": {SawtoothMaterializedFeatures: []resources.SawtoothMaterializedFeatures{{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			// The streaming mode is what tells the two online members apart.
			StreamingMode: entry.StreamingMode,
		}}},
		"batch pair": {BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := normalizeSet(t, set)
			require.NotEmpty(t, expanded.MaterializedFeatures)

			for _, member := range expanded.MaterializedFeatures {
				assert.Equal(t, shorthandFeatureName, member.FeatureName)
				assert.Equal(t, map[string]string{"team": "feature-store"}, member.Tags)
				assert.Equal(t, "budget-1", member.BudgetPolicyId)
				assert.Equal(t, notifications, member.Notifications)
				assert.Equal(t, ml.MaterializedFeaturePipelineScheduleStatePaused, member.PipelineScheduleState)
			}
		})
	}
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersAppendsToDeclaredMembers(t *testing.T) {
	declared := ml.MaterializedFeature{
		FeatureName:         "main.myschema.other",
		OfflineStoreConfig:  shorthandOfflineStore(),
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr},
	}
	set := normalizeSet(t, &resources.MaterializationSet{
		MaterializationSet: ml.MaterializationSet{
			MaterializedFeatures: []ml.MaterializedFeature{declared},
		},
		SawtoothMaterializedFeatures: []resources.SawtoothMaterializedFeatures{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
			StreamingMode:       &ml.StreamingMode{Mode: ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm},
		}},
		BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{{
			FeatureName:         "main.myschema.pair",
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr},
		}},
	})

	require.Len(t, set.MaterializedFeatures, 6)
	assert.Equal(t, declared, set.MaterializedFeatures[0])
	assert.Nil(t, set.SawtoothMaterializedFeatures)
	assert.Nil(t, set.BatchOfflineAndOnlineMaterializedFeatures)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersLeavesASetWithoutShorthandAlone(t *testing.T) {
	declared := ml.MaterializedFeature{
		FeatureName:         shorthandFeatureName,
		OfflineStoreConfig:  shorthandOfflineStore(),
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr},
	}

	for name, set := range map[string]*resources.MaterializationSet{
		"no shorthand field": {
			MaterializationSet: ml.MaterializationSet{
				MaterializedFeatures: []ml.MaterializedFeature{declared},
			},
		},
		"empty shorthand field": {
			MaterializationSet: ml.MaterializationSet{
				MaterializedFeatures: []ml.MaterializedFeature{declared},
			},
			SawtoothMaterializedFeatures:              []resources.SawtoothMaterializedFeatures{},
			BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := normalizeSet(t, set)
			assert.Equal(t, []ml.MaterializedFeature{declared}, expanded.MaterializedFeatures)
		})
	}
}

// --- the member key must be unique within a set ------------------------------------------------

// duplicateMemberKeyError is the wording the service uses for the same rejection, so the two read
// alike wherever a caller meets it. See FeatureStoreMessages.materializationSetMemberKeysNotUnique.
func duplicateMemberKeyError(featureNames string) string {
	return "Features " + featureNames + " in materialization set 'set1' each declares more than " +
		"one materialized feature with the following same fields: feature name, online store " +
		"name, trigger is streaming mode, which is not allowed."
}

func offlineMember(featureName string) ml.MaterializedFeature {
	return ml.MaterializedFeature{
		FeatureName:         featureName,
		OfflineStoreConfig:  shorthandOfflineStore(),
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
	}
}

func onlineMember(featureName, storeName string) ml.MaterializedFeature {
	store := shorthandOnlineStore()
	store.OnlineStoreName = storeName

	return ml.MaterializedFeature{
		FeatureName:         featureName,
		OnlineStoreConfig:   store,
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
	}
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersAcceptsDistinctKeys(t *testing.T) {
	set := normalizeSet(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			offlineMember(shorthandFeatureName),
			onlineMember(shorthandFeatureName, "store"),
			onlineMember(shorthandFeatureName, "other-store"),
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
			offlineMember(shorthandFeatureName),
			onlineMember(shorthandFeatureName, "store"),
			offlineMember(shorthandFeatureName),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, duplicateMemberKeyError("'"+shorthandFeatureName+"'"), diags[0].Summary)
}

// The offline destination is outside the member key, because the read never returns it, so two
// members differing only there are the same member declared twice.
func TestNormalizeMaterializationSetMaterializedFeatureMembersRejectsMembersDifferingOutsideTheKey(t *testing.T) {
	elsewhere := offlineMember(shorthandFeatureName)
	elsewhere.OfflineStoreConfig = &ml.OfflineStoreConfig{
		CatalogName: "main", SchemaName: "myschema", TableNamePrefix: "feat_elsewhere",
	}

	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{offlineMember(shorthandFeatureName), elsewhere},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+shorthandFeatureName+"'"), diags[0].Summary)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersNamesEveryCollidingFeatureSorted(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			onlineMember("main.myschema.other", "store"),
			onlineMember("main.myschema.other", "store"),
			offlineMember(shorthandFeatureName),
			offlineMember(shorthandFeatureName),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t,
		duplicateMemberKeyError("'"+shorthandFeatureName+"', 'main.myschema.other'"),
		diags[0].Summary)
}

func TestNormalizeMaterializationSetMaterializedFeatureMembersNamesAFeatureOnceForTwoCollisions(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{
			offlineMember(shorthandFeatureName),
			offlineMember(shorthandFeatureName),
			onlineMember(shorthandFeatureName, "store"),
			onlineMember(shorthandFeatureName, "store"),
		},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+shorthandFeatureName+"'"), diags[0].Summary)
}

// A shorthand entry and a member written out reach the same member, which neither declaration
// shows on its own.
func TestNormalizeMaterializationSetMaterializedFeatureMembersRejectsAShorthandCollidingWithADeclaredMember(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		MaterializedFeatures: []ml.MaterializedFeature{offlineMember(shorthandFeatureName)},
		BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
		}},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+shorthandFeatureName+"'"), diags[0].Summary)
	assert.Equal(t,
		[]string{
			"resources.materialization_sets.set1.batch_offline_and_online_materialized_features[0]",
			"resources.materialization_sets.set1.materialized_features[0]",
		},
		pathStrings(diags[0].Paths))
}

// A sawtooth entry with no streaming mode expands into a batch online and a streaming online
// member that nothing in the key tells apart.
func TestNormalizeMaterializationSetMaterializedFeatureMembersRejectsASawtoothEntryCollidingWithItself(t *testing.T) {
	_, diags := normalizeSetDiags(t, &resources.MaterializationSet{
		SawtoothMaterializedFeatures: []resources.SawtoothMaterializedFeatures{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
		}},
	})

	require.Len(t, diags, 1)
	assert.Equal(t, duplicateMemberKeyError("'"+shorthandFeatureName+"'"), diags[0].Summary)
	assert.Equal(t,
		[]string{"resources.materialization_sets.set1.sawtooth_materialized_features[0]"},
		pathStrings(diags[0].Paths))
}

func pathStrings(paths []*structpath.PathNode) []string {
	out := make([]string, len(paths))
	for i, path := range paths {
		out[i] = path.String()
	}
	return out
}
