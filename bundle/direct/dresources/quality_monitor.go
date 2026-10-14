package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sdkclients"
)

type QualityMonitorState struct {
	qualitymonitors.CreateMonitor

	// The table name is a required field but not included as a JSON field in [qualitymonitors.CreateMonitor].
	TableName string `json:"table_name"`
}

type ResourceQualityMonitor struct {
	client *sdkclients.Clients
}

func (*ResourceQualityMonitor) New(client *sdkclients.Clients) *ResourceQualityMonitor {
	return &ResourceQualityMonitor{client: client}
}

func (*ResourceQualityMonitor) PrepareState(input *resources.QualityMonitor) *QualityMonitorState {
	return &QualityMonitorState{
		CreateMonitor: input.CreateMonitor,
		TableName:     input.TableName,
	}
}

func (*ResourceQualityMonitor) RemapState(info *qualitymonitors.MonitorInfo) *QualityMonitorState {
	return &QualityMonitorState{
		CreateMonitor: qualitymonitors.CreateMonitor{
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
		},
		TableName: info.TableName,
	}
}

func (r *ResourceQualityMonitor) DoRead(ctx context.Context, id string) (*qualitymonitors.MonitorInfo, error) {
	return r.client.QualityMonitors.Get(ctx, qualitymonitors.GetQualityMonitorRequest{
		TableName: id,
	})
}

func (r *ResourceQualityMonitor) DoCreate(ctx context.Context, config *QualityMonitorState) (string, *qualitymonitors.MonitorInfo, error) {
	req := config.CreateMonitor
	req.TableName = config.TableName
	response, err := r.client.QualityMonitors.Create(ctx, req)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.TableName, response, nil
}

func (r *ResourceQualityMonitor) DoUpdate(ctx context.Context, id string, config *QualityMonitorState, _ *PlanEntry) (*qualitymonitors.MonitorInfo, error) {
	updateRequest := qualitymonitors.UpdateMonitor{
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
	}

	response, err := r.client.QualityMonitors.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ResourceQualityMonitor) DoDelete(ctx context.Context, id string, _ *QualityMonitorState) error {
	_, err := r.client.QualityMonitors.Delete(ctx, qualitymonitors.DeleteQualityMonitorRequest{
		TableName: id,
	})
	return err
}
