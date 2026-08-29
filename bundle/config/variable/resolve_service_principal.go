package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type resolveServicePrincipal struct {
	name string
}

func (l resolveServicePrincipal) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	//nolint:staticcheck // this API is deprecated but we still need use it as there is no replacement yet.
	it := w.ServicePrincipalsV2.List(ctx, iam.ListServicePrincipalsRequest{
		Filter: fmt.Sprintf("displayName eq '%s'", l.name),
	})

	servicePrincipals, err := listing.ToSliceN(ctx, it, 1)
	if err != nil {
		return "", err
	}
	if len(servicePrincipals) == 0 {
		return "", fmt.Errorf("service principal named %q does not exist", l.name)
	}

	if len(servicePrincipals) > 1 {
		return "", fmt.Errorf("multiple service principals found with display name %q", l.name)
	}

	return servicePrincipals[0].ApplicationId, nil
}

func (l resolveServicePrincipal) String() string {
	return "service-principal: " + l.name
}
