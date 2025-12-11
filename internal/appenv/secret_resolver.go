package appenv

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/apps/runlocal"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type SecretResolver struct {
	client *databricks.WorkspaceClient
}

func NewSecretResolver(client *databricks.WorkspaceClient) *SecretResolver {
	return &SecretResolver{client: client}
}

func (r *SecretResolver) Resolve(ctx context.Context, envVars []string, appResources []apps.AppResource, spec *runlocal.AppSpec) []string {
	resolved := make([]string, 0, len(envVars))

	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			resolved = append(resolved, envVar)
			continue
		}

		envName := parts[0]
		envValue := parts[1]

		if envValue != "***" {
			resolved = append(resolved, envVar)
			continue
		}

		var valueFrom string
		for _, specEnv := range spec.EnvVars {
			if specEnv.Name == envName && specEnv.ValueFrom != nil {
				valueFrom = *specEnv.ValueFrom
				break
			}
		}

		if valueFrom == "" {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: env var %s has *** value but no valueFrom in spec, keeping as ***", envName))
			resolved = append(resolved, envVar)
			continue
		}

		var secretResource *apps.AppResourceSecret
		for _, resource := range appResources {
			if resource.Name == valueFrom && resource.Secret != nil {
				secretResource = resource.Secret
				break
			}
		}

		if secretResource == nil {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: env var %s references resource '%s' but it's not a secret resource, keeping as ***", envName, valueFrom))
			resolved = append(resolved, envVar)
			continue
		}

		cmdio.LogString(ctx, fmt.Sprintf("Resolving secret for: %s", envName))
		secretResp, err := r.client.Secrets.GetSecret(ctx, workspace.GetSecretRequest{
			Scope: secretResource.Scope,
			Key:   secretResource.Key,
		})
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: failed to resolve secret %s: %v, keeping as ***", envName, err))
			resolved = append(resolved, envVar)
			continue
		}

		decodedValue, err := base64.StdEncoding.DecodeString(secretResp.Value)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: failed to decode secret %s: %v, keeping as ***", envName, err))
			resolved = append(resolved, envVar)
			continue
		}

		cmdio.LogString(ctx, fmt.Sprintf("Successfully resolved secret from scope '%s', key '%s'", secretResource.Scope, secretResource.Key))
		resolved = append(resolved, fmt.Sprintf("%s=%s", envName, string(decodedValue)))
	}

	return resolved
}
