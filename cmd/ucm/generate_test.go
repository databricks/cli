package ucm

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateFake is the direct.Client fake wired into cmd-level generate tests.
// Only the List methods are populated; the rest are satisfied via shared stubs.
type generateFake struct {
	host               string
	catalogs           []catalog.CatalogInfo
	schemas            map[string][]catalog.SchemaInfo
	volumes            map[string][]catalog.VolumeInfo
	storageCredentials []catalog.StorageCredentialInfo
	externalLocations  []catalog.ExternalLocationInfo
	connections        []catalog.ConnectionInfo
}

func (f *generateFake) ListCatalogs(_ context.Context) ([]catalog.CatalogInfo, error) {
	return f.catalogs, nil
}

func (f *generateFake) ListSchemas(_ context.Context, name string) ([]catalog.SchemaInfo, error) {
	return f.schemas[name], nil
}

func (f *generateFake) ListVolumes(_ context.Context, cat, sch string) ([]catalog.VolumeInfo, error) {
	return f.volumes[cat+"."+sch], nil
}

func (f *generateFake) ListStorageCredentials(_ context.Context) ([]catalog.StorageCredentialInfo, error) {
	return f.storageCredentials, nil
}

func (f *generateFake) ListExternalLocations(_ context.Context) ([]catalog.ExternalLocationInfo, error) {
	return f.externalLocations, nil
}

func (f *generateFake) ListConnections(_ context.Context) ([]catalog.ConnectionInfo, error) {
	return f.connections, nil
}

func (*generateFake) GetCatalog(context.Context, string) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*generateFake) CreateCatalog(context.Context, catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateCatalog(context.Context, catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteCatalog(context.Context, string) error                    { return nil }
func (*generateFake) GetSchema(context.Context, string) (*catalog.SchemaInfo, error) { return nil, nil }
func (*generateFake) CreateSchema(context.Context, catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateSchema(context.Context, catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteSchema(context.Context, string) error { return nil }
func (*generateFake) GetStorageCredential(context.Context, string) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*generateFake) CreateStorageCredential(context.Context, catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateStorageCredential(context.Context, catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteStorageCredential(context.Context, string) error { return nil }
func (*generateFake) GetExternalLocation(context.Context, string) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*generateFake) CreateExternalLocation(context.Context, catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateExternalLocation(context.Context, catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteExternalLocation(context.Context, string) error { return nil }
func (*generateFake) GetVolume(context.Context, string) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*generateFake) CreateVolume(context.Context, catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateVolume(context.Context, catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteVolume(context.Context, string) error { return nil }
func (*generateFake) GetConnection(context.Context, string) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*generateFake) CreateConnection(context.Context, catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*generateFake) UpdateConnection(context.Context, catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}
func (*generateFake) DeleteConnection(context.Context, string) error                     { return nil }
func (*generateFake) UpdatePermissions(context.Context, catalog.UpdatePermissions) error { return nil }

// installGenerateFake swaps generateClientFactory for the duration of t.
func installGenerateFake(t *testing.T, f *generateFake) {
	t.Helper()
	host := f.host
	prev := generateClientFactory
	generateClientFactory = func(*cobra.Command) (string, direct.Client, error) {
		return host, f, nil
	}
	t.Cleanup(func() { generateClientFactory = prev })
}

// runGenerate invokes the ucm root with `generate args...` against workDir.
// The auth PreRunE hook is stripped because generate's real hook
// (root.MustWorkspaceClient) would try to resolve live credentials.
func runGenerate(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workDir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := New()
	stripAuthHooks(cmd)

	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"generate"}, args...))

	ctx, diagOut := cmdio.NewTestContextWithStderr(context.Background())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetRoot(ctx, workDir)
	cmd.SetContext(ctx)

	err = cmd.Execute()
	return out.String(), diagOut.String() + errOut.String(), err
}

func TestCmd_Generate_WritesUcmYmlAndSeedState(t *testing.T) {
	fake := &generateFake{
		host: "https://acme-prod.cloud.databricks.com",
		catalogs: []catalog.CatalogInfo{
			{Name: "team_alpha", Comment: "alpha", Properties: map[string]string{"owner": "alpha"}},
		},
		schemas: map[string][]catalog.SchemaInfo{
			"team_alpha": {{Name: "bronze", CatalogName: "team_alpha"}},
		},
		volumes: map[string][]catalog.VolumeInfo{
			"team_alpha.bronze": {{Name: "landing", CatalogName: "team_alpha", SchemaName: "bronze", VolumeType: catalog.VolumeTypeManaged}},
		},
	}
	installGenerateFake(t, fake)

	work := t.TempDir()
	stdout, stderr, err := runGenerate(t, work, "--name", "scanned-prod")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)
	require.NoError(t, err)

	ymlPath := filepath.Join(work, "ucm.yml")
	data, err := os.ReadFile(ymlPath)
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "name: scanned-prod")
	assert.Contains(t, contents, "host: https://acme-prod.cloud.databricks.com")
	assert.Contains(t, contents, "team_alpha")
	assert.Contains(t, contents, "bronze")
	assert.Contains(t, contents, "landing")

	statePath := filepath.Join(work, ".databricks", "ucm", "default", "resources.json")
	stateData, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var s direct.State
	require.NoError(t, json.Unmarshal(stateData, &s))
	assert.Equal(t, direct.StateVersion, s.Version)
	assert.Len(t, s.Catalogs, 1)
	assert.Len(t, s.Schemas, 1)
	assert.Len(t, s.Volumes, 1)

	assert.Contains(t, stderr, "Scanned:")
	assert.Contains(t, stderr, "catalogs=1")
}

func TestCmd_Generate_RefusesToOverwriteWithoutForce(t *testing.T) {
	installGenerateFake(t, &generateFake{host: "https://x.example.com"})

	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "ucm.yml"), []byte("existing"), 0o644))

	_, _, err := runGenerate(t, work, "--name", "n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCmd_Generate_ForceOverwrites(t *testing.T) {
	installGenerateFake(t, &generateFake{host: "https://x.example.com"})

	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "ucm.yml"), []byte("existing"), 0o644))

	_, _, err := runGenerate(t, work, "--name", "fresh", "--force")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "ucm.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "fresh")
	assert.NotContains(t, string(data), "existing")
}

