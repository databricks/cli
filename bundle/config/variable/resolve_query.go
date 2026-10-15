package variable

import (
	"context"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/queries/v1"
)

type resolveQuery struct {
	name string
}

func (l resolveQuery) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	entity, err := findByName(w.Queries.ListQueriesIter(ctx, queries.ListQueriesRequest{}), l.name, "ListQueryObjectsResponseQuery", func(q *queries.ListQueryObjectsResponseQuery) string {
		return utils.Deref(q.DisplayName)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.Id), nil
}

func (l resolveQuery) String() string {
	return "query: " + l.name
}
