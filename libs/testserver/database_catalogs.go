package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/database"
)

func (s *FakeWorkspace) DatabaseCatalogCreate(req Request) Response {
	defer s.LockUnlock()()

	databaseCatalog := database.DatabaseCatalog{}
	err := json.Unmarshal(req.Body, &databaseCatalog)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			StatusCode: 400,
		}
	}

	// check that the instance exists:
	found := false
	for _, instance := range s.DatabaseInstances {
		if instance.Name == databaseCatalog.DatabaseInstanceName {
			found = true
			break
		}
	}
	if !found {
		return Response{
			Body:       fmt.Sprintf("database instance with name '%s' not found", databaseCatalog.DatabaseInstanceName),
			StatusCode: 404,
		}
	}

	// The catalog name is unique in the metastore, whichever API created the catalog.
	_, dbExists := s.DatabaseCatalogs[databaseCatalog.Name]
	_, ucExists := s.Catalogs[databaseCatalog.Name]
	if dbExists || ucExists {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "CATALOG_ALREADY_EXISTS",
				"message":    fmt.Sprintf("Catalog '%s' already exists", databaseCatalog.Name),
			},
		}
	}

	// Assign uid like the real backend does
	databaseCatalog.Uid = nextUUID()

	s.DatabaseCatalogs[databaseCatalog.Name] = databaseCatalog

	// The catalog is registered in Unity Catalog, which is the only place its create time shows.
	now := nowMilli()
	s.Catalogs[databaseCatalog.Name] = catalog.CatalogInfo{
		Name:        databaseCatalog.Name,
		FullName:    databaseCatalog.Name,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   creatorOf(req),
		UpdatedBy:   s.CurrentUser().UserName,
		Owner:       s.CurrentUser().UserName,
		MetastoreId: nextUUID(),
		CatalogType: catalog.CatalogTypeManagedOnlineCatalog,
	}

	return Response{
		Body: databaseCatalog,
	}
}

func (s *FakeWorkspace) DatabaseCatalogGet(name string) Response {
	defer s.LockUnlock()()

	catalog, ok := s.DatabaseCatalogs[name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"error_code": "CATALOG_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Catalog '%s' does not exist.", name),
			},
		}
	}

	// create_database_if_not_exists is a write-only field - the real API doesn't return it
	result := catalog
	result.CreateDatabaseIfNotExists = false

	return Response{Body: result}
}

// DatabaseCatalogUpdate models the real Database API, which has no update
// endpoint: PATCH /api/2.0/database/catalogs/{name} returns 501 NOT_IMPLEMENTED.
// The direct engine recreates database_catalogs on any change and never calls
// this; modeling the 501 guards against reintroducing an update path.
func (s *FakeWorkspace) DatabaseCatalogUpdate(req Request, name string) Response {
	return Response{
		StatusCode: 501,
		Body: map[string]string{
			"error_code": "NOT_IMPLEMENTED",
			"message":    "Update Database Catalog is not yet implemented",
		},
	}
}

func (s *FakeWorkspace) DatabaseCatalogDelete(name string) Response {
	defer s.LockUnlock()()

	if _, ok := s.DatabaseCatalogs[name]; !ok {
		return Response{StatusCode: 404}
	}
	delete(s.DatabaseCatalogs, name)
	delete(s.Catalogs, name)
	return Response{}
}
