package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveAlert struct {
	name string
}

func (l resolveAlert) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Alerts.GetByDisplayName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.Id, nil
}

func (l resolveAlert) String() string {
	return "alert: " + l.name
}
