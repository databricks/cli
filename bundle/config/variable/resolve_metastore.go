package variable

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/uc/metastores/v1"
)

type resolveMetastore struct {
	name string
}

func (l resolveMetastore) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	// Collect all metastores with the given name.
	var entities []*metastores.MetastoreInfo
	for entity, err := range w.Metastores.ListMetastoresIter(ctx, metastores.ListMetastoresRequest{}) {
		if err != nil {
			return "", err
		}
		if utils.Deref(entity.Name) == l.name {
			entities = append(entities, entity)
		}
	}

	// Return the ID of the first matching metastore.
	switch len(entities) {
	case 0:
		return "", fmt.Errorf("metastore named %q does not exist", l.name)
	case 1:
		return utils.Deref(entities[0].MetastoreId), nil
	default:
		return "", fmt.Errorf("there are %d instances of metastores named %q", len(entities), l.name)
	}
}

func (l resolveMetastore) String() string {
	return "metastore: " + l.name
}
