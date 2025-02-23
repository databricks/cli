package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveDashboard struct {
	name string
}

func (l resolveDashboard) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Dashboards.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.Id, nil
}

func (l resolveDashboard) String() string {
	return "dashboard: " + l.name
}
