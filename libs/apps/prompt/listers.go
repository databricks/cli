package prompt

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/listing"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/database/v1"
	"github.com/databricks/sdk-go/experiments/v1"
	"github.com/databricks/sdk-go/genie/v1"
	"github.com/databricks/sdk-go/jobs/v2"
	"github.com/databricks/sdk-go/modelserving/v1"
	"github.com/databricks/sdk-go/postgres/v1"
	"github.com/databricks/sdk-go/secrets/v1"
	"github.com/databricks/sdk-go/uc/catalogs/v1"
	"github.com/databricks/sdk-go/uc/connections/v1"
	"github.com/databricks/sdk-go/uc/functions/v1"
	"github.com/databricks/sdk-go/uc/schemas/v1"
	"github.com/databricks/sdk-go/uc/volumes/v1"
	"github.com/databricks/sdk-go/vectorsearch/v1"
	"github.com/databricks/sdk-go/warehouses/v1"
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

func workspaceClient(ctx context.Context) (*sdkclients.Clients, error) {
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
	resp, err := w.Secrets.ListScopes(ctx, secrets.ListScopesRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(resp.Scopes))
	for _, s := range resp.Scopes {
		out = append(out, ListItem{ID: utils.Deref(s.Name), Label: utils.Deref(s.Name)})
	}
	return out, nil
}

// ListSecretKeys returns secret keys within a scope as selectable items.
func ListSecretKeys(ctx context.Context, scope string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := w.Secrets.ListSecrets(ctx, secrets.ListSecretsRequest{Scope: new(scope)})
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(resp.Secrets))
	for _, k := range resp.Secrets {
		out = append(out, ListItem{ID: utils.Deref(k.Key), Label: utils.Deref(k.Key)})
	}
	return out, nil
}

// ListSchemas returns UC schemas within a catalog as selectable items.
func ListSchemas(ctx context.Context, catalogName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	schemas, err := collectAll(w.Schemas.ListSchemasIter(ctx, schemas.ListSchemasRequest{CatalogName: new(catalogName)}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(schemas), maxListResults))
	for _, s := range schemas {
		out = append(out, ListItem{ID: utils.Deref(s.Name), Label: utils.Deref(s.Name)})
	}
	return capResults(out), nil
}

// ListVolumesInSchema returns UC volumes within a catalog.schema as selectable items.
func ListVolumesInSchema(ctx context.Context, catalogName, schemaName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	vols, err := collectAll(w.Volumes.ListVolumesIter(ctx, volumes.ListVolumesRequest{
		CatalogName: new(catalogName),
		SchemaName:  new(schemaName),
	}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(vols), maxListResults))
	for _, v := range vols {
		volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, utils.Deref(v.Name))
		out = append(out, ListItem{ID: volumePath, Label: utils.Deref(v.Name)})
	}
	return capResults(out), nil
}

