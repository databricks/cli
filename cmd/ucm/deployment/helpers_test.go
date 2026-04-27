package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/require"
)

// setupUcmFixture writes a ucm.yml with every supported resource kind into a
// fresh temp dir, loads it via ucm.Load, and selects the default target. The
// returned *ucm.Ucm is ready for resolveBindable (and phase-layer) tests.
func setupUcmFixture(t *testing.T) *ucm.Ucm {
	t.Helper()
	dir := t.TempDir()
	yml := `ucm:
  name: test-bind
  engine: direct

workspace:
  host: https://example.cloud.databricks.com

resources:
  catalogs:
    my_catalog:
      name: team_alpha
  schemas:
    my_schema:
      catalog_name: team_alpha
      name: bronze
  storage_credentials:
    my_sc:
      name: sc1
      aws_iam_role:
        role_arn: arn:aws:iam::1:role/x
  external_locations:
    my_loc:
      name: loc1
      url: s3://b/x
      credential_name: sc1
  volumes:
    my_vol:
      name: vol1
      catalog_name: team_alpha
      schema_name: bronze
      volume_type: MANAGED
  connections:
    my_conn:
      name: conn1
      connection_type: POSTGRESQL
      options: { host: db }
  grants:
    grant_a:
      securable: { type: catalog, name: team_alpha }
      principal: g
      privileges: [USE_CATALOG]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(yml), 0o644))

	u, err := ucm.Load(t.Context(), dir)
	require.NoError(t, err)
	// Direct-engine state paths depend on Config.Ucm.Target being set.
	u.Config.Ucm.Target = "default"
	return u
}
