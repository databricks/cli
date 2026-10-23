package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
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

func expandSet(t *testing.T, set *resources.MaterializationSet) *resources.MaterializationSet {
	t.Helper()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				MaterializationSets: map[string]*resources.MaterializationSet{"set1": set},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, resourcemutator.ExpandMaterializationSetMembers())
	require.NoError(t, diags.Error())

	return b.Config.Resources.MaterializationSets["set1"]
}

func TestExpandMaterializationSetMembersSawtooth(t *testing.T) {
	set := expandSet(t, &resources.MaterializationSet{
		MaterializationSet: ml.MaterializationSet{},
		SawtoothMaterializedFeatures: []ml.MaterializedFeature{{
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

func TestExpandMaterializationSetMembersBatchOfflineAndOnline(t *testing.T) {
	trigger := &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr}
	set := expandSet(t, &resources.MaterializationSet{
		BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{{
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

func TestExpandMaterializationSetMembersCarriesSharedFields(t *testing.T) {
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
		"sawtooth":   {SawtoothMaterializedFeatures: []ml.MaterializedFeature{entry}},
		"batch pair": {BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{entry}},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := expandSet(t, set)
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

// TestExpandMaterializationSetMembersKeepsStreamingModeOnBatchPair pins that the batch pair carries
// a trigger it has no use for rather than discarding it, so the service reports the mistake instead
// of the shorthand swallowing it.
func TestExpandMaterializationSetMembersKeepsStreamingModeOnBatchPair(t *testing.T) {
	set := expandSet(t, &resources.MaterializationSet{
		BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{{
			FeatureName:        shorthandFeatureName,
			OfflineStoreConfig: shorthandOfflineStore(),
			OnlineStoreConfig:  shorthandOnlineStore(),
			StreamingMode:      &ml.StreamingMode{Mode: ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm},
		}},
	})

	require.Len(t, set.MaterializedFeatures, 2)
	for _, member := range set.MaterializedFeatures {
		assert.Equal(t, ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm, member.StreamingMode.Mode)
	}
}

func TestExpandMaterializationSetMembersAppendsToDeclaredMembers(t *testing.T) {
	declared := ml.MaterializedFeature{
		FeatureName:         "main.myschema.other",
		OfflineStoreConfig:  shorthandOfflineStore(),
		CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeManual, CronExpression: shorthandCronExpr},
	}
	set := expandSet(t, &resources.MaterializationSet{
		MaterializationSet: ml.MaterializationSet{
			MaterializedFeatures: []ml.MaterializedFeature{declared},
		},
		SawtoothMaterializedFeatures: []ml.MaterializedFeature{{
			FeatureName:         shorthandFeatureName,
			OfflineStoreConfig:  shorthandOfflineStore(),
			OnlineStoreConfig:   shorthandOnlineStore(),
			CronScheduleTrigger: &ml.CronSchedule{Mode: ml.CronScheduleModeDerived},
			StreamingMode:       &ml.StreamingMode{Mode: ml.StreamingModeStreamingModeTypeStreamingModeTypeRtm},
		}},
		BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{{
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

func TestExpandMaterializationSetMembersLeavesASetWithoutShorthandAlone(t *testing.T) {
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
			SawtoothMaterializedFeatures:              []ml.MaterializedFeature{},
			BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := expandSet(t, set)
			assert.Equal(t, []ml.MaterializedFeature{declared}, expanded.MaterializedFeatures)
		})
	}
}
