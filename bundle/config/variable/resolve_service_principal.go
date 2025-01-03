package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveServicePrincipal struct {
	name string
}

func (l resolveServicePrincipal) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.ServicePrincipals.GetByDisplayName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.ApplicationId, nil
}

func (l resolveServicePrincipal) String() string {
	return "service-principal: " + l.name
}
