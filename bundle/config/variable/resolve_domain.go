package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/domains"
)

// resolveDomain resolves a domain by the key of the governed tag it is built on. The tag
// key is the name of a domain, and a governed tag has at most one domain.
type resolveDomain struct {
	name string
}

func (l resolveDomain) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	result, err := w.Domains.ListDomainsAll(ctx, domains.ListDomainsRequest{})
	if err != nil {
		return "", err
	}

	for _, entity := range result {
		if entity.TagKey == l.name {
			return entity.DomainId, nil
		}
	}

	return "", fmt.Errorf("domain with tag key %q does not exist, or you do not have permission to manage it", l.name)
}

func (l resolveDomain) String() string {
	return "domain: " + l.name
}
