package variable

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Lookup resolves a known UC entity name into its ID at runtime. Only
// UC-applicable lookups are supported in v1; additional kinds (storage
// credentials, external locations, ...) will land as separate fields.
type Lookup struct {
	Metastore string `json:"metastore,omitempty"`
}

type resolver interface {
	Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error)
	String() string
}

func (l *Lookup) constructResolver() (resolver, error) {
	var resolvers []resolver

	if l.Metastore != "" {
		resolvers = append(resolvers, resolveMetastore{name: l.Metastore})
	}

	switch len(resolvers) {
	case 0:
		return nil, errors.New("no valid lookup fields provided")
	case 1:
		return resolvers[0], nil
	default:
		return nil, errors.New("exactly one lookup field must be provided")
	}
}

// Resolve looks the entity up in the target workspace and returns its ID.
func (l *Lookup) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	r, err := l.constructResolver()
	if err != nil {
		return "", err
	}
	return r.Resolve(ctx, w)
}

// String returns a human-readable representation of the lookup.
func (l *Lookup) String() string {
	r, _ := l.constructResolver()
	if r == nil {
		return ""
	}
	return r.String()
}

type resolveMetastore struct {
	name string
}

func (l resolveMetastore) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	result, err := w.Metastores.ListAll(ctx, catalog.ListMetastoresRequest{})
	if err != nil {
		return "", err
	}

	var entities []catalog.MetastoreInfo
	for _, entity := range result {
		if entity.Name == l.name {
			entities = append(entities, entity)
		}
	}

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
