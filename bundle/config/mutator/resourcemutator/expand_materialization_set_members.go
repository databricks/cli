package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type expandMaterializationSetMembers struct{}

func ExpandMaterializationSetMembers() bundle.Mutator {
	return &expandMaterializationSetMembers{}
}

func (m *expandMaterializationSetMembers) Name() string {
	return "ExpandMaterializationSetMembers"
}

// sawtoothMembers builds the three materialized features a sawtooth-window feature is made of. The
// destination and the trigger are split across them; every other field the entry carries is held by
// all three.
//
// materialized_feature_id is deliberately absent: the server assigns one per member, so a single
// declared id cannot belong to three. ForceSendFields is absent because the dyn-to-typed
// conversion rebuilds it per member from the fields that member holds.
func sawtoothMembers(entry ml.MaterializedFeature) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			StreamingMode:         entry.StreamingMode,
		},
	}
}

// batchOfflineAndOnlineMembers builds the two materialized features a feature materialized to both
// stores is made of. Only the destination is split; every other field the entry carries, trigger
// included, is held by both.
//
// The omissions are the same as in sawtoothMembers.
func batchOfflineAndOnlineMembers(entry ml.MaterializedFeature) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
			StreamingMode:         entry.StreamingMode,
		},
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
			StreamingMode:         entry.StreamingMode,
		},
	}
}

func (m *expandMaterializationSetMembers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, set := range b.Config.Resources.MaterializationSets {
		if set == nil {
			continue
		}

		// Appended rather than assigned: a set may declare one feature in shorthand and another
		// as members, and the adapter keys members by content rather than by position.
		for _, entry := range set.SawtoothMaterializedFeatures {
			set.MaterializedFeatures = append(set.MaterializedFeatures, sawtoothMembers(entry)...)
		}
		for _, entry := range set.BatchOfflineAndOnlineMaterializedFeatures {
			set.MaterializedFeatures = append(set.MaterializedFeatures, batchOfflineAndOnlineMembers(entry)...)
		}

		set.SawtoothMaterializedFeatures = nil
		set.BatchOfflineAndOnlineMaterializedFeatures = nil
	}

	return nil
}
