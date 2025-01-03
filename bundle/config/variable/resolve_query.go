package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveQuery struct {
	name string
}

func (l resolveQuery) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Queries.GetByDisplayName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.Id, nil
}

func (l resolveQuery) String() string {
	return "query: " + l.name
}
