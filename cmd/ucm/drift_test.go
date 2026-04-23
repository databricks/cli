package ucm

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// driftFakeClient is the CLI-side stand-in for direct.Client. It answers
// catalog reads only — every other Get returns (nil, nil) which the drift
// comparator treats as "resource missing entirely", so fixtures that only
// record catalogs still produce meaningful assertions.
type driftFakeClient struct {
	catalogs map[string]*catalog.CatalogInfo
}

func (c *driftFakeClient) GetCatalog(_ context.Context, name string) (*catalog.CatalogInfo, error) {
	return c.catalogs[name], nil
}

func (c *driftFakeClient) GetSchema(context.Context, string) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (c *driftFakeClient) GetStorageCredential(context.Context, string) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (c *driftFakeClient) GetExternalLocation(context.Context, string) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (c *driftFakeClient) GetVolume(context.Context, string) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (c *driftFakeClient) GetConnection(context.Context, string) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

// Write-side stubs — unreachable from drift code but required for direct.Client.
func (*driftFakeClient) CreateCatalog(context.Context, catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateCatalog(context.Context, catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	panic("unexpected write")
}
func (*driftFakeClient) DeleteCatalog(context.Context, string) error { panic("unexpected write") }
func (*driftFakeClient) CreateSchema(context.Context, catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateSchema(context.Context, catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	panic("unexpected write")
}
func (*driftFakeClient) DeleteSchema(context.Context, string) error { panic("unexpected write") }
func (*driftFakeClient) CreateStorageCredential(context.Context, catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateStorageCredential(context.Context, catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) DeleteStorageCredential(context.Context, string) error {
	panic("unexpected write")
}

func (*driftFakeClient) CreateExternalLocation(context.Context, catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateExternalLocation(context.Context, catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) DeleteExternalLocation(context.Context, string) error {
	panic("unexpected write")
}

func (*driftFakeClient) CreateVolume(context.Context, catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateVolume(context.Context, catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	panic("unexpected write")
}
func (*driftFakeClient) DeleteVolume(context.Context, string) error { panic("unexpected write") }
func (*driftFakeClient) CreateConnection(context.Context, catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) UpdateConnection(context.Context, catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	panic("unexpected write")
}

func (*driftFakeClient) DeleteConnection(context.Context, string) error {
	panic("unexpected write")
}

func (*driftFakeClient) UpdatePermissions(context.Context, catalog.UpdatePermissions) error {
	panic("unexpected write")
}

// seedDirectState writes a direct.State file under workDir/.databricks/ucm/<target>/
// so the drift command's direct.LoadState picks it up. The valid fixture
// selects the default target; tests that need a non-default target should
// pass the name through.
func seedDirectState(t *testing.T, workDir, target string, state *direct.State) {
	t.Helper()
	path := filepath.Join(workDir, ".databricks", "ucm", target, direct.StateFileName)
	require.NoError(t, direct.SaveState(path, state))
}

func TestCmd_Drift_NoStatePrintsNoDrift(t *testing.T) {
	h := newVerbHarness(t).WithDirectClient(&driftFakeClient{})

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "drift")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "No drift detected")
	_ = h
}

func TestCmd_Drift_ReportsFieldMismatch(t *testing.T) {
	work := cloneFixture(t, validFixtureDir(t))
	seedDirectState(t, work, "default", &direct.State{
		Version:  direct.StateVersion,
		Catalogs: map[string]*direct.CatalogState{"sales": {Name: "sales", Comment: "sales data"}},
	})
	client := &driftFakeClient{
		catalogs: map[string]*catalog.CatalogInfo{
			"sales": {Name: "sales", Comment: "sales domain data"},
		},
	}
	newVerbHarness(t).WithDirectClient(client)

	stdout, _, err := runVerbInDir(t, work, "drift")

	// Drift detected → RunE returns ErrAlreadyPrinted which bubbles up as
	// a non-nil error on cobra.Execute. The text output is still produced.
	require.Error(t, err)
	assert.Contains(t, stdout, "DRIFT DETECTED on 1 resource(s)")
	assert.Contains(t, stdout, "resources.catalogs.sales")
	assert.Contains(t, stdout, `comment: state="sales data" live="sales domain data"`)
}

// TestRenderDriftText_NoDriftPrintsPositiveLine exercises the renderer
// directly — the integration tests cover the full verb path, and renderer
// tests assert the exact wire format without depending on the cobra harness
// wiring the -o flag (only root.New does that, not New() in isolation).
func TestRenderDriftText_NoDriftPrintsPositiveLine(t *testing.T) {
	var buf bytes.Buffer
	renderDriftText(&buf, &direct.Report{})
	assert.Equal(t, "No drift detected.\n", buf.String())
}

func TestRenderDriftText_DriftPrintsSpecFormat(t *testing.T) {
	var buf bytes.Buffer
	report := &direct.Report{Drift: []direct.ResourceDrift{
		{
			Key:    "resources.catalogs.sales",
			Fields: []direct.FieldDrift{{Field: "comment", State: "sales data", Live: "sales domain data"}},
		},
		{
			Key:    "resources.external_locations.shared",
			Fields: []direct.FieldDrift{{Field: "read_only", State: false, Live: true}},
		},
	}}
	renderDriftText(&buf, report)
	got := buf.String()
	assert.Contains(t, got, "DRIFT DETECTED on 2 resource(s):")
	assert.Contains(t, got, `  comment: state="sales data" live="sales domain data"`)
	assert.Contains(t, got, `  read_only: state=false live=true`)
}

func TestRenderDriftJSON_ProducesDriftKey(t *testing.T) {
	var buf bytes.Buffer
	report := &direct.Report{Drift: []direct.ResourceDrift{
		{
			Key:    "resources.catalogs.sales",
			Fields: []direct.FieldDrift{{Field: "comment", State: "a", Live: "b"}},
		},
	}}
	require.NoError(t, renderDriftJSON(&buf, report))

	var round struct {
		Drift []struct {
			Key    string `json:"key"`
			Fields []struct {
				Field string `json:"field"`
				State any    `json:"state"`
				Live  any    `json:"live"`
			} `json:"fields"`
		} `json:"drift"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &round))
	require.Len(t, round.Drift, 1)
	assert.Equal(t, "resources.catalogs.sales", round.Drift[0].Key)
	require.Len(t, round.Drift[0].Fields, 1)
	assert.Equal(t, "comment", round.Drift[0].Fields[0].Field)
	assert.Equal(t, "a", round.Drift[0].Fields[0].State)
	assert.Equal(t, "b", round.Drift[0].Fields[0].Live)
}
