package resources

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
)

// DataQualityMonitor is a data quality monitor on a Unity Catalog schema
// (anomaly detection) or table (data profiling), managed through the
// data-quality/v1 API. It supersedes quality_monitors.
type DataQualityMonitor struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	dataquality.Monitor
}

func (m *DataQualityMonitor) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, m)
}

func (m DataQualityMonitor) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(m)
}

// DataQualityMonitorID composes the bundle ID of a monitor. A monitor has no
// name of its own: it is identified by the object it monitors. The format
// matches the Terraform provider's import ID for databricks_data_quality_monitor.
func DataQualityMonitorID(objectType, objectID string) string {
	return objectType + "," + objectID
}

func ParseDataQualityMonitorID(id string) (objectType, objectID string, err error) {
	objectType, objectID, ok := strings.Cut(id, ",")
	if !ok || objectType == "" || objectID == "" {
		return "", "", fmt.Errorf("invalid data quality monitor id %q (want <object_type>,<object_id>)", id)
	}
	return objectType, objectID, nil
}

func (m *DataQualityMonitor) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	objectType, objectID, err := ParseDataQualityMonitorID(id)
	if err != nil {
		return false, err
	}
	_, err = w.DataQuality.GetMonitor(ctx, dataquality.GetMonitorRequest{
		ObjectType: objectType,
		ObjectId:   objectID,
	})
	if err != nil {
		log.Debugf(ctx, "data quality monitor %s does not exist", id)
		if apierr.IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (*DataQualityMonitor) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "data_quality_monitor",
		PluralName:    "data_quality_monitors",
		SingularTitle: "Data quality monitor",
		PluralTitle:   "Data quality monitors",
	}
}

// The object_id is a UUID, which has no Catalog Explorer URL.
func (m *DataQualityMonitor) InitializeURL(baseURL url.URL) {}

func (m *DataQualityMonitor) GetURL() (string, bool) {
	return "", false
}

func (m *DataQualityMonitor) GetName() string {
	return DataQualityMonitorID(m.ObjectType, m.ObjectId)
}
