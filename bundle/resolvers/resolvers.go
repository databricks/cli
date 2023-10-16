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
	resolvers["connections"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Connections.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.FullName), nil
	}
	resolvers["dashboards"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Dashboards.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["data-sources"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.DataSources.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["git-credentials"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.GitCredentials.GetByGitProvider(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.CredentialId), nil
	}
	resolvers["global-init-scripts"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.GlobalInitScripts.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.ScriptId), nil
	}
	resolvers["groups"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Groups.GetByDisplayName(ctx, name)
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
	resolvers["ip-access-lists"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.IpAccessLists.GetByLabel(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.ListId), nil
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
	resolvers["registered-models"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.RegisteredModels.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.FullName), nil
	}
	resolvers["repos"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Repos.GetByPath(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["schemas"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Schemas.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.FullName), nil
	}
	resolvers["service-principals"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.ServicePrincipals.GetByDisplayName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["tables"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Tables.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.TableId), nil
	}
	resolvers["token-management"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.TokenManagement.GetByComment(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.TokenId), nil
	}
	resolvers["tokens"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Tokens.GetByComment(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.TokenId), nil
	}
	resolvers["users"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Users.GetByUserName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["volumes"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Volumes.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.VolumeId), nil
	}
	resolvers["warehouses"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Warehouses.GetByName(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.Id), nil
	}
	resolvers["workspace"] = func(ctx context.Context, b *bundle.Bundle, name string) (string, error) {
		w := b.WorkspaceClient()
		entity, err := w.Workspace.GetByPath(ctx, name)
		if err != nil {
			return "", err
		}

		return fmt.Sprint(entity.ObjectId), nil
	}

	return resolvers
}