func TestCmd_Generate_KindsFilter(t *testing.T) {
	fake := &generateFake{
		host: "https://x.example.com",
		catalogs: []catalog.CatalogInfo{
			{Name: "team_alpha"},
		},
		schemas: map[string][]catalog.SchemaInfo{
			"team_alpha": {{Name: "bronze", CatalogName: "team_alpha"}},
		},
		// If schemas were in the scan set these would appear; with
		// --kinds catalog only catalogs land in the emitted yaml.
		connections: []catalog.ConnectionInfo{
			{Name: "should-not-appear", ConnectionType: catalog.ConnectionTypePostgresql, Options: map[string]string{"host": "h"}},
		},
	}
	installGenerateFake(t, fake)

	work := t.TempDir()
	_, _, err := runGenerate(t, work, "--name", "n", "--kinds", "catalog")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "ucm.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "team_alpha")
	assert.NotContains(t, contents, "should-not-appear")
	assert.NotContains(t, contents, "bronze")
}

func TestCmd_Generate_UnknownKindErrors(t *testing.T) {
	installGenerateFake(t, &generateFake{host: "https://x.example.com"})

	work := t.TempDir()
	_, stderr, err := runGenerate(t, work, "--name", "n", "--kinds", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error()+stderr, "unknown kind")
}

func TestCmd_Generate_DefaultNameDerivesFromHost(t *testing.T) {
	installGenerateFake(t, &generateFake{host: "https://acme-prod.cloud.databricks.com"})

	work := t.TempDir()
	_, _, err := runGenerate(t, work)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "ucm.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: acme-prod")
}

func TestCmd_Generate_UsesPreRunE(t *testing.T) {
	root := New()
	for _, sub := range root.Commands() {
		if sub.Name() != "generate" {
			continue
		}
		assert.Nil(t, sub.PersistentPreRunE, "generate must not set PersistentPreRunE")
		assert.NotNil(t, sub.PreRunE, "generate must set PreRunE for workspace-client auth")
		return
	}
	t.Fatal("generate verb not registered")
}
