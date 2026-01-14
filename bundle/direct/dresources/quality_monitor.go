package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
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

func (*ResourceQualityMonitor) RemapState(info *catalog.MonitorInfo) *QualityMonitorState {
	return &QualityMonitorState{
		CreateMonitor: catalog.CreateMonitor{
			AssetsDir:                info.AssetsDir,
			BaselineTableName:        info.BaselineTableName,
			CustomMetrics:            info.CustomMetrics,
			DataClassificationConfig: info.DataClassificationConfig,
			InferenceLog:             info.InferenceLog,
			LatestMonitorFailureMsg:  info.LatestMonitorFailureMsg,
			Notifications:            info.Notifications,
			OutputSchemaName:         info.OutputSchemaName,
			Schedule:                 info.Schedule,
			SkipBuiltinDashboard:     false,
			SlicingExprs:             info.SlicingExprs,
			Snapshot:                 info.Snapshot,
			TableName:                info.TableName,
			TimeSeries:               info.TimeSeries,
			WarehouseId:              "",
			ForceSendFields:          utils.FilterFields[catalog.CreateMonitor](info.ForceSendFields),
		},
		TableName: info.TableName,
	}
}

func (r *ResourceQualityMonitor) DoRead(ctx context.Context, id string) (*catalog.MonitorInfo, error) {
	return r.client.QualityMonitors.Get(ctx, catalog.GetQualityMonitorRequest{
		TableName: id,
	})
}

func (r *ResourceQualityMonitor) DoCreate(ctx context.Context, config *QualityMonitorState) (string, *catalog.MonitorInfo, error) {
	req := config.CreateMonitor
	req.TableName = config.TableName
	response, err := r.client.QualityMonitors.Create(ctx, req)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.TableName, response, nil
}

func (r *ResourceQualityMonitor) DoUpdate(ctx context.Context, id string, config *QualityMonitorState, _ Changes) (*catalog.MonitorInfo, error) {
	updateRequest := catalog.UpdateMonitor{
		TableName:                id,
		BaselineTableName:        config.BaselineTableName,
		CustomMetrics:            config.CustomMetrics,
		DashboardId:              "",
		DataClassificationConfig: config.DataClassificationConfig,
		InferenceLog:             config.InferenceLog,
		LatestMonitorFailureMsg:  "",
		Notifications:            config.Notifications,
		OutputSchemaName:         config.OutputSchemaName,
		Schedule:                 config.Schedule,
		SlicingExprs:             config.SlicingExprs,
		Snapshot:                 config.Snapshot,
		TimeSeries:               config.TimeSeries,
		ForceSendFields:          utils.FilterFields[catalog.UpdateMonitor](config.ForceSendFields),
	}

	response, err := r.client.QualityMonitors.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ResourceQualityMonitor) DoDelete(ctx context.Context, id string) error {
	_, err := r.client.QualityMonitors.Delete(ctx, catalog.DeleteQualityMonitorRequest{
		TableName: id,
	})
	return err
}

func (*ResourceQualityMonitor) FieldTriggers() map[string]deployplan.ActionType {
	return map[string]deployplan.ActionType{
		"assets_dir": deployplan.Recreate,
		"table_name": deployplan.Recreate,
	}
}

func (r *ResourceQualityMonitor) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *catalog.MonitorInfo) error {
	if path.String() == "warehouse_id" && change.Old == change.New {
		change.Action = deployplan.Skip
		change.Reason = deployplan.ReasonConfigOnly
	}
	return nil
}