// ListFunctionsInSchema returns UC functions within a catalog.schema as selectable items.
func ListFunctionsInSchema(ctx context.Context, catalogName, schemaName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	fns, err := collectAll(w.Functions.ListFunctionsIter(ctx, functions.ListFunctionsRequest{
		CatalogName: new(catalogName),
		SchemaName:  new(schemaName),
	}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, min(len(fns), maxListResults))
	for _, f := range fns {
		fullName := utils.Deref(f.FullName)
		if fullName == "" {
			fullName = fmt.Sprintf("%s.%s.%s", catalogName, schemaName, utils.Deref(f.Name))
		}
		out = append(out, ListItem{ID: fullName, Label: utils.Deref(f.Name)})
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
	api, err := apiclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	// TODO: use the SDK to list databases once available
	var resp listDatabasesResponse
	path := fmt.Sprintf("/api/2.0/database/instances/%s/databases", url.PathEscape(instanceName))
	headers := map[string]string{"Accept": "application/json"}
	maps.Copy(headers, auth.WorkspaceIDHeaders(w.Config))
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
	branches, err := collectAll(w.Postgres.ListBranchesIter(ctx, postgres.ListBranchesRequest{Parent: new(projectName)}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(branches))
	for _, b := range branches {
		label := extractIDFromName(utils.Deref(b.Name), "branches")
		if b.Status != nil {
			if utils.Deref(b.Status.Default) {
				label += " (default)"
			}
			if utils.Deref(b.Status.IsProtected) {
				label += " (protected)"
			}
			if b.Status.CurrentState == postgres.BranchStatus_State_Archived {
				label += " (archived)"
			}
		}
		out = append(out, ListItem{ID: utils.Deref(b.Name), Label: label})
	}
	return out, nil
}

// ListPostgresDatabases returns databases within a Lakebase Autoscaling branch as selectable items.
func ListPostgresDatabases(ctx context.Context, branchName string) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	databases, err := collectAll(w.Postgres.ListDatabasesIter(ctx, postgres.ListDatabasesRequest{Parent: new(branchName)}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(databases))
	for _, db := range databases {
		label := extractIDFromName(utils.Deref(db.Name), "databases")
		if db.Status != nil && utils.Deref(db.Status.PostgresDatabase) != "" {
			label = *db.Status.PostgresDatabase
		}
		out = append(out, ListItem{ID: utils.Deref(db.Name), Label: label})
	}
	return out, nil
}

// ListPostgresEndpoints returns endpoints for a branch as raw Endpoint objects.
// Returns raw objects (not ListItem) since we need multiple fields (Name, Status.Hosts.Host).
func ListPostgresEndpoints(ctx context.Context, branchName string) ([]*postgres.Endpoint, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	return collectAll(w.Postgres.ListEndpointsIter(ctx, postgres.ListEndpointsRequest{Parent: new(branchName)}))
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
	iter := listing.FromSeq(w.Warehouses.ListWarehousesIter(ctx, warehouses.ListWarehousesRequest{PageSize: new(pageSize)}))
	mapFn := func(wh *warehouses.EndpointInfo) ListItem {
		label := utils.Deref(wh.Name)
		if wh.State != "" {
			label = fmt.Sprintf("%s (%s)", utils.Deref(wh.Name), wh.State)
		}
		return ListItem{ID: utils.Deref(wh.Id), Label: label}
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
	iter := listing.FromSeq(w.Jobs.ListJobsIter(ctx, jobs.ListJobsRequest{Limit: new(apiLimit)}))
	mapFn := func(j *jobs.BaseJob) ListItem {
		label := jobName(j)
		id := strconv.FormatInt(utils.Deref(j.JobId), 10)
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
	jobList, err := collectAll(w.Jobs.ListJobsIter(ctx, jobs.ListJobsRequest{Name: new(name)}))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(jobList))
	for _, j := range jobList {
		label := jobName(j)
		id := strconv.FormatInt(utils.Deref(j.JobId), 10)
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
	resp, err := w.ServingEndpoints.ListInferenceEndpoints(ctx, modelserving.ListInferenceEndpointsRequest{})
	if err != nil {
		return nil, err
	}
	iter := listing.SliceIterator[modelserving.InferenceEndpoint](resp.Endpoints)
	mapFn := func(e modelserving.InferenceEndpoint) ListItem {
		name := utils.Deref(e.Name)
		if name == "" {
			name = utils.Deref(e.Id)
		}
		return ListItem{ID: utils.Deref(e.Name), Label: name}
	}
	items, hasMore, err := collectN(ctx, &iter, pageSize, mapFn)
	if err != nil {
		return nil, err
	}
	return &PagedFetcher{
		Items:   items,
		HasMore: hasMore,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return collectN(ctx, &iter, pageSize, mapFn)
		},
	}, nil
}

// ListExperiments lists MLflow experiments as a paged result.
func ListExperiments(ctx context.Context) (*PagedFetcher, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := listing.FromSeq(w.Experiments.ListExperimentsIter(ctx, experiments.ListExperimentsRequest{MaxResults: new(int64(pageSize))}))
	mapFn := func(e *experiments.Experiment) ListItem {
		label := utils.Deref(e.Name)
		if label == "" {
			label = utils.Deref(e.ExperimentId)
		}
		return ListItem{ID: utils.Deref(e.ExperimentId), Label: label}
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
		req := genie.GenieListSpacesRequest{PageToken: new(nextToken), PageSize: new(pageSize)}
		resp, err := w.Genie.GenieListSpaces(ctx, req)
		if err != nil {
			return nil, false, err
		}
		items := make([]ListItem, 0, len(resp.Spaces))
		for _, s := range resp.Spaces {
			id := utils.Deref(s.SpaceId)
			label := utils.Deref(s.Title)
			if label == "" {
				label = utils.Deref(s.Description)
			}
			if label == "" {
				label = id
			}
			items = append(items, ListItem{ID: id, Label: label})
		}
		nextToken = utils.Deref(resp.NextPageToken)
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
	iter := listing.FromSeq(w.Connections.ListConnectionsIter(ctx, connections.ListConnectionsRequest{MaxResults: new(pageSize)}))
	mapFn := func(c *connections.ConnectionInfo) ListItem {
		name := utils.Deref(c.Name)
		if name == "" {
			name = utils.Deref(c.FullName)
		}
		return ListItem{ID: utils.Deref(c.FullName), Label: name}
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
	endpoints, err := collectAll(w.VectorSearch.ListEndpointsIter(ctx, vectorsearch.ListEndpointsRequest{}))
	if err != nil {
		return nil, err
	}
	capped := false
	for _, ep := range endpoints {
		epName := utils.Deref(ep.Name)
		indexes, err := collectAll(w.VectorSearch.ListVectorIndexIter(ctx, vectorsearch.ListVectorIndexRequest{EndpointName: ep.Name}))
		if err != nil {
			log.Warnf(ctx, "Failed to list indexes for endpoint %q: %v", epName, err)
			continue
		}
		for _, idx := range indexes {
			label := utils.Deref(idx.Name)
			if label == "" {
				label = epName + "/ (unnamed)"
			}
			id := epName + "/" + utils.Deref(idx.Name)
			out = append(out, ListItem{ID: id, Label: fmt.Sprintf("%s / %s", epName, label)})
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
	iter := listing.FromSeq(w.Catalogs.ListCatalogsIter(ctx, catalogs.ListCatalogsRequest{MaxResults: new(pageSize)}))
	mapFn := func(c *catalogs.CatalogInfo) ListItem {
		return ListItem{ID: utils.Deref(c.Name), Label: utils.Deref(c.Name)}
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
	iter := listing.FromSeq(w.Database.ListDatabaseInstancesIter(ctx, database.ListDatabaseInstancesRequest{PageSize: new(apiLimit)}))
	mapFn := func(inst *database.DatabaseInstance) ListItem {
		return ListItem{ID: utils.Deref(inst.Name), Label: utils.Deref(inst.Name)}
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
	iter := listing.FromSeq(w.Postgres.ListProjectsIter(ctx, postgres.ListProjectsRequest{PageSize: new(apiLimit)}))
	mapFn := func(p *postgres.Project) ListItem {
		label := utils.Deref(p.Name)
		if p.Status != nil && utils.Deref(p.Status.DisplayName) != "" {
			label = *p.Status.DisplayName
		}
		return ListItem{ID: utils.Deref(p.Name), Label: label}
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

// jobName returns the job's configured name, or "" when it has none.
func jobName(j *jobs.BaseJob) string {
	if j.Settings == nil {
		return ""
	}
	return utils.Deref(j.Settings.Name)
}

// collectAll drains an sdk-go list sequence.
func collectAll[T any](seq iter.Seq2[*T, error]) ([]*T, error) {
	var out []*T
	for v, err := range seq {
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
