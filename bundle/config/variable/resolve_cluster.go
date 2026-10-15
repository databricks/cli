package variable

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/sdkclients"
)

type resolveCluster struct {
	name string
}

type listedCluster struct {
	ClusterId   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
}

type listClustersResponse struct {
	Clusters      []listedCluster `json:"clusters"`
	NextPageToken string          `json:"next_page_token"`
}

// We added a custom resolver for the cluster to add filtering for the cluster source when we list all clusters.
// Without the filtering listing could take a very long time (5-10 mins) which leads to lookup timeouts.
// sdk-go does not expose the filter, so the list endpoint is called directly.
func (l resolveCluster) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	api, err := apiclient.New(w.Config)
	if err != nil {
		return "", err
	}

	var result []listedCluster
	pageToken := ""
	for {
		query := url.Values{"filter_by.cluster_sources": {"API", "UI"}}
		if pageToken != "" {
			query.Set("page_token", pageToken)
		}
		var resp listClustersResponse
		err := api.Do(ctx, http.MethodGet, "/api/2.1/clusters/list", auth.WorkspaceIDHeaders(w.Config), nil, query, &resp)
		if err != nil {
			return "", err
		}
		result = append(result, resp.Clusters...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	tmp := map[string][]listedCluster{}
	for _, v := range result {
		key := v.ClusterName
		tmp[key] = append(tmp[key], v)
	}

	name := l.name
	alternatives, ok := tmp[name]
	if !ok || len(alternatives) == 0 {
		return "", fmt.Errorf("cluster named '%s' does not exist", name)
	}
	if len(alternatives) > 1 {
		return "", fmt.Errorf("there are %d instances of clusters named '%s'", len(alternatives), name)
	}
	return alternatives[0].ClusterId, nil
}

func (l resolveCluster) String() string {
	return "cluster: " + l.name
}
