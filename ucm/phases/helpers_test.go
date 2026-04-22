package phases_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/require"
)

// fakeTf satisfies phases.TerraformWrapper for tests. Each method bumps a
// counter and returns the pre-seeded err/plan value so test cases can assert
// on call order and inject failures mid-sequence.
type fakeTf struct {
	mu sync.Mutex

	RenderCalls  int
	InitCalls    int
	PlanCalls    int
	ApplyCalls   int
	DestroyCalls int
	ImportCalls  int

	RenderErr  error
	InitErr    error
	PlanErr    error
	ApplyErr   error
	DestroyErr error
	ImportErr  error

	LastImportAddress string
	LastImportId      string

	PlanResult *terraform.PlanResult
}

func (f *fakeTf) Render(_ context.Context, _ *ucm.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RenderCalls++
	return f.RenderErr
}

func (f *fakeTf) Init(_ context.Context, _ *ucm.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.InitCalls++
	return f.InitErr
}

func (f *fakeTf) Plan(_ context.Context, _ *ucm.Ucm) (*terraform.PlanResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PlanCalls++
	return f.PlanResult, f.PlanErr
}

func (f *fakeTf) Apply(_ context.Context, _ *ucm.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ApplyCalls++
	return f.ApplyErr
}

func (f *fakeTf) Destroy(_ context.Context, _ *ucm.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DestroyCalls++
	return f.DestroyErr
}

func (f *fakeTf) Import(_ context.Context, _ *ucm.Ucm, address, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ImportCalls++
	f.LastImportAddress = address
	f.LastImportId = id
	return f.ImportErr
}

// fixture bundles the dependencies every phase test needs: a minimal Ucm with
// a target selected, a local-filer-backed Backend that satisfies deploy.Pull
// and deploy.Push, and the per-test fakeTf.
type fixture struct {
	t        *testing.T
	u        *ucm.Ucm
	backend  deploy.Backend
	tf       *fakeTf
	remote   libsfiler.Filer
	localDir string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	projDir := t.TempDir()
	remoteDir := t.TempDir()

	remote, err := libsfiler.NewLocalClient(remoteDir)
	require.NoError(t, err)

	u := &ucm.Ucm{RootPath: projDir}
	u.Config.Ucm = config.Ucm{Name: "test", Target: "dev"}

	return &fixture{
		t:  t,
		u:  u,
		tf: &fakeTf{},
		backend: deploy.Backend{
			StateFiler: ucmfiler.NewStateFilerFromFiler(remote),
			LockFiler:  remote,
			User:       "alice@example.com",
		},
		remote:   remote,
		localDir: filepath.Join(projDir, filepath.FromSlash(deploy.LocalCacheDir), "dev"),
	}
}

// errSentinel is a stable error identity for tests that assert the wrapped
// cause propagates through logdiag-formatted diagnostics.
var errSentinel = errors.New("sentinel")

// fakeDirectClient is the phases-level stand-in for direct.Client. It lets
// direct-engine tests exercise Plan/Deploy/Destroy without a real SDK. For
// the zero-resource fixture tests where Apply has nothing to do, the fake is
// never invoked — phases.Options.DirectClientFactory just needs to hand back
// a non-nil Client to satisfy the factory signature.
type fakeDirectClient struct{}

func (*fakeDirectClient) GetCatalog(_ context.Context, _ string) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateCatalog(_ context.Context, _ catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateCatalog(_ context.Context, _ catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteCatalog(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) GetSchema(_ context.Context, _ string) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateSchema(_ context.Context, _ catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateSchema(_ context.Context, _ catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteSchema(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) GetStorageCredential(_ context.Context, _ string) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateStorageCredential(_ context.Context, _ catalog.CreateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateStorageCredential(_ context.Context, _ catalog.UpdateStorageCredential) (*catalog.StorageCredentialInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteStorageCredential(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) GetExternalLocation(_ context.Context, _ string) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateExternalLocation(_ context.Context, _ catalog.CreateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateExternalLocation(_ context.Context, _ catalog.UpdateExternalLocation) (*catalog.ExternalLocationInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteExternalLocation(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) GetVolume(_ context.Context, _ string) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateVolume(_ context.Context, _ catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateVolume(_ context.Context, _ catalog.UpdateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteVolume(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) GetConnection(_ context.Context, _ string) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) CreateConnection(_ context.Context, _ catalog.CreateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) UpdateConnection(_ context.Context, _ catalog.UpdateConnection) (*catalog.ConnectionInfo, error) {
	return nil, nil
}

func (*fakeDirectClient) DeleteConnection(_ context.Context, _ string) error { return nil }

func (*fakeDirectClient) UpdatePermissions(_ context.Context, _ catalog.UpdatePermissions) error {
	return nil
}

func fakeDirectClientFactory() phases.DirectClientFactory {
	return func(_ context.Context, _ *ucm.Ucm) (direct.Client, error) {
		return &fakeDirectClient{}, nil
	}
}
