package variable

import (
	"context"
	"errors"

	"github.com/databricks/databricks-sdk-go"
)

type Lookup struct {
	Alert string `json:"alert,omitempty"`

	ClusterPolicy string `json:"cluster_policy,omitempty"`

	Cluster string `json:"cluster,omitempty"`

	Dashboard string `json:"dashboard,omitempty"`

	InstancePool string `json:"instance_pool,omitempty"`

	Job string `json:"job,omitempty"`

	Metastore string `json:"metastore,omitempty"`

	NotificationDestination string `json:"notification_destination,omitempty"`

	Pipeline string `json:"pipeline,omitempty"`

	Query string `json:"query,omitempty"`

	ServicePrincipal string `json:"service_principal,omitempty"`

	Warehouse string `json:"warehouse,omitempty"`
}

type resolver interface {
	// Resolve resolves the underlying entity's ID.
	Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error)

	// String returns a human-readable representation of the resolver.
	String() string
}

func (l *Lookup) constructResolver() (resolver, error) {
	var resolvers []resolver

	if l.Alert != "" {
		resolvers = append(resolvers, resolveAlert{name: l.Alert})
	}
	if l.ClusterPolicy != "" {
		resolvers = append(resolvers, resolveClusterPolicy{name: l.ClusterPolicy})
	}
	if l.Cluster != "" {
		resolvers = append(resolvers, resolveCluster{name: l.Cluster})
	}
	if l.Dashboard != "" {
		resolvers = append(resolvers, resolveDashboard{name: l.Dashboard})
	}
	if l.InstancePool != "" {
		resolvers = append(resolvers, resolveInstancePool{name: l.InstancePool})
	}
	if l.Job != "" {
		resolvers = append(resolvers, resolveJob{name: l.Job})
	}
	if l.Metastore != "" {
		resolvers = append(resolvers, resolveMetastore{name: l.Metastore})
	}
	if l.NotificationDestination != "" {
		resolvers = append(resolvers, resolveNotificationDestination{name: l.NotificationDestination})
	}
	if l.Pipeline != "" {
		resolvers = append(resolvers, resolvePipeline{name: l.Pipeline})
	}
	if l.Query != "" {
		resolvers = append(resolvers, resolveQuery{name: l.Query})
	}
	if l.ServicePrincipal != "" {
		resolvers = append(resolvers, resolveServicePrincipal{name: l.ServicePrincipal})
	}
	if l.Warehouse != "" {
		resolvers = append(resolvers, resolveWarehouse{name: l.Warehouse})
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

func (l *Lookup) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	r, err := l.constructResolver()
	if err != nil {
		return "", err
	}

	return r.Resolve(ctx, w)
}

func (l *Lookup) String() string {
	r, _ := l.constructResolver()
	if r == nil {
		return ""
	}

	return r.String()
}
