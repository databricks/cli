package variable

import (
	"context"
	"fmt"

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
	return fmt.Sprint(entity.ApplicationId), nil
}

func (l resolveServicePrincipal) String() string {
	return fmt.Sprintf("service-principal: %s", l.name)
}
