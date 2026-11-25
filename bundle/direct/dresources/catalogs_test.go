package dresources

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	registerTestConfig("catalogs", &resources.Catalog{
		Name:    "mycatalog",
		Comment: "Test catalog",
		// Note: EnablePredictiveOptimization cannot be set during creation,
		// only during updates. It is not included in the test config.
		// IsolationMode has the same restriction but is applied by DoCreate with a
		// follow-up update, so it is set here to cover that path.
		IsolationMode: catalog.CatalogIsolationModeIsolated,
	})

	registerTestDeps("catalogs.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "catalog",
			FullName:      "mycatalog",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeUseCatalog},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}

func TestResourceCatalog_DoCreate_IsolationMode(t *testing.T) {
	tests := []struct {
		name          string
		isolationMode catalog.CatalogIsolationMode
		// The fake, like UC, reports OPEN for a catalog created without isolation_mode.
		want catalog.CatalogIsolationMode
	}{
		{name: "unset keeps the UC default", isolationMode: "", want: catalog.CatalogIsolationModeOpen},
		{name: "ISOLATED is applied after create", isolationMode: catalog.CatalogIsolationModeIsolated, want: catalog.CatalogIsolationModeIsolated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := setupTestServerClient(t)
			adapter := (*ResourceCatalog)(nil).New(client)
			ctx := t.Context()

			id, remote, err := adapter.DoCreate(ctx, &CatalogState{
				Name:          "test_catalog",
				IsolationMode: tt.isolationMode,
			})
			require.NoError(t, err)
			assert.Equal(t, "test_catalog", id)
			assert.Equal(t, tt.want, remote.IsolationMode)

			read, err := adapter.DoRead(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tt.want, read.IsolationMode)
		})
	}
}

func TestResourceCatalog_DoUpdate_IsolationMode(t *testing.T) {
	_, client := setupTestServerClient(t)
	adapter := (*ResourceCatalog)(nil).New(client)
	ctx := t.Context()

	config := &CatalogState{
		Name:          "test_catalog",
		IsolationMode: catalog.CatalogIsolationModeIsolated,
	}
	id, _, err := adapter.DoCreate(ctx, config)
	require.NoError(t, err)

	// An update that omits isolation_mode leaves the remote value alone (partial-update
	// semantics), which is what lets a config drop the field without forcing OPEN.
	config.IsolationMode = ""
	remote, err := adapter.DoUpdate(ctx, id, config, &PlanEntry{})
	require.NoError(t, err)
	assert.Equal(t, catalog.CatalogIsolationModeIsolated, remote.IsolationMode)

	config.IsolationMode = catalog.CatalogIsolationModeOpen
	remote, err = adapter.DoUpdate(ctx, id, config, &PlanEntry{})
	require.NoError(t, err)
	assert.Equal(t, catalog.CatalogIsolationModeOpen, remote.IsolationMode)
}
