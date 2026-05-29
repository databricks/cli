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
	"github.com/databricks/databricks-sdk-go/service/serving"
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
		volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, v.Name)
		out = append(out, ListItem{ID: volumePath, Label: v.Name})
	}
	return capResults(out), nil
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
		if b.Status != nil {
			if b.Status.Default {
				label += " (default)"
			}
			if b.Status.IsProtected {
				label += " (protected)"
			}
			if b.Status.CurrentState == postgres.BranchStatusStateArchived {
				label += " (archived)"
			}
		}
		out = append(out, ListItem{ID: b.Name, Label: label})
	}
	return out, nil
}

// ListPostgresDatabases returns databases within a Lakebase Autoscaling branch as selectable items.
func ListPostgresDatabases(ctx context.Context, branchName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Postgres.ListDatabases(ctx, postgres.ListDatabasesRequest{Parent: branchName})
	databases, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(databases))
	for _, db := range databases {
		label := extractIDFromName(db.Name, "databases")
		if db.Status != nil && db.Status.PostgresDatabase != "" {
			label = db.Status.PostgresDatabase
		}
		out = append(out, ListItem{ID: db.Name, Label: label})
	}
	return out, nil
}

// ListPostgresEndpoints returns endpoints for a branch as raw Endpoint objects.
// Returns raw objects (not ListItem) since we need multiple fields (Name, Status.Hosts.Host).
func ListPostgresEndpoints(ctx context.Context, branchName string) ([]postgres.Endpoint, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Postgres.ListEndpoints(ctx, postgres.ListEndpointsRequest{Parent: branchName})
	return listing.ToSlice(ctx, iter)
}

// ---------------------------------------------------------------------------
// Paged lister constructors — return a PagedFetcher that loads pageSize items
// at a time, keeping the SDK iterator alive for incremental "Load more".
// ---------------------------------------------------------------------------

// ListSQLWarehouses lists SQL warehouses as a paged result.
func ListSQLWarehouses(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Warehouses.List(ctx, sql.ListWarehousesRequest{PageSize: pageSize})
	mapFn := func(wh sql.EndpointInfo) ListItem {
		label := wh.Name
		if wh.State != "" {
			label = fmt.Sprintf("%s (%s)", wh.Name, wh.State)
		}
		return ListItem{ID: wh.Id, Label: label}
	}
	items, hasMore, err := collectN(ctx, iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, pageSize, mapFn)
		},
	}, nil
}

// ListJobs lists jobs as a paged result.
func ListJobs(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	apiLimit := min(pageSize, 100)
	iter := w.Jobs.List(ctx, jobs.ListJobsRequest{Limit: apiLimit})
	mapFn := func(j jobs.BaseJob) ListItem {
		label := j.Settings.Name
		id := strconv.FormatInt(j.JobId, 10)
		if label == "" {
			label = id
		}
		return ListItem{ID: id, Label: label}
	}
	items, hasMore, err := collectN(ctx, iter, apiLimit, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, apiLimit, mapFn)
		},
	}, nil
}

// SearchJobs performs a server-side search for jobs by name (exact, case-insensitive).
func SearchJobs(ctx context.Context, name string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Jobs.List(ctx, jobs.ListJobsRequest{Name: name})
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

// ListServingEndpoints lists serving endpoints as a paged result.
func ListServingEndpoints(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.ServingEndpoints.List(ctx)
	mapFn := func(e serving.ServingEndpoint) ListItem {
		name := e.Name
		if name == "" {
			name = e.Id
		}
		return ListItem{ID: e.Name, Label: name}
	}
	items, hasMore, err := collectN(ctx, iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, pageSize, mapFn)
		},
	}, nil
}

// ListExperiments lists MLflow experiments as a paged result.
func ListExperiments(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Experiments.ListExperiments(ctx, ml.ListExperimentsRequest{MaxResults: int64(pageSize)})
	mapFn := func(e ml.Experiment) ListItem {
		label := e.Name
		if label == "" {
			label = e.ExperimentId
		}
		return ListItem{ID: e.ExperimentId, Label: label}
	}
	items, hasMore, err := collectN(ctx, iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, pageSize, mapFn)
		},
	}, nil
}

