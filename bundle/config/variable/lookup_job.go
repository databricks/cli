package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupJob struct {
	name string
}

func (l lookupJob) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Jobs.GetBySettingsName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.JobId), nil
}

func (l lookupJob) String() string {
	return fmt.Sprintf("job: %s", l.name)
}
