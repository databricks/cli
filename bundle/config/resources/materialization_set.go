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
// the author declares once. It embeds the API's member, so a field added to the proto reaches the
// bundle without a change here; the expansion splits the destinations and the triggers the entry
// carries across the three members.
//
// The members a shorthand cannot carry are rejected by
// NormalizeMaterializationSetMaterializedFeatureMembers rather than left out of the type: a
// promoted field cannot be removed from the generated schema.
type SawtoothMaterializedFeatures struct {
	ml.MaterializedFeature
}

// BatchOfflineAndOnlineMaterializedFeatures is one feature materialized to both stores on the batch
// path, which the author declares once. Its two members differ only in their destination.
type BatchOfflineAndOnlineMaterializedFeatures struct {
	ml.MaterializedFeature
}

// Each wrapper needs its own pair: the embedded type's methods promote, so a field added to the
// wrapper later would not be serialized.
func (s *SawtoothMaterializedFeatures) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SawtoothMaterializedFeatures) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *BatchOfflineAndOnlineMaterializedFeatures) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s BatchOfflineAndOnlineMaterializedFeatures) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
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
