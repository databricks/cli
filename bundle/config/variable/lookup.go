// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package variable

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go"
)

type Lookup struct {
	alert string

	clusterPolicy string

	cluster string

	dashboard string

	instancePool string

	job string

	metastore string

	pipeline string

	query string

	warehouse string
}

func LookupFromMap(m map[string]any) *Lookup {
	l := &Lookup{}
	if v, ok := m["alert"]; ok {
		l.alert = v.(string)
	}
	if v, ok := m["cluster-policy"]; ok {
		l.clusterPolicy = v.(string)
	}
	if v, ok := m["cluster"]; ok {
		l.cluster = v.(string)
	}
	if v, ok := m["dashboard"]; ok {
		l.dashboard = v.(string)
	}
	if v, ok := m["instance-pool"]; ok {
		l.instancePool = v.(string)
	}
	if v, ok := m["job"]; ok {
		l.job = v.(string)
	}
	if v, ok := m["metastore"]; ok {
		l.metastore = v.(string)
	}
	if v, ok := m["pipeline"]; ok {
		l.pipeline = v.(string)
	}
	if v, ok := m["query"]; ok {
		l.query = v.(string)
	}
	if v, ok := m["warehouse"]; ok {
		l.warehouse = v.(string)
	}

	return l
}

func (l *Lookup) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	if err := l.validate(); err != nil {
		return "", err
	}

	resolvers := resolvers()
	if l.alert != "" {
		return resolvers["alert"](ctx, w, l.alert)
	}
	if l.clusterPolicy != "" {
		return resolvers["cluster-policy"](ctx, w, l.clusterPolicy)
	}
	if l.cluster != "" {
		return resolvers["cluster"](ctx, w, l.cluster)
	}
	if l.dashboard != "" {
		return resolvers["dashboard"](ctx, w, l.dashboard)
	}
	if l.instancePool != "" {
		return resolvers["instance-pool"](ctx, w, l.instancePool)
	}
	if l.job != "" {
		return resolvers["job"](ctx, w, l.job)
	}
	if l.metastore != "" {
		return resolvers["metastore"](ctx, w, l.metastore)
	}
	if l.pipeline != "" {
		return resolvers["pipeline"](ctx, w, l.pipeline)
	}
	if l.query != "" {
		return resolvers["query"](ctx, w, l.query)
	}
	if l.warehouse != "" {
		return resolvers["warehouse"](ctx, w, l.warehouse)
	}

	return "", fmt.Errorf("no valid lookup fields provided")
}

func (l *Lookup) String() string {
	if l.alert != "" {
		return fmt.Sprintf("alert: %s", l.alert)
	}
	if l.clusterPolicy != "" {
		return fmt.Sprintf("cluster-policy: %s", l.clusterPolicy)
	}
	if l.cluster != "" {
		return fmt.Sprintf("cluster: %s", l.cluster)
	}
	if l.dashboard != "" {
		return fmt.Sprintf("dashboard: %s", l.dashboard)
	}
	if l.instancePool != "" {
		return fmt.Sprintf("instance-pool: %s", l.instancePool)
	}
	if l.job != "" {
		return fmt.Sprintf("job: %s", l.job)
	}
	if l.metastore != "" {
		return fmt.Sprintf("metastore: %s", l.metastore)
	}
	if l.pipeline != "" {
		return fmt.Sprintf("pipeline: %s", l.pipeline)
	}
	if l.query != "" {
		return fmt.Sprintf("query: %s", l.query)
	}
	if l.warehouse != "" {
		return fmt.Sprintf("warehouse: %s", l.warehouse)
	}

	return ""
}

func (l *Lookup) validate() error {
	// Validate that only one field is set
	count := 0
	if l.alert != "" {
		count++
	}
	if l.clusterPolicy != "" {
		count++
	}
	if l.cluster != "" {
		count++
	}
	if l.dashboard != "" {
		count++
	}
	if l.instancePool != "" {
		count++
	}
	if l.job != "" {
		count++
	}
	if l.metastore != "" {
		count++
	}
	if l.pipeline != "" {
		count++
	}
	if l.query != "" {
		count++
	}
	if l.warehouse != "" {
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
	resolvers["warehouse"] = func(ctx context.Context, w *databricks.WorkspaceClient, name string) (string, error) {
		entity, err := w.Warehouses.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}

	return resolvers
}
