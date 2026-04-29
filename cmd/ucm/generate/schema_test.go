package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateSchemaCommand_Help(t *testing.T) {
	cmd := NewGenerateSchemaCommand()
	assert.Contains(t, cmd.Long, "Unity Catalog schema")
	assert.NotNil(t, cmd.Flag("existing-schema-name"))
}

func TestGenerateSchema_WritesYAML(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockSchemasAPI().EXPECT().
		GetByFullName(mock.Anything, "prod.events").
		Return(&catalog.SchemaInfo{
			Name:        "events",
			CatalogName: "prod",
			Comment:     "event data",
			Properties:  map[string]string{"team": "data"},
		}, nil)

	stderr, err := runSubcmd(t, w,
		"schema",
		"--existing-schema-name", "prod.events",
		"--output-dir", work,
	)
	require.NoError(t, err, "stderr=%s", stderr)

	// Default key derivation: dots become underscores.
	data, err := os.ReadFile(filepath.Join(work, "schemas_prod_events.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "schemas:")
	assert.Contains(t, contents, "name: events")
	assert.Contains(t, contents, "catalog_name: prod")
}

func TestGenerateSchema_KeyOverride(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockSchemasAPI().EXPECT().
		GetByFullName(mock.Anything, "prod.events").
		Return(&catalog.SchemaInfo{Name: "events", CatalogName: "prod"}, nil)

	_, err := runSubcmd(t, w,
		"--key", "events_schema",
		"schema",
		"--existing-schema-name", "prod.events",
		"--output-dir", work,
	)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(work, "schemas_events_schema.yml"))
	require.NoError(t, err)
}
