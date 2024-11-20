package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupAlert struct {
	name string
}

func (l *lookupAlert) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Alerts.GetByDisplayName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.Id), nil
}

func (l *lookupAlert) String() string {
	return fmt.Sprintf("alert: %s", l.name)
}
