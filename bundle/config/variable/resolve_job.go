package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type resolveJob struct {
	name string
}

func (l resolveJob) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Jobs.GetBySettingsName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.JobId), nil
}

func (l resolveJob) String() string {
	return fmt.Sprintf("job: %s", l.name)
}
