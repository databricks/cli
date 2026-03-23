package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type QualityMonitorState struct {
	catalog.CreateMonitor

	// The table name is a required field but not included as a JSON field in [catalog.CreateMonitor].
	TableName string `json:"table_name"`
}

// We need to provide these custom marshaller from CreateMonitor takes over and ignores TableName field
func (s *QualityMonitorState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s QualityMonitorState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceQualityMonitor struct {
	client *databricks.WorkspaceClient
}

func (*ResourceQualityMonitor) New(client *databricks.WorkspaceClient) *ResourceQualityMonitor {
	return &ResourceQualityMonitor{client: client}
}

func (*ResourceQualityMonitor) PrepareState(input *resources.QualityMonitor) *QualityMonitorState {
	return &QualityMonitorState{
		CreateMonitor: input.CreateMonitor,
		TableName:     input.TableName,
	}
}

// qualityMonitorRemapCopy maps MonitorInfo (remote GET response) to CreateMonitor (local state).
var qualityMonitorRemapCopy = newCopy[catalog.MonitorInfo, catalog.CreateMonitor]()

func (*ResourceQualityMonitor) RemapState(info *catalog.MonitorInfo) *QualityMonitorState {
	return &QualityMonitorState{
		CreateMonitor: *qualityMonitorRemapCopy.Do(info),
		TableName:     info.TableName,
	}
}

func (r *ResourceQualityMonitor) DoRead(ctx context.Context, id string) (*catalog.MonitorInfo, error) {
	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	return r.client.QualityMonitors.Get(ctx, catalog.GetQualityMonitorRequest{
		TableName: id,
	})
}

func (r *ResourceQualityMonitor) DoCreate(ctx context.Context, config *QualityMonitorState) (string, *catalog.MonitorInfo, error) {
	req := config.CreateMonitor
	req.TableName = config.TableName
	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	response, err := r.client.QualityMonitors.Create(ctx, req)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.TableName, response, nil
}

// qualityMonitorUpdateCopy maps CreateMonitor (local state) to UpdateMonitor (API request).
var qualityMonitorUpdateCopy = newCopy[catalog.CreateMonitor, catalog.UpdateMonitor]()

func (r *ResourceQualityMonitor) DoUpdate(ctx context.Context, id string, config *QualityMonitorState, _ Changes) (*catalog.MonitorInfo, error) {
	updateRequest := qualityMonitorUpdateCopy.Do(&config.CreateMonitor)
	updateRequest.TableName = id

	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	response, err := r.client.QualityMonitors.Update(ctx, *updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ResourceQualityMonitor) DoDelete(ctx context.Context, id string) error {
	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	_, err := r.client.QualityMonitors.Delete(ctx, catalog.DeleteQualityMonitorRequest{
		TableName: id,
	})
	return err
}

