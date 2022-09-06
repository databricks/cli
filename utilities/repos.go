package utilities

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

// Remove once this function is in go sdk
// https://github.com/databricks/databricks-sdk-go/issues/58
// Tracked in : https://github.com/databricks/bricks/issues/26
func GetAllRepos(ctx context.Context, wsc *workspaces.WorkspacesClient, pathPrefix string) (resultRepos []repos.RepoInfo, err error) {
	nextPageToken := ""
	for {
		listReposResponse, err := wsc.Repos.List(ctx, repos.ListRequest{
			PathPrefix:    pathPrefix,
			NextPageToken: nextPageToken,
		})
		if err != nil {
			break
		}
		resultRepos = append(resultRepos, listReposResponse.Repos...)
		if nextPageToken == "" {
			break
		}
	}
	return
}
