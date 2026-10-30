package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
)

// Data quality monitor.
// API: https://docs.databricks.com/api/workspace/dataquality
// Terraform: databricks_data_quality_monitor
type ResourceDataQualityMonitor struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDataQualityMonitor) New(client *databricks.WorkspaceClient) *ResourceDataQualityMonitor {
	return &ResourceDataQualityMonitor{client: client}
}

func (*ResourceDataQualityMonitor) PrepareState(input *resources.DataQualityMonitor) *dataquality.Monitor {
	return &input.Monitor
}

func (r *ResourceDataQualityMonitor) DoRead(ctx context.Context, id string) (*dataquality.Monitor, error) {
	objectType, objectID, err := resources.ParseDataQualityMonitorID(id)
	if err != nil {
		return nil, err
	}
	return r.client.DataQuality.GetMonitor(ctx, dataquality.GetMonitorRequest{
		ObjectType: objectType,
		ObjectId:   objectID,
	})
}

func (r *ResourceDataQualityMonitor) DoCreate(ctx context.Context, config *dataquality.Monitor) (string, *dataquality.Monitor, error) {
	resp, err := r.client.DataQuality.CreateMonitor(ctx, dataquality.CreateMonitorRequest{
		Monitor: *config,
	})
	if err != nil {
		return "", nil, err
	}
	return resources.DataQualityMonitorID(resp.ObjectType, resp.ObjectId), resp, nil
}

func (r *ResourceDataQualityMonitor) DoUpdate(ctx context.Context, id string, config *dataquality.Monitor, _ *PlanEntry) (*dataquality.Monitor, error) {
	objectType, objectID, err := resources.ParseDataQualityMonitorID(id)
	if err != nil {
		return nil, err
	}
	return r.client.DataQuality.UpdateMonitor(ctx, dataquality.UpdateMonitorRequest{
		Monitor:    *config,
		ObjectType: objectType,
		ObjectId:   objectID,
		UpdateMask: "*",
	})
}

func (r *ResourceDataQualityMonitor) DoDelete(ctx context.Context, id string, _ *dataquality.Monitor) error {
	objectType, objectID, err := resources.ParseDataQualityMonitorID(id)
	if err != nil {
		return err
	}
	return r.client.DataQuality.DeleteMonitor(ctx, dataquality.DeleteMonitorRequest{
		ObjectType: objectType,
		ObjectId:   objectID,
	})
}
