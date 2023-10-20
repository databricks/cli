// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package resolvers

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
)

type ResolverFunc func(ctx context.Context, b *bundle.Bundle, name string) (string, error)

func Resolvers() map[string](ResolverFunc) {
	resolvers := make(map[string](ResolverFunc), 0)

	resolvers["alerts"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Alerts.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["cluster-policies"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.ClusterPolicies.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.PolicyId), nil
	}
	resolvers["clusters"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Clusters.GetByClusterName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.ClusterId), nil
	}
	resolvers["dashboards"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Dashboards.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["instance-pools"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.InstancePools.GetByInstancePoolName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.InstancePoolId), nil
	}
	resolvers["jobs"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Jobs.GetBySettingsName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.JobId), nil
	}
	resolvers["metastores"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Metastores.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.MetastoreId), nil
	}
	resolvers["pipelines"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Pipelines.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.PipelineId), nil
	}
	resolvers["queries"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Queries.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["warehouses"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Warehouses.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}

	return resolvers
}
