// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package variable

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/databricks-sdk-go"
)

type Lookup struct {
	Field string `json:"field,omitempty"`

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
	if v, ok := m["cluster_policy"]; ok {
		l.ClusterPolicy = v.(string)
	}
	if v, ok := m["cluster"]; ok {
		l.Cluster = v.(string)
	}
	if v, ok := m["dashboard"]; ok {
		l.Dashboard = v.(string)
	}
	if v, ok := m["instance_pool"]; ok {
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
	if v, ok := m["service_principal"]; ok {
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

	r := allResolvers()
	if l.Alert != "" {
		fieldString := "Id"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Alert(ctx, w, l.Alert, fieldString)
	}
	if l.ClusterPolicy != "" {
		fieldString := "PolicyId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.ClusterPolicy(ctx, w, l.ClusterPolicy, fieldString)
	}
	if l.Cluster != "" {
		fieldString := "ClusterId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Cluster(ctx, w, l.Cluster, fieldString)
	}
	if l.Dashboard != "" {
		fieldString := "Id"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Dashboard(ctx, w, l.Dashboard, fieldString)
	}
	if l.InstancePool != "" {
		fieldString := "InstancePoolId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.InstancePool(ctx, w, l.InstancePool, fieldString)
	}
	if l.Job != "" {
		fieldString := "JobId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Job(ctx, w, l.Job, fieldString)
	}
	if l.Metastore != "" {
		fieldString := "MetastoreId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Metastore(ctx, w, l.Metastore, fieldString)
	}
	if l.Pipeline != "" {
		fieldString := "PipelineId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Pipeline(ctx, w, l.Pipeline, fieldString)
	}
	if l.Query != "" {
		fieldString := "Id"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Query(ctx, w, l.Query, fieldString)
	}
	if l.ServicePrincipal != "" {
		fieldString := "ApplicationId"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.ServicePrincipal(ctx, w, l.ServicePrincipal, fieldString)
	}
	if l.Warehouse != "" {
		fieldString := "Id"
		if l.Field != "" {
			fieldString = l.Field
		}
		return r.Warehouse(ctx, w, l.Warehouse, fieldString)
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

type resolverFunc func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error)
type resolvers struct {
	Alert            resolverFunc
	ClusterPolicy    resolverFunc
	Cluster          resolverFunc
	Dashboard        resolverFunc
	InstancePool     resolverFunc
	Job              resolverFunc
	Metastore        resolverFunc
	Pipeline         resolverFunc
	Query            resolverFunc
	ServicePrincipal resolverFunc
	Warehouse        resolverFunc
}

func allResolvers() *resolvers {
	r := &resolvers{}
	r.Alert = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Alerts.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.ClusterPolicy = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.ClusterPolicies.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Cluster = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Clusters.GetByClusterName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Dashboard = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Dashboards.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.InstancePool = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.InstancePools.GetByInstancePoolName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Job = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Jobs.GetBySettingsName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Metastore = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Metastores.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Pipeline = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Pipelines.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Query = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Queries.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.ServicePrincipal = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.ServicePrincipals.GetByDisplayName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}
	r.Warehouse = func(ctx context.Context, w *databricks.WorkspaceClient, name string, field string) (string, error) {
		entity, err := w.Warehouses.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		s := reflect.ValueOf(entity)
		v := s.Elem().FieldByName(field)
		if !v.IsValid() {
			return "", fmt.Errorf("field %s does not exist", field)
		}
		return fmt.Sprint(v.Interface()), nil
	}

	return r
}
