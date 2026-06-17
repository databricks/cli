package resources

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/stretchr/testify/assert"
)

func TestSyncedDatabaseTableGetName(t *testing.T) {
	s := &SyncedDatabaseTable{
		SyncedDatabaseTable: database.SyncedDatabaseTable{
			Name: "${resources.database_catalogs.cat.name}.public.t",
		},
	}

	// Before deploy the configured name (with its unresolved reference) is used.
	assert.Equal(t, "${resources.database_catalogs.cat.name}.public.t", s.GetName())

	// After deploy the resolved name is loaded into the id and preferred.
	s.ID = "my_catalog.public.t"
	assert.Equal(t, "my_catalog.public.t", s.GetName())
}

func TestSyncedDatabaseTableInitializeURL(t *testing.T) {
	baseURL := url.URL{Scheme: "https", Host: "example.com"}

	s := &SyncedDatabaseTable{
		SyncedDatabaseTable: database.SyncedDatabaseTable{
			Name: "${resources.database_catalogs.cat.name}.public.t",
		},
	}

	// An unresolved reference is not a three-part name, so no URL is produced.
	s.InitializeURL(baseURL)
	assert.Empty(t, s.URL)

	// The resolved three-part id yields a URL.
	s.ID = "my_catalog.public.t"
	s.InitializeURL(baseURL)
	assert.Contains(t, s.URL, "my_catalog.public.t")
}
