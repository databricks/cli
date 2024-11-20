package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupServicePrincipal struct {
	name string
}

func (l *lookupServicePrincipal) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.ServicePrincipals.GetByDisplayName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.ApplicationId), nil
}

func (l *lookupServicePrincipal) String() string {
	return fmt.Sprintf("service-principal: %s", l.name)
}
