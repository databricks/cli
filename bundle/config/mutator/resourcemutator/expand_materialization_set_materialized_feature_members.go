package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type expandMaterializationSetMaterializedFeatureMembers struct{}

func ExpandMaterializationSetMaterializedFeatureMembers() bundle.Mutator {
	return &expandMaterializationSetMaterializedFeatureMembers{}
}

func (m *expandMaterializationSetMaterializedFeatureMembers) Name() string {
	return "ExpandMaterializationSetMaterializedFeatureMembers"
}

// sawtoothMaterializedFeatureMembers builds the three materialized features a sawtooth-window
// feature needs to declare for its materialization: a batch offline materialized feature, a batch
// online materialized feature, and a streaming online materialized feature, where the batch offline
// and batch online use a DERIVED cron schedule and the streaming online and batch online must have
// the same online destination.
//
// materialized_feature_id is absent here because it is assigned server side.
func sawtoothMaterializedFeatureMembers(entry resources.SawtoothMaterializedFeatures) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		// Batch offline.
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
		// Batch online.
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
		// Streaming online.
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

// batchOfflineAndOnlineMaterializedFeatureMembers builds two materializations of the specified
// feature to the online and offline destination.
func batchOfflineAndOnlineMaterializedFeatureMembers(entry resources.BatchOfflineAndOnlineMaterializedFeatures) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		// Batch offline.
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
		// Batch online.
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
	}
}

// Apply mutates typed state, and the Python path merges the dynamic tree back once the normalize
// sequence returns, so this must not be the last entry in applyNormalizeMutators: that tree picks up
// a typed mutation only when the next mutator enters.
func (m *expandMaterializationSetMaterializedFeatureMembers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, set := range b.Config.Resources.MaterializationSets {
		if set == nil {
			continue
		}

		// Append rather than assign because a set can declare one feature in shorthand and another
		// as individual materialized feature members. The adapter keys materialized feature members
		// by content rather than by position.
		for _, entry := range set.SawtoothMaterializedFeatures {
			set.MaterializedFeatures = append(set.MaterializedFeatures,
				sawtoothMaterializedFeatureMembers(entry)...)
		}
		for _, entry := range set.BatchOfflineAndOnlineMaterializedFeatures {
			set.MaterializedFeatures = append(set.MaterializedFeatures,
				batchOfflineAndOnlineMaterializedFeatureMembers(entry)...)
		}

		set.SawtoothMaterializedFeatures = nil
		set.BatchOfflineAndOnlineMaterializedFeatures = nil
	}

	return nil
}
