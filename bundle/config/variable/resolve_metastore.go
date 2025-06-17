package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type resolveMetastore struct {
	name string
}

func (l resolveMetastore) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	result, err := w.Metastores.ListAll(ctx, catalog.ListMetastoresRequest{})
	if err != nil {
		return "", err
	}

	// Collect all metastores with the given name.
	var entities []catalog.MetastoreInfo
	for _, entity := range result {
		if entity.Name == l.name {
			entities = append(entities, entity)
		}
	}

	// Return the ID of the first matching metastore.
	switch len(entities) {
	case 0:
		return "", fmt.Errorf("metastore named %q does not exist", l.name)
	case 1:
		return entities[0].MetastoreId, nil
	default:
		return "", fmt.Errorf("there are %d instances of metastores named %q", len(entities), l.name)
	}
}

func (l resolveMetastore) String() string {
	return "metastore: " + l.name
}
