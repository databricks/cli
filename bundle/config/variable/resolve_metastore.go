package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveMetastore struct {
	name string
}

func (l resolveMetastore) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Metastores.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.MetastoreId, nil
}

func (l resolveMetastore) String() string {
	return "metastore: " + l.name
}
