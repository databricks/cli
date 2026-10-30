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
