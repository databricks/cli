package appenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type envResponse struct {
	EnvVariables []string `json:"env_variables"`
}

type FetchResult struct {
	EnvVars   []string
	Resources []apps.AppResource
}

type EnvFetcher struct {
	client *databricks.WorkspaceClient
}

func NewEnvFetcher(client *databricks.WorkspaceClient) *EnvFetcher {
	return &EnvFetcher{client: client}
}

func (f *EnvFetcher) Fetch(ctx context.Context, appName string) (*FetchResult, error) {
	app, err := f.client.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	if app.Url == "" {
		return nil, errors.New("app URL is empty")
	}

	cfg := cmdctx.ConfigUsed(ctx)
	if cfg == nil {
		return nil, errors.New("missing workspace configuration")
	}

	tokenSource := cfg.GetTokenSource()
	if tokenSource == nil {
		return nil, errors.New("configuration does not support OAuth tokens")
	}

	token, err := tokenSource.Token(ctx)
	if err != nil {
		return nil, err
	}

	envURL, err := url.Parse(app.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse app URL: %w", err)
	}
	envURL.Path = envURL.Path + "/.runtime/env"

	req, err := http.NewRequestWithContext(ctx, "GET", envURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch env vars: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch env vars: status %d", resp.StatusCode)
	}

	var envResp envResponse
	if err := json.NewDecoder(resp.Body).Decode(&envResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &FetchResult{
		EnvVars:   envResp.EnvVariables,
		Resources: app.Resources,
	}, nil
}
