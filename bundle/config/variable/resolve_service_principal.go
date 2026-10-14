package variable

import (
	"context"
	"fmt"
	"iter"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/scim/v1"
)

type resolveServicePrincipal struct {
	name string
}

func (l resolveServicePrincipal) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	// Only the first match is used, so fetch no more than the first page.
	next, stop := iter.Pull2(w.Scim.ListServicePrincipalsIter(ctx, scim.ListServicePrincipalsRequest{
		Filter: new(fmt.Sprintf("displayName eq '%s'", l.name)),
	}))
	defer stop()
	var servicePrincipals []*scim.ServicePrincipal
	sp, err, ok := next()
	if err != nil {
		return "", err
	}
	if ok {
		servicePrincipals = append(servicePrincipals, sp)
	}
	if len(servicePrincipals) == 0 {
		return "", fmt.Errorf("service principal named %q does not exist", l.name)
	}

	if len(servicePrincipals) > 1 {
		return "", fmt.Errorf("multiple service principals found with display name %q", l.name)
	}

	return utils.Deref(servicePrincipals[0].ApplicationId), nil
}

func (l resolveServicePrincipal) String() string {
	return "service-principal: " + l.name
}
