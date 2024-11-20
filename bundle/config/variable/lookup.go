package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type resolver interface {
	Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error)

	String() string
}

type Lookup struct {
	Alert string `json:"alert,omitempty"`

	ClusterPolicy string `json:"cluster_policy,omitempty"`

	Cluster string `json:"cluster,omitempty"`

	Dashboard string `json:"dashboard,omitempty"`

	InstancePool string `json:"instance_pool,omitempty"`

	Job string `json:"job,omitempty"`

	Metastore string `json:"metastore,omitempty"`

	Pipeline string `json:"pipeline,omitempty"`

	Query string `json:"query,omitempty"`

	ServicePrincipal string `json:"service_principal,omitempty"`

	Warehouse string `json:"warehouse,omitempty"`
}

func (l *Lookup) constructResolver() (resolver, error) {
	var resolvers []resolver

	if l.Alert != "" {
		resolvers = append(resolvers, lookupAlert{name: l.Alert})
	}
	if l.ClusterPolicy != "" {
		resolvers = append(resolvers, lookupClusterPolicy{name: l.ClusterPolicy})
	}
	if l.Cluster != "" {
		resolvers = append(resolvers, lookupCluster{name: l.Cluster})
	}
	if l.Dashboard != "" {
		resolvers = append(resolvers, lookupDashboard{name: l.Dashboard})
	}
	if l.InstancePool != "" {
		resolvers = append(resolvers, lookupInstancePool{name: l.InstancePool})
	}
	if l.Job != "" {
		resolvers = append(resolvers, lookupJob{name: l.Job})
	}
	if l.Metastore != "" {
		resolvers = append(resolvers, lookupMetastore{name: l.Metastore})
	}
	if l.Pipeline != "" {
		resolvers = append(resolvers, lookupPipeline{name: l.Pipeline})
	}
	if l.Query != "" {
		resolvers = append(resolvers, lookupQuery{name: l.Query})
	}
	if l.ServicePrincipal != "" {
		resolvers = append(resolvers, lookupServicePrincipal{name: l.ServicePrincipal})
	}
	if l.Warehouse != "" {
		resolvers = append(resolvers, lookupWarehouse{name: l.Warehouse})
	}

	switch len(resolvers) {
	case 0:
		return nil, fmt.Errorf("no valid lookup fields provided")
	case 1:
		return resolvers[0], nil
	default:
		return nil, fmt.Errorf("exactly one lookup field must be provided")
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
