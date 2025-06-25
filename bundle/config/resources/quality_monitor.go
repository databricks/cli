package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type QualityMonitor struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

	// The table name is a required field but not included as a JSON field in [catalog.CreateMonitor].
	TableName string `json:"table_name"`

	// This struct defines the creation payload for a monitor.
	catalog.CreateMonitor
}

func (s *QualityMonitor) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s QualityMonitor) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *QualityMonitor) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.QualityMonitors.Get(ctx, catalog.GetQualityMonitorRequest{
		TableName: id,
	})
	if err != nil {
		log.Debugf(ctx, "quality monitor %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*QualityMonitor) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "quality_monitor",
		PluralName:    "quality_monitors",
		SingularTitle: "Quality Monitor",
		PluralTitle:   "Quality Monitors",
	}
}

func (s *QualityMonitor) InitializeURL(baseURL url.URL) {
	if s.TableName == "" {
		return
	}
	baseURL.Path = "explore/data/" + strings.ReplaceAll(s.TableName, ".", "/")
	s.URL = baseURL.String()
}

func (s *QualityMonitor) GetName() string {
	return s.TableName
}

func (s *QualityMonitor) GetURL() string {
	return s.URL
}
