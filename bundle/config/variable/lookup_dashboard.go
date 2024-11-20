package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupDashboard struct {
	name string
}

func (l lookupDashboard) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Dashboards.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.Id), nil
}

func (l lookupDashboard) String() string {
	return fmt.Sprintf("dashboard: %s", l.name)
}
