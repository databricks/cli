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

func TestNewGenerateConnectionCommand_Help(t *testing.T) {
	cmd := NewGenerateConnectionCommand()
	assert.Contains(t, cmd.Long, "connection")
	assert.NotNil(t, cmd.Flag("existing-connection-name"))
}

func TestGenerateConnection_WritesYAML(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockConnectionsAPI().EXPECT().
		GetByName(mock.Anything, "prod_conn").
		Return(&catalog.ConnectionInfo{
			Name:           "prod_conn",
			ConnectionType: catalog.ConnectionTypePostgresql,
			Options: map[string]string{
				"host": "db.example.com",
				"port": "5432",
			},
			Comment: "prod pg",
		}, nil)

	_, err := runSubcmd(t, w,
		"connection",
		"--existing-connection-name", "prod_conn",
		"--output-dir", work,
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "connections_prod_conn.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "connections:")
	assert.Contains(t, contents, "connection_type: POSTGRESQL")
	// options entries (both keys and values) are double-quoted by tagsStyleKeys.
	assert.Contains(t, contents, `"host": "db.example.com"`)
	assert.Contains(t, contents, `"port": "5432"`)
}
