// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package variable

import (
	"context"
	"fmt"
	"strings"

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

	if l.Alert != "" {
		return lookupAlert{name: l.Alert}, nil
	}
	if l.ClusterPolicy != "" {
		return lookupClusterPolicy{name: l.ClusterPolicy}, nil
	}
	if l.Cluster != "" {
		return lookupCluster{name: l.Cluster}, nil
	}
	if l.Dashboard != "" {
		return lookupDashboard{name: l.Dashboard}, nil
	}
	if l.InstancePool != "" {
		return lookupInstancePool{name: l.InstancePool}, nil
	}
	if l.Job != "" {
		return lookupJob{name: l.Job}, nil
	}
	if l.Metastore != "" {
		return lookupMetastore{name: l.Metastore}, nil
	}
	if l.Pipeline != "" {
		return lookupPipeline{name: l.Pipeline}, nil
	}
	if l.Query != "" {
		return lookupQuery{name: l.Query}, nil
	}
	if l.ServicePrincipal != "" {
		return lookupServicePrincipal{name: l.ServicePrincipal}, nil
	}
	if l.Warehouse != "" {
		return lookupWarehouse{name: l.Warehouse}, nil
	}

	return nil, fmt.Errorf("no valid lookup fields provided")
}

// func LookupFromMap(m map[string]any) *Lookup {
// 	l := &Lookup{}
// 	if v, ok := m["alert"]; ok {
// 		l.Alert = v.(string)
// 	}
// 	if v, ok := m["cluster_policy"]; ok {
// 		l.ClusterPolicy = v.(string)
// 	}
// 	if v, ok := m["cluster"]; ok {
// 		l.Cluster = v.(string)
// 	}
// 	if v, ok := m["dashboard"]; ok {
// 		l.Dashboard = v.(string)
// 	}
// 	if v, ok := m["instance_pool"]; ok {
// 		l.InstancePool = v.(string)
// 	}
// 	if v, ok := m["job"]; ok {
// 		l.Job = v.(string)
// 	}
// 	if v, ok := m["metastore"]; ok {
// 		l.Metastore = v.(string)
// 	}
// 	if v, ok := m["pipeline"]; ok {
// 		l.Pipeline = v.(string)
// 	}
// 	if v, ok := m["query"]; ok {
// 		l.Query = v.(string)
// 	}
// 	if v, ok := m["service_principal"]; ok {
// 		l.ServicePrincipal = v.(string)
// 	}
// 	if v, ok := m["warehouse"]; ok {
// 		l.Warehouse = v.(string)
// 	}

// 	return l
// }

// func (l *Lookup) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
// 	if err := l.validate(); err != nil {
// 		return "", err
// 	}

// 	r := allResolvers()
// 	if l.Alert != "" {
// 		return r.Alert(ctx, w, l.Alert)
// 	}
// 	if l.ClusterPolicy != "" {
// 		return r.ClusterPolicy(ctx, w, l.ClusterPolicy)
// 	}
// 	if l.Cluster != "" {
// 		return r.Cluster(ctx, w, l.Cluster)
// 	}
// 	if l.Dashboard != "" {
// 		return r.Dashboard(ctx, w, l.Dashboard)
// 	}
// 	if l.InstancePool != "" {
// 		return r.InstancePool(ctx, w, l.InstancePool)
// 	}
// 	if l.Job != "" {
// 		return r.Job(ctx, w, l.Job)
// 	}
// 	if l.Metastore != "" {
// 		return r.Metastore(ctx, w, l.Metastore)
// 	}
// 	if l.Pipeline != "" {
// 		return r.Pipeline(ctx, w, l.Pipeline)
// 	}
// 	if l.Query != "" {
// 		return r.Query(ctx, w, l.Query)
// 	}
// 	if l.ServicePrincipal != "" {
// 		return r.ServicePrincipal(ctx, w, l.ServicePrincipal)
// 	}
// 	if l.Warehouse != "" {
// 		return r.Warehouse(ctx, w, l.Warehouse)
// 	}

// 	return "", fmt.Errorf("no valid lookup fields provided")
// }

func (l *Lookup) String() string {
	r, _ := l.constructResolver()
	if r != nil {
		return r.String()
	}
	return ""
}

func (l *Lookup) validate() error {
	// Validate that only one field is set
	count := 0
	if l.Alert != "" {
		count++
	}
	if l.ClusterPolicy != "" {
		count++
	}
	if l.Cluster != "" {
		count++
	}
	if l.Dashboard != "" {
		count++
	}
	if l.InstancePool != "" {
		count++
	}
	if l.Job != "" {
		count++
	}
	if l.Metastore != "" {
		count++
	}
	if l.Pipeline != "" {
		count++
	}
	if l.Query != "" {
		count++
	}
	if l.ServicePrincipal != "" {
		count++
	}
	if l.Warehouse != "" {
		count++
	}

	if count != 1 {
		return fmt.Errorf("exactly one lookup field must be provided")
	}

	if strings.Contains(l.String(), "${var") {
		return fmt.Errorf("lookup fields cannot contain variable references")
	}

	return nil
}
