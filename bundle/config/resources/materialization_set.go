package resources

import (
	"context"
	"net/url"
	"strconv"
	"strings"

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

// escapeMemberKeyPart percent-escapes the separator so the join in MaterializedFeatureMemberKey is
// injective. A feature name is validated only for length and non-emptiness, so it may contain the
// separator; two members sharing a key would be diffed as one, and the request would then omit a
// member the server deletes because it cannot see it.
func escapeMemberKeyPart(part string) string {
	part = strings.ReplaceAll(part, "%", "%25")
	return strings.ReplaceAll(part, "|", "%7C")
}

// MaterializedFeatureMemberKey identifies a materialized feature member by content rather than
// position, as labelled components joined by the separator escapeMemberKeyPart escapes.
//
// online_store_name is omitted rather than left empty when the member has no online destination,
// which is what keeps an offline member distinct from an online one whose store name is unset.
//
// The server pairs members by the same components in MaterializationSetMemberKey -- see
// feature-store/src/main/scala/com/databricks/feature-store/utils/materialized-features/MaterializationSetUtils.scala
// -- holding them in a case class rather than a joined string, so this encoding is the CLI's own
// and only the component list has to agree. Adding, dropping or redefining one takes the same
// change there.
func MaterializedFeatureMemberKey(member ml.MaterializedFeature) (string, string) {
	parts := []string{"feature_name=" + escapeMemberKeyPart(member.FeatureName)}
	if member.OnlineStoreConfig != nil {
		parts = append(parts,
			"online_store_name="+escapeMemberKeyPart(member.OnlineStoreConfig.OnlineStoreName))
	}
	parts = append(parts, "is_streaming_mode="+strconv.FormatBool(member.StreamingMode != nil))

	return "member", strings.Join(parts, "|")
}

// MaterializedFeatureMemberKeyFieldDescription names the key's components in plain words. It
// mirrors MaterializationSetMemberKey.fieldDescription in MaterializationSetUtils.scala, so the
// CLI's rejection reads like the service's own.
const MaterializedFeatureMemberKeyFieldDescription = "feature name, online store name, trigger is streaming mode"

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
