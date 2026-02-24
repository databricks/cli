package prompt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// maxListResults caps the number of items returned by listers to avoid
// very long API traversals on large workspaces.
const maxListResults = 500

// ListItem is a generic item for resource pickers (id and display label).
type ListItem struct {
	ID    string
	Label string
}

// capResults truncates a slice to maxListResults.
func capResults(items []ListItem) []ListItem {
	if len(items) > maxListResults {
		return items[:maxListResults]
	}
	return items
}

func workspaceClient(ctx context.Context) (*databricks.WorkspaceClient, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}
	return w, nil
}

// ListSecretScopes returns secret scopes as selectable items.
func ListSecretScopes(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Secrets.ListScopes(ctx)
	scopes, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, ListItem{ID: s.Name, Label: s.Name})
	}
	return out, nil
}

// ListSecretKeys returns secret keys within a scope as selectable items.
func ListSecretKeys(ctx context.Context, scope string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Secrets.ListSecrets(ctx, workspace.ListSecretsRequest{Scope: scope})
	keys, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, ListItem{ID: k.Key, Label: k.Key})
	}
	return out, nil
}

// ListJobs returns jobs as selectable items.
func ListJobs(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Jobs.List(ctx, jobs.ListJobsRequest{})
	jobList, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(jobList))
	for _, j := range jobList {
		label := j.Settings.Name
		id := strconv.FormatInt(j.JobId, 10)
		if label == "" {
			label = id
		}
		out = append(out, ListItem{ID: id, Label: label})
	}
	return out, nil
}

// ListSQLWarehousesItems returns SQL warehouses as ListItems (reuses same API as ListSQLWarehouses).
func ListSQLWarehousesItems(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Warehouses.List(ctx, sql.ListWarehousesRequest{})
	whs, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(whs))
	for _, wh := range whs {
		label := wh.Name
		if wh.State != "" {
			label = fmt.Sprintf("%s (%s)", wh.Name, wh.State)
		}
		out = append(out, ListItem{ID: wh.Id, Label: label})
	}
	return out, nil
}

// ListServingEndpoints returns serving endpoints as selectable items.
func ListServingEndpoints(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.ServingEndpoints.List(ctx)
	endpoints, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(endpoints))
	for _, e := range endpoints {
		name := e.Name
		if name == "" {
			name = e.Id
		}
		out = append(out, ListItem{ID: e.Id, Label: name})
	}
	return out, nil
}

// ListCatalogs returns UC catalogs as selectable items.
func ListCatalogs(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	catIter := w.Catalogs.List(ctx, catalog.ListCatalogsRequest{})
	cats, err := listing.ToSlice(ctx, catIter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(cats), maxListResults))
	for _, c := range cats {
		out = append(out, ListItem{ID: c.Name, Label: c.Name})
	}
	return capResults(out), nil
}

// ListSchemas returns UC schemas within a catalog as selectable items.
func ListSchemas(ctx context.Context, catalogName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	schemaIter := w.Schemas.List(ctx, catalog.ListSchemasRequest{CatalogName: catalogName})
	schemas, err := listing.ToSlice(ctx, schemaIter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(schemas), maxListResults))
	for _, s := range schemas {
		out = append(out, ListItem{ID: s.Name, Label: s.Name})
	}
	return capResults(out), nil
}

// ListVolumesInSchema returns UC volumes within a catalog.schema as selectable items.
func ListVolumesInSchema(ctx context.Context, catalogName, schemaName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	volIter := w.Volumes.List(ctx, catalog.ListVolumesRequest{
		CatalogName: catalogName,
		SchemaName:  schemaName,
	})
	vols, err := listing.ToSlice(ctx, volIter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(vols), maxListResults))
	for _, v := range vols {
		fullName := fmt.Sprintf("%s.%s.%s", catalogName, schemaName, v.Name)
		out = append(out, ListItem{ID: fullName, Label: v.Name})
	}
	return capResults(out), nil
}

// ListVectorSearchIndexes returns vector search indexes as selectable items (id = endpoint/index name).
func ListVectorSearchIndexes(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	var out []ListItem
	epIter := w.VectorSearchEndpoints.ListEndpoints(ctx, vectorsearch.ListEndpointsRequest{})
	endpoints, err := listing.ToSlice(ctx, epIter)
	if err != nil {
		return nil, err
	}
	for _, ep := range endpoints {
		indexIter := w.VectorSearchIndexes.ListIndexes(ctx, vectorsearch.ListIndexesRequest{EndpointName: ep.Name})
		indexes, err := listing.ToSlice(ctx, indexIter)
		if err != nil {
			log.Warnf(ctx, "Failed to list indexes for endpoint %q: %v", ep.Name, err)
			continue
		}
		for _, idx := range indexes {
			label := idx.Name
			if label == "" {
				label = ep.Name + "/ (unnamed)"
			}
			id := ep.Name + "/" + idx.Name
			out = append(out, ListItem{ID: id, Label: fmt.Sprintf("%s / %s", ep.Name, label)})
		}
	}
	return out, nil
}

