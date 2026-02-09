package prompt

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

// ListItem is a generic item for resource pickers (id and display label).
type ListItem struct {
	ID    string
	Label string
}

func workspaceClient(ctx context.Context) (*databricks.WorkspaceClient, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}
	return w, nil
}

// ListSecrets returns secret scopes as selectable items (id = scope name).
func ListSecrets(ctx context.Context) ([]ListItem, error) {
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

// ListVolumes returns UC volumes as selectable items (id = full name catalog.schema.volume).
func ListVolumes(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	var out []ListItem
	catIter := w.Catalogs.List(ctx, catalog.ListCatalogsRequest{})
	catalogs, err := listing.ToSlice(ctx, catIter)
	if err != nil {
		return nil, err
	}
	const maxSchemas = 50
	for _, cat := range catalogs {
		schemaIter := w.Schemas.List(ctx, catalog.ListSchemasRequest{CatalogName: cat.Name})
		schemas, err := listing.ToSlice(ctx, schemaIter)
		if err != nil {
			continue
		}
		for _, sch := range schemas {
			volIter := w.Volumes.List(ctx, catalog.ListVolumesRequest{
				CatalogName: cat.Name,
				SchemaName:  sch.Name,
			})
			vols, err := listing.ToSlice(ctx, volIter)
			if err != nil {
				continue
			}
			for _, v := range vols {
				fullName := fmt.Sprintf("%s.%s.%s", cat.Name, sch.Name, v.Name)
				out = append(out, ListItem{ID: fullName, Label: fullName})
			}
		}
		if len(out) >= maxSchemas*10 {
			break
		}
	}
	return out, nil
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

// ListFunctions returns UC functions as selectable items (id = full name).
func ListFunctions(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	var out []ListItem
	catIter := w.Catalogs.List(ctx, catalog.ListCatalogsRequest{})
	catalogs, err := listing.ToSlice(ctx, catIter)
	if err != nil {
		return nil, err
	}
	for _, cat := range catalogs {
		schemaIter := w.Schemas.List(ctx, catalog.ListSchemasRequest{CatalogName: cat.Name})
		schemas, err := listing.ToSlice(ctx, schemaIter)
		if err != nil {
			continue
		}
		for _, sch := range schemas {
			fnIter := w.Functions.List(ctx, catalog.ListFunctionsRequest{
				CatalogName: cat.Name,
				SchemaName:  sch.Name,
			})
			fns, err := listing.ToSlice(ctx, fnIter)
			if err != nil {
				continue
			}
			for _, f := range fns {
				fullName := f.FullName
				if fullName == "" {
					fullName = fmt.Sprintf("%s.%s.%s", cat.Name, sch.Name, f.Name)
				}
				out = append(out, ListItem{ID: fullName, Label: fullName})
			}
		}
	}
	return out, nil
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

// ListDatabases returns UC catalogs as selectable items (id = catalog name).
func ListDatabases(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Catalogs.List(ctx, catalog.ListCatalogsRequest{})
	catalogs, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(catalogs))
	for _, c := range catalogs {
		out = append(out, ListItem{ID: c.Name, Label: c.Name})
	}
	return out, nil
}

// ListGenieSpaces returns Genie spaces as selectable items.
func ListGenieSpaces(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := w.Genie.ListSpaces(ctx, dashboards.GenieListSpacesRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(resp.Spaces))
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

// ListAppsItems returns apps as ListItems (id = app name).
func ListAppsItems(ctx context.Context) ([]ListItem, error) {
	w, err := workspaceClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := w.Apps.List(ctx, apps.ListAppsRequest{})
	appList, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(appList))
	for _, a := range appList {
		label := a.Name
		if a.Description != "" {
			label = a.Name + " — " + a.Description
		}
		out = append(out, ListItem{ID: a.Name, Label: label})
	}
	return out, nil
}
