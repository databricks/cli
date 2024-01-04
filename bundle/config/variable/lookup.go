// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package variable

import (
	"context"
	"fmt"
	"strings"

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

	Pipeline string `json:"pipeline,omitempty"`

	Query string `json:"query,omitempty"`

	ServicePrincipal string `json:"service_principal,omitempty"`

	Warehouse string `json:"warehouse,omitempty"`
}

func LookupFromMap(m map[string]any) *Lookup {
	l := &Lookup{}
	if v, ok := m["alert"]; ok {
		l.Alert = v.(string)
	}
	if v, ok := m["cluster-policy"]; ok {
		l.ClusterPolicy = v.(string)
	}
	if v, ok := m["cluster"]; ok {
		l.Cluster = v.(string)
	}
	if v, ok := m["dashboard"]; ok {
		l.Dashboard = v.(string)
	}
	if v, ok := m["instance-pool"]; ok {
		l.InstancePool = v.(string)
	}
	if v, ok := m["job"]; ok {
		l.Job = v.(string)
	}
	if v, ok := m["metastore"]; ok {
		l.Metastore = v.(string)
	}
	if v, ok := m["pipeline"]; ok {
		l.Pipeline = v.(string)
	}
	if v, ok := m["query"]; ok {
		l.Query = v.(string)
	}
	if v, ok := m["service-principal"]; ok {
		l.ServicePrincipal = v.(string)
	}
	if v, ok := m["warehouse"]; ok {
		l.Warehouse = v.(string)
	}

	return l
}

func (l *Lookup) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	if err := l.validate(); err != nil {
		return "", err
	}

	resolvers := resolvers()
	if l.Alert != "" {
		return resolvers["alert"](ctx, w, l.Alert)
	}
	if l.ClusterPolicy != "" {
		return resolvers["cluster-policy"](ctx, w, l.ClusterPolicy)
	}
	if l.Cluster != "" {
		return resolvers["cluster"](ctx, w, l.Cluster)
	}
	if l.Dashboard != "" {
		return resolvers["dashboard"](ctx, w, l.Dashboard)
	}
	if l.InstancePool != "" {
		return resolvers["instance-pool"](ctx, w, l.InstancePool)
	}
	if l.Job != "" {
		return resolvers["job"](ctx, w, l.Job)
	}
	if l.Metastore != "" {
		return resolvers["metastore"](ctx, w, l.Metastore)
	}
	if l.Pipeline != "" {
		return resolvers["pipeline"](ctx, w, l.Pipeline)
	}
	if l.Query != "" {
		return resolvers["query"](ctx, w, l.Query)
	}
	if l.ServicePrincipal != "" {
		return resolvers["service-principal"](ctx, w, l.ServicePrincipal)
	}
	if l.Warehouse != "" {
		return resolvers["warehouse"](ctx, w, l.Warehouse)
	}

	return "", fmt.Errorf("no valid lookup fields provided")
}

func (l *Lookup) String() string {
	if l.Alert != "" {
		return fmt.Sprintf("alert: %s", l.Alert)
	}
	if l.ClusterPolicy != "" {
		return fmt.Sprintf("cluster-policy: %s", l.ClusterPolicy)
	}
	if l.Cluster != "" {
		return fmt.Sprintf("cluster: %s", l.Cluster)
	}
	if l.Dashboard != "" {
		return fmt.Sprintf("dashboard: %s", l.Dashboard)
	}
	if l.InstancePool != "" {
		return fmt.Sprintf("instance-pool: %s", l.InstancePool)
	}
	if l.Job != "" {
		return fmt.Sprintf("job: %s", l.Job)
	}
	if l.Metastore != "" {
		return fmt.Sprintf("metastore: %s", l.Metastore)
	}
	if l.Pipeline != "" {
		return fmt.Sprintf("pipeline: %s", l.Pipeline)
	}
	if l.Query != "" {
		return fmt.Sprintf("query: %s", l.Query)
	}
	if l.ServicePrincipal != "" {
		return fmt.Sprintf("service-principal: %s", l.ServicePrincipal)
	}
	if l.Warehouse != "" {
		return fmt.Sprintf("warehouse: %s", l.Warehouse)
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

type resolverFunc func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error)

func resolvers() map[string](resolverFunc) {
	resolvers := make(map[string](resolverFunc), 0)
	resolvers["alert"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Alerts.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["cluster-policy"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.ClusterPolicies.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.PolicyId), nil
	}
	resolvers["cluster"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Clusters.GetByClusterName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.ClusterId), nil
	}
	resolvers["dashboard"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Dashboards.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["instance-pool"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.InstancePools.GetByInstancePoolName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.InstancePoolId), nil
	}
	resolvers["job"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Jobs.GetBySettingsName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.JobId), nil
	}
	resolvers["metastore"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Metastores.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.MetastoreId), nil
	}
	resolvers["pipeline"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Pipelines.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.PipelineId), nil
	}
	resolvers["query"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Queries.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["service-principal"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.ServicePrincipals.GetByDisplayName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["warehouse"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Warehouses.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}

	return resolvers
}
