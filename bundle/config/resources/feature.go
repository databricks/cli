package resources

import (
	"context"
	"net/url"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type Feature struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	ml.Feature
}

func (f *Feature) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, f)
}

func (f Feature) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(f)
}

func (f *Feature) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (bool, error) {
	_, err := w.FeatureEngineering.GetFeature(ctx, ml.GetFeatureRequest{
		FullName: fullName,
	})
	if err != nil {
		log.Debugf(ctx, "feature with full name %s does not exist: %v", fullName, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (*Feature) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "feature",
		PluralName:    "features",
		SingularTitle: "Feature",
		PluralTitle:   "Features",
	}
}

func (f *Feature) InitializeURL(baseURL url.URL) {
	if f.ID == "" {
		return
	}
	f.URL = workspaceurls.ResourceURL(baseURL, "features", f.ID)
}

// GetName returns the fully qualified name. Callers read it off the config, which only ever carries
// what the user wrote plus id and modified_status (see statemgmt.StateToBundle), so the OUTPUT_ONLY
// name field is always empty here.
func (f *Feature) GetName() string {
	return f.FullName
}
