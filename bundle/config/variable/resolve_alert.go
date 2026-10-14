package variable

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/alerts/v1"
)

type resolveAlert struct {
	name string
}

func (l resolveAlert) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	entity, err := findByName(w.AlertsV1.ListAlertsIter(ctx, alerts.ListAlertsRequest{}), l.name, "ListAlertsResponseAlert", func(a *alerts.ListAlertsResponseAlert) string {
		return utils.Deref(a.DisplayName)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.Id), nil
}

func (l resolveAlert) String() string {
	return "alert: " + l.name
}
