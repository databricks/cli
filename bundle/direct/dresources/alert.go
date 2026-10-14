package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	alerts "github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/utils"
	sdkalerts "github.com/databricks/sdk-go/alerts/v2"
)

type ResourceAlert struct {
	client *sdkclients.Clients
}

// New initializes a ResourceAlert with the given client.
func (*ResourceAlert) New(client *sdkclients.Clients) *ResourceAlert {
	return &ResourceAlert{client: client}
}

// PrepareState converts bundle config to the SDK type.
func (*ResourceAlert) PrepareState(input *resources.Alert) *alerts.Alert {
	return &input.Alert
}

// DoRead reads the alert by id.
func (r *ResourceAlert) DoRead(ctx context.Context, id string) (*alerts.Alert, error) {
	alert, err := r.client.Alerts.GetAlert(ctx, sdkalerts.GetAlertRequest{Id: new(id)})
	if err != nil {
		return nil, err
	}

	// If the alert is already marked as thrashed, return a 404 on DoRead.
	if alert.LifecycleState == sdkalerts.AlertLifecycleState_Deleted {
		return nil, errResourceDoesNotExist("alert " + id + " is deleted")
	}
	return alerts.AlertFromSDK(alert)
}

// DoCreate creates the alert and returns its id.
func (r *ResourceAlert) DoCreate(ctx context.Context, config *alerts.Alert) (string, *alerts.Alert, error) {
	request, err := alerts.CreateAlertRequestToSDK(&alerts.CreateAlertRequest{Alert: config})
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Alerts.CreateAlert(ctx, *request)
	if err != nil || response == nil {
		return "", nil, err
	}
	alert, err := alerts.AlertFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(alert.Id), alert, nil
}

// DoUpdate updates the alert in place.
func (r *ResourceAlert) DoUpdate(ctx context.Context, id string, config *alerts.Alert, _ *PlanEntry) (*alerts.Alert, error) {
	alert := *config
	alert.Id = new(id)
	request, err := alerts.UpdateAlertRequestToSDK(&alerts.UpdateAlertRequest{
		Alert:      &alert,
		UpdateMask: new("*"),
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.Alerts.UpdateAlert(ctx, *request)
	if err != nil {
		return nil, err
	}
	return alerts.AlertFromSDK(response)
}

// DoDelete deletes the alert by id.
func (r *ResourceAlert) DoDelete(ctx context.Context, id string, _ *alerts.Alert) error {
	_, err := r.client.Alerts.TrashAlert(ctx, sdkalerts.TrashAlertRequest{
		Id:    new(id),
		Purge: new(true),
	})
	return err
}
