package ucm

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/logdiag"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/require"
)

// fakeTf satisfies phases.TerraformWrapper for verb smoke tests. Mirrors the
// shape of ucm/phases/helpers_test.go's fakeTf but lives in this package so
// we don't have to re-export it.
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

func (f *fakeTf) Render(_ context.Context, _ *ucmpkg.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RenderCalls++
	return f.RenderErr
}

func (f *fakeTf) Init(_ context.Context, _ *ucmpkg.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.InitCalls++
	return f.InitErr
}

func (f *fakeTf) Plan(_ context.Context, _ *ucmpkg.Ucm) (*terraform.PlanResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PlanCalls++
	return f.PlanResult, f.PlanErr
}

func (f *fakeTf) Apply(_ context.Context, _ *ucmpkg.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ApplyCalls++
	return f.ApplyErr
}

func (f *fakeTf) Destroy(_ context.Context, _ *ucmpkg.Ucm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DestroyCalls++
	return f.DestroyErr
}

// verbHarness bundles the fake terraform wrapper, the remote-state filer
// backing Pull/Push, and an override of buildPhaseOptions so the verb under
// test runs against the fake instead of reaching for a real workspace client.
type verbHarness struct {
	tf     *fakeTf
	remote libsfiler.Filer
}

// newVerbHarness builds a harness keyed to a temp-dir "remote" filer and
// reassigns the package-level buildPhaseOptions to emit phases.Options that
// point at the fake. The original buildPhaseOptions is restored on test
// cleanup so later tests observe the production default.
func newVerbHarness(t *testing.T) *verbHarness {
	t.Helper()

	remoteDir := t.TempDir()
	remote, err := libsfiler.NewLocalClient(remoteDir)
	require.NoError(t, err)

	h := &verbHarness{
		tf:     &fakeTf{},
		remote: remote,
	}

	prev := buildPhaseOptions
	buildPhaseOptions = func(_ context.Context, _ *ucmpkg.Ucm) (phases.Options, error) {
		return phases.Options{
			Backend: deploy.Backend{
				StateFiler: ucmfiler.NewStateFilerFromFiler(remote),
				LockFiler:  remote,
				User:       "alice@example.com",
			},
			TerraformFactory: func(_ context.Context, _ *ucmpkg.Ucm) (phases.TerraformWrapper, error) {
				return h.tf, nil
			},
		}, nil
	}
	t.Cleanup(func() { buildPhaseOptions = prev })

	return h
}

// runVerb clones fixtureDir into a fresh temp dir and then invokes
// `databricks ucm <args...>` with cwd set to the clone. Cloning keeps
// state-pull side effects out of the repo checkout.
func runVerb(t *testing.T, fixtureDir string, args ...string) (string, string, error) {
	t.Helper()
	work := cloneFixture(t, fixtureDir)
	return runVerbInDir(t, work, args...)
}

// runVerbInDir runs the ucm cobra tree in workDir as-is (no cloning). Use
// this from tests that need to seed files into the cwd before invocation
// (e.g. summary tests seeding a tfstate).
func runVerbInDir(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workDir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := New()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	ctx, diagOut := cmdio.NewTestContextWithStderr(context.Background())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetRoot(ctx, workDir)
	cmd.SetContext(ctx)

	err = cmd.Execute()
	return out.String(), diagOut.String() + errOut.String(), err
}

// cloneFixture copies the flat set of files in fixtureDir into a per-test
// temp dir. Only top-level files are copied — the ucm fixtures don't nest
// and the state-pull side of verb tests would otherwise mutate the repo
// checkout.
func cloneFixture(t *testing.T, fixtureDir string) string {
	t.Helper()
	dst := t.TempDir()
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixtureDir, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644))
	}
	return dst
}

// validFixtureDir is the canonical on-disk ucm.yml the verb smoke tests drive
// against. Kept next to the cobra wiring so the test's intent (a happy-path
// run) stays obvious.
func validFixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "valid")
}

// assertSentinel is a stable error identity for tests that assert a specific
// phase error bubbles up to the cobra RunE. Name deliberately parallels the
// phases-package errSentinel without colliding.
var assertSentinel = errors.New("ucm verb test sentinel")
