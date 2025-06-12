package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveMetastore struct {
	name string
}

func (l resolveMetastore) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	// PLACEHOLDER, this will be fixed in the SDK bump.
	return "", nil
}

func (l resolveMetastore) String() string {
	return "metastore: " + l.name
}
