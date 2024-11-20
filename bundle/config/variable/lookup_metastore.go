package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupMetastore struct {
	name string
}

func (l *lookupMetastore) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Metastores.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.MetastoreId), nil
}

func (l *lookupMetastore) String() string {
	return fmt.Sprintf("metastore: %s", l.name)
}
