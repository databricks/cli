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
// managed as one unit. Members are addressed by content rather than by position, so the set
// is reconciled against its desired membership rather than updated field by field.
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

// InitializeURL leaves URL unset because a set has no standalone page in the workspace; its
// members surface under their own features.
func (s *MaterializationSet) InitializeURL(_ url.URL) {}
