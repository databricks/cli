package variable

import (
	"context"
	"fmt"

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
	return fmt.Sprint(entity.Id), nil
}

func (l resolveQuery) String() string {
	return fmt.Sprintf("query: %s", l.name)
}
