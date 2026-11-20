package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structcopy"
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

var (
	toQualityMonitorState = newCopier[catalog.MonitorInfo, QualityMonitorState]()

	// TableName addresses the monitor, so it is the id. latest_monitor_failure_msg is the
	// backend's latest error message, which the update API ignores.
	toUpdateMonitor = newCopier[QualityMonitorState, catalog.UpdateMonitor](
		structcopy.Skip("TableName", "LatestMonitorFailureMsg"),
	)
)

func (*ResourceQualityMonitor) RemapState(info *catalog.MonitorInfo) *QualityMonitorState {
	return toQualityMonitorState.Copy(info)
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

func (r *ResourceQualityMonitor) DoUpdate(ctx context.Context, id string, config *QualityMonitorState, _ *PlanEntry) (*catalog.MonitorInfo, error) {
	updateRequest := toUpdateMonitor.Copy(config)
	updateRequest.TableName = id

	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	response, err := r.client.QualityMonitors.Update(ctx, *updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ResourceQualityMonitor) DoDelete(ctx context.Context, id string, _ *QualityMonitorState) error {
	//nolint:staticcheck // Direct quality_monitor resource still uses legacy monitor endpoints; v1 data-quality migration is separate work.
	_, err := r.client.QualityMonitors.Delete(ctx, catalog.DeleteQualityMonitorRequest{
		TableName: id,
	})
	return err
}
