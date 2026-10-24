package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// MaterializationSet is the bundle config for a group of materialized features created and
// managed as one unit.
type MaterializationSet struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	ml.MaterializationSet

	// SawtoothMaterializedFeatures declares, per entry, the whole materialization of one
	// sawtooth-window feature: an offline batch member, an online batch member and an online
	// streaming member. ExpandMaterializationSetMaterializedFeatureMembers fans each entry out
	// into three materialized features.
	SawtoothMaterializedFeatures []SawtoothMaterializedFeatures `json:"sawtooth_materialized_features,omitempty"`

	// BatchOfflineAndOnlineMaterializedFeatures declares, per entry, one feature materialized to
	// the online and offline destination. ExpandMaterializationSetMaterializedFeatureMembers fans
	// each entry out into two MaterializedFeatures.
	BatchOfflineAndOnlineMaterializedFeatures []BatchOfflineAndOnlineMaterializedFeatures `json:"batch_offline_and_online_materialized_features,omitempty"`
}

// SawtoothMaterializedFeatures is the materialized features of one sawtooth-window feature, which
// the author declares once. ml.MaterializedFeature cannot stand in for it: a materialized feature
// holds one destination and one trigger, both declared as proto oneofs, while this holds both
// destinations and both trigger kinds and the expansion splits them across the three members.
//
// materialized_feature_id is absent because the server assigns one per member.
type SawtoothMaterializedFeatures struct {
	FeatureName           string                                      `json:"feature_name"`
	Tags                  map[string]string                           `json:"tags,omitempty"`
	BudgetPolicyId        string                                      `json:"budget_policy_id,omitempty"`
	Notifications         *ml.MaterializationNotifications            `json:"notifications,omitempty"`
	PipelineScheduleState ml.MaterializedFeaturePipelineScheduleState `json:"pipeline_schedule_state,omitempty"`

	OfflineStoreConfig *ml.OfflineStoreConfig `json:"offline_store_config,omitempty"`
	OnlineStoreConfig  *ml.OnlineStoreConfig  `json:"online_store_config,omitempty"`

	// The batch members take the cron schedule and the table trigger; the streaming member takes
	// the streaming mode.
	CronSchedule        string            `json:"cron_schedule,omitempty"`
	CronScheduleTrigger *ml.CronSchedule  `json:"cron_schedule_trigger,omitempty"`
	TableTrigger        *ml.TableTrigger  `json:"table_trigger,omitempty"`
	StreamingMode       *ml.StreamingMode `json:"streaming_mode,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

// BatchOfflineAndOnlineMaterializedFeatures is the materialized features of one feature taken to
// both stores on the batch path, which the author declares once. It carries no streaming mode: a
// feature on the batch path has none, and the two members it expands into differ only in their
// destination.
//
// materialized_feature_id is absent because the server assigns one per member.
type BatchOfflineAndOnlineMaterializedFeatures struct {
	FeatureName           string                                      `json:"feature_name"`
	Tags                  map[string]string                           `json:"tags,omitempty"`
	BudgetPolicyId        string                                      `json:"budget_policy_id,omitempty"`
	Notifications         *ml.MaterializationNotifications            `json:"notifications,omitempty"`
	PipelineScheduleState ml.MaterializedFeaturePipelineScheduleState `json:"pipeline_schedule_state,omitempty"`

	OfflineStoreConfig *ml.OfflineStoreConfig `json:"offline_store_config,omitempty"`
	OnlineStoreConfig  *ml.OnlineStoreConfig  `json:"online_store_config,omitempty"`

	CronSchedule        string           `json:"cron_schedule,omitempty"`
	CronScheduleTrigger *ml.CronSchedule `json:"cron_schedule_trigger,omitempty"`
	TableTrigger        *ml.TableTrigger `json:"table_trigger,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func (s *MaterializationSet) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MaterializationSet) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *MaterializationSet) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.FeatureEngineering.GetMaterializationSet(ctx, ml.GetMaterializationSetRequest{
		MaterializationSetId: id,
	})
	if err != nil {
		log.Debugf(ctx, "materialization set %s does not exist: %v", id, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (*MaterializationSet) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "materialization_set",
		PluralName:    "materialization_sets",
		SingularTitle: "Materialization Set",
		PluralTitle:   "Materialization Sets",
	}
}

// GetName returns the in-product name, which is empty: a set is identified only by its
// server-assigned id.
func (s *MaterializationSet) GetName() string {
	return ""
}

// GetURL reports that this type has no workspace URL: a set has no standalone page, and its
// members surface under their own features.
func (s *MaterializationSet) GetURL() (string, bool) {
	return "", false
}

// InitializeURL leaves URL unset. See GetURL.
func (s *MaterializationSet) InitializeURL(_ url.URL) {}
