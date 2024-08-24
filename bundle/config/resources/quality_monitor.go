package resources

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type QualityMonitor struct {
	// Represents the Input Arguments for Terraform and will get
	// converted to a HCL representation for CRUD
	*catalog.CreateMonitor

	// This represents the id which is the full name of the monitor
	// (catalog_name.schema_name.table_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
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

func (s *QualityMonitor) TerraformResourceName() string {
	return "databricks_quality_monitor"
}

func (s *QualityMonitor) InitializeURL(urlPrefix string, urlSuffix string) {
	if s.TableName == "" {
		return
	}
	s.URL = urlPrefix + "explore/data/" + strings.ReplaceAll(s.TableName, ".", "/") + urlSuffix
}

func (s *QualityMonitor) GetName() string {
	return s.TableName
}

func (s *QualityMonitor) GetURL() string {
	return s.URL
}
