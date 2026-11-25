package resources

import (
	"context"
	"net/url"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type FeatureSpec struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	ml.FeatureSpec

	// List of grants to apply on this feature spec.
	Grants []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func (f *FeatureSpec) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, f)
}

func (f FeatureSpec) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(f)
}

func (f *FeatureSpec) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.FeatureEngineering.GetFeatureSpec(ctx, ml.GetFeatureSpecRequest{
		Name: name,
	})
	if err != nil {
		log.Debugf(ctx, "feature spec with name %s does not exist: %v", name, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (*FeatureSpec) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "feature_spec",
		PluralName:    "feature_specs",
		SingularTitle: "Feature Spec",
		PluralTitle:   "Feature Specs",
	}
}

func (f *FeatureSpec) InitializeURL(baseURL url.URL) {
	if f.ID == "" {
		return
	}
	f.URL = workspaceurls.ResourceURL(baseURL, "feature_specs", f.ID)
}

func (f *FeatureSpec) GetName() string {
	return f.Name
}
