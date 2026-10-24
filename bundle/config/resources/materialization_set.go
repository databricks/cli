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
	SawtoothMaterializedFeatures []ml.MaterializedFeature `json:"sawtooth_materialized_features,omitempty"`

	// BatchOfflineAndOnlineMaterializedFeatures declares, per entry, one feature materialized to
	// the online and offline destination. ExpandMaterializationSetMaterializedFeatureMembers fans
	// each entry out into two MaterializedFeatures.
	BatchOfflineAndOnlineMaterializedFeatures []ml.MaterializedFeature `json:"batch_offline_and_online_materialized_features,omitempty"`
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