// ListFunctionsInSchema returns UC functions within a catalog.schema as selectable items.
func ListFunctionsInSchema(ctx context.Context, catalogName, schemaName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	fnIter := w.Functions.List(ctx, catalog.ListFunctionsRequest{
		CatalogName: catalogName,
		SchemaName:  schemaName,
	})
	fns, err := listing.ToSlice(ctx, fnIter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(fns), maxListResults))
	for _, f := range fns {
		fullName := f.FullName
		if fullName == "" {
			fullName = fmt.Sprintf("%s.%s.%s", catalogName, schemaName, f.Name)
		}
		out = append(out, ListItem{ID: fullName, Label: f.Name})
	}
	return capResults(out), nil
}

// ListConnections returns UC connections as selectable items.
func ListConnections(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Connections.List(ctx, catalog.ListConnectionsRequest{})
	conns, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(conns))
	for _, c := range conns {
		name := c.Name
		if name == "" {
			name = c.FullName
		}
		out = append(out, ListItem{ID: c.FullName, Label: name})
	}
	return out, nil
}

// ListDatabaseInstances returns Lakebase database instances as selectable items.
func ListDatabaseInstances(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Database.ListDatabaseInstances(ctx, database.ListDatabaseInstancesRequest{})
	instances, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(instances))
	for _, inst := range instances {
		out = append(out, ListItem{ID: inst.Name, Label: inst.Name})
	}
	return out, nil
}

// listDatabasesResponse is the response from the /databases endpoint.
type listDatabasesResponse struct {
	Databases []struct {
		Name               string `json:"name"`
		IsUsableByCustomer bool   `json:"is_usable_by_customer"`
	} `json:"databases"`
}

// ListDatabases returns databases within a Lakebase instance as selectable items.
func ListDatabases(ctx context.Context, instanceName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	api, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	// TODO: use the SDK to list databases once available
	var resp listDatabasesResponse
	path := fmt.Sprintf("/api/2.0/database/instances/%s/databases", url.PathEscape(instanceName))
	headers := map[string]string{"Accept": "application/json"}
	err = api.Do(ctx, http.MethodGet, path, headers, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(resp.Databases))
	for _, db := range resp.Databases {
		if !db.IsUsableByCustomer {
			continue
		}
		out = append(out, ListItem{ID: db.Name, Label: db.Name})
	}
	return out, nil
}

// extractIDFromName extracts the ID segment after a named component in a resource path.
// For example, extractIDFromName("projects/foo/branches/bar", "branches") returns "bar".
func extractIDFromName(name, component string) string {
	parts := strings.Split(name, "/")
	for i := range len(parts) - 1 {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

// ListPostgresProjects returns Lakebase Autoscaling (V2) projects as selectable items.
func ListPostgresProjects(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Postgres.ListProjects(ctx, postgres.ListProjectsRequest{})
	projects, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(projects))
	for _, p := range projects {
		label := p.Name
		if p.Status != nil && p.Status.DisplayName != "" {
			label = p.Status.DisplayName
		}
		out = append(out, ListItem{ID: p.Name, Label: label})
	}
	return out, nil
}

// ListPostgresBranches returns branches within a Lakebase Autoscaling project as selectable items.
func ListPostgresBranches(ctx context.Context, projectName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Postgres.ListBranches(ctx, postgres.ListBranchesRequest{Parent: projectName})
	branches, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(branches))
	for _, b := range branches {
		label := extractIDFromName(b.Name, "branches")
		out = append(out, ListItem{ID: b.Name, Label: label})
	}
	return out, nil
}

// ListGenieSpaces returns Genie spaces as selectable items.
func ListGenieSpaces(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	var out []ListItem
	req := dashboards.GenieListSpacesRequest{}
	for {
		resp, err := w.Genie.ListSpaces(ctx, req)
		if err != nil {
			return nil, err
		}
		for _, s := range resp.Spaces {
			id := s.SpaceId
			label := s.Title
			if label == "" {
				label = s.Description
			}
			if label == "" {
				label = id
			}
			out = append(out, ListItem{ID: id, Label: label})
		}
		if resp.NextPageToken == "" {
			break
		}
		req.PageToken = resp.NextPageToken
	}
	return out, nil
}

// ListExperiments returns MLflow experiments as selectable items.
func ListExperiments(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Experiments.ListExperiments(ctx, ml.ListExperimentsRequest{})
	exps, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(exps))
	for _, e := range exps {
		label := e.Name
		if label == "" {
			label = e.ExperimentId
		}
		out = append(out, ListItem{ID: e.ExperimentId, Label: label})
	}
	return out, nil
}

// TODO: uncomment when bundles support app as an app resource type.
// // ListAppsItems returns apps as ListItems (id = app name).
// func ListAppsItems(ctx context.Context) ([]ListItem, error) {
// 	w, err := workspaceClient(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	iter := w.Apps.List(ctx, apps.ListAppsRequest{})
// 	appList, err := listing.ToSlice(ctx, iter)
// 	if err != nil {
// 		return nil, err
// 	}
// 	out := make([]ListItem, 0, len(appList))
// 	for _, a := range appList {
// 		label := a.Name
// 		out = append(out, ListItem{ID: a.Name, Label: label})
// 	}
// 	return out, nil
// }
