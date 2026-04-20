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
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/terraform"
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

	RenderErr  error
	InitErr    error
	PlanErr    error
	ApplyErr   error
	DestroyErr error

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
