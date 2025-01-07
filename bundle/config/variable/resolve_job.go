package variable

import (
	"context"
	"strconv"

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
	return strconv.FormatInt(entity.JobId, 10), nil
}

func (l resolveJob) String() string {
	return "job: " + l.name
}