// ListGenieSpaces lists Genie spaces as a paged result (manual pagination).
func ListGenieSpaces(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	var nextToken string
	fetchPage := func(ctx context.Context) ([]ListItem, bool, error) {
		req := dashboards.GenieListSpacesRequest{PageToken: nextToken, PageSize: pageSize}
		resp, err := w.Genie.ListSpaces(ctx, req)
		if err != nil {
			return nil, false, err
		}
		items := make([]ListItem, 0, len(resp.Spaces))
		for _, s := range resp.Spaces {
			id := s.SpaceId
			label := s.Title
			if label == "" {
				label = s.Description
			}
			if label == "" {
				label = id
			}
			items = append(items, ListItem{ID: id, Label: label})
		}
		nextToken = resp.NextPageToken
		return items, nextToken != "", nil
	}
	items, hasMore, err := fetchPage(ctx)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:    items,
		HasMore:  hasMore,
		loadMore: fetchPage,
	}, nil
}

// ListConnections lists UC connections as a paged result.
func ListConnections(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Connections.List(ctx, catalog.ListConnectionsRequest{MaxResults: pageSize})
	mapFn := func(c catalog.ConnectionInfo) ListItem {
		name := c.Name
		if name == "" {
			name = c.FullName
		}
		return ListItem{ID: c.FullName, Label: name}
	}
	items, hasMore, err := collectN(ctx, iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, pageSize, mapFn)
		},
	}, nil
}

// ListVectorSearchIndexes lists vector search indexes as a paged result.
// Unlike other listers, this eagerly loads all indexes across all endpoints
// because the API requires a two-level query (endpoint -> indexes).
// Incremental loading isn't feasible without restructuring the picker into a
// two-step flow. The result is capped at maxTotalResults.
func ListVectorSearchIndexes(ctx context.Context) (*PagedFetcher, error) {
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
	capped := false
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
			if len(out) >= maxTotalResults {
				capped = true
				break
			}
		}
		if capped {
			break
		}
	}
	return &PagedFetcher{Items: out, Capped: capped}, nil
}

// ---------------------------------------------------------------------------
// First-step paged constructors — used to prefetch the initial picker of
// multi-step resource prompts (catalog → schema → resource, etc.).
// ---------------------------------------------------------------------------

// ListCatalogs lists UC catalogs as a paged result (first step of volume/function pickers).
func ListCatalogs(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Catalogs.List(ctx, catalog.ListCatalogsRequest{MaxResults: pageSize})
	mapFn := func(c catalog.CatalogInfo) ListItem {
		return ListItem{ID: c.Name, Label: c.Name}
	}
	items, hasMore, err := collectN(ctx, iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, pageSize, mapFn)
		},
	}, nil
}

// ListDatabaseInstances lists Lakebase database instances as a paged result.
func ListDatabaseInstances(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	apiLimit := min(pageSize, 100)
	iter := w.Database.ListDatabaseInstances(ctx, database.ListDatabaseInstancesRequest{PageSize: apiLimit})
	mapFn := func(inst database.DatabaseInstance) ListItem {
		return ListItem{ID: inst.Name, Label: inst.Name}
	}
	items, hasMore, err := collectN(ctx, iter, apiLimit, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, apiLimit, mapFn)
		},
	}, nil
}

// ListPostgresProjects lists Lakebase Autoscaling projects as a paged result.
func ListPostgresProjects(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	apiLimit := min(pageSize, 100)
	iter := w.Postgres.ListProjects(ctx, postgres.ListProjectsRequest{PageSize: apiLimit})
	mapFn := func(p postgres.Project) ListItem {
		label := p.Name
		if p.Status != nil && p.Status.DisplayName != "" {
			label = p.Status.DisplayName
		}
		return ListItem{ID: p.Name, Label: label}
	}
	items, hasMore, err := collectN(ctx, iter, apiLimit, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, iter, apiLimit, mapFn)
		},
	}, nil
}
