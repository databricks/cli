package ucm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/phases"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
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
	ImportCalls  int
	StateRmCalls int

	RenderErr  error
	InitErr    error
	PlanErr    error
	ApplyErr   error
	DestroyErr error
	ImportErr  error
	StateRmErr error

	LastImportAddress  string
	LastImportId       string
	LastStateRmAddress string

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

func (f *fakeTf) Apply(_ context.Context, _ *ucmpkg.Ucm, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ApplyCalls++
	return f.ApplyErr
}

func (f *fakeTf) Destroy(_ context.Context, _ *ucmpkg.Ucm, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DestroyCalls++
	return f.DestroyErr
}

func (f *fakeTf) Import(_ context.Context, _ *ucmpkg.Ucm, address, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ImportCalls++
	f.LastImportAddress = address
	f.LastImportId = id
	return f.ImportErr
}

func (f *fakeTf) StateRm(_ context.Context, _ *ucmpkg.Ucm, address string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StateRmCalls++
	f.LastStateRmAddress = address
	return f.StateRmErr
}

// verbHarness bundles the fake terraform wrapper, the remote-state filer
// backing Pull/Push, and an override of buildPhaseOptions so the verb under
// test runs against the fake instead of reaching for a real workspace client.
// directClient is optional — set it via WithDirectClient before runVerb to
// route the direct-engine verbs (drift, and the direct branches of
// plan/deploy/destroy) through an in-memory fake.
type verbHarness struct {
	tf           *fakeTf
	remote       libsfiler.Filer
	directClient direct.Client
}

// WithDirectClient configures the harness to hand the given client back from
// DirectClientFactory. Returns the harness for fluent setup.
func (h *verbHarness) WithDirectClient(c direct.Client) *verbHarness {
	h.directClient = c
	return h
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
		opts := phases.Options{
			Backend: deploy.Backend{
				StateFiler: ucmfiler.NewStateFilerFromFiler(remote),
				LockFiler:  remote,
				User:       "alice@example.com",
			},
			TerraformFactory: func(_ context.Context, _ *ucmpkg.Ucm) (phases.TerraformWrapper, error) {
				return h.tf, nil
			},
		}
		if h.directClient != nil {
			opts.DirectClientFactory = func(_ context.Context, _ *ucmpkg.Ucm) (direct.Client, error) {
				return h.directClient, nil
			}
		}
		return opts, nil
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
//
// The helper:
//   - Writes a fixture databrickscfg + restricts PATH so MustConfigureUcm
//     resolves a real (but offline) workspace client without invoking
//     az/gcloud or hitting the network.
//   - Installs a TestProcessHook that swaps in a mock WorkspaceClient on the
//     loaded Ucm and pre-seeds CurrentUser. Mock expectations cover the
//     deploy/destroy precheck calls (GetStatusByPath, Grants.GetEffective)
//     so the network-touching mutators short-circuit.
//   - Builds a minimal cobra root that owns the persistent --output flag the
//     verbs read via root.OutputType. Mirrors what cmd/root.New() supplies
//     in production without dragging in the full PersistentPreRunE chain.
//   - Strips PreRunE auth hooks on init/generate; their real hooks
//     (root.MustWorkspaceClient) need live credentials.
func runVerbInDir(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()

	setupTestEnvironment(t)
	t.Chdir(workDir)
	installTestProcessHook(t)

	rootCmd, ucmCmd, out, errOut := buildTestCobraRoot(t)
	rootCmd.SetArgs(append([]string{"ucm"}, args...))
	rootCmd.SetContext(buildTestContext(t, out, errOut))
	stripAuthHooks(ucmCmd)

	err := rootCmd.Execute()
	return out.String(), errOut.String(), err
}

// buildTestContext wires a cmdio context that fans out cmdio.LogString-style
// writes to the test's stderr buffer + a mock cmdctx workspace client so verbs
// (init/generate) that read off cmdctx don't panic when their PreRunE is
// stripped. Mirrors what root.New's PersistentPreRunE +
// root.MustWorkspaceClient do in production.
func buildTestContext(t *testing.T, out, errOut io.Writer) context.Context {
	t.Helper()
	ctx := t.Context()
	ctx = telemetry.WithNewLogger(ctx)
	cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, out, errOut, "", "")
	ctx = cmdio.InContext(ctx, cmdIO)

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{Host: "https://example.cloud.databricks.com"}
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)
	return ctx
}

// buildTestCobraRoot constructs a minimal cobra root with the persistent flags
// the ucm verbs read in production (currently --output) and adds the ucm
// subtree. Out/Err are captured into the returned buffers.
func buildTestCobraRoot(t *testing.T) (*cobra.Command, *cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	rootCmd := &cobra.Command{Use: "test-root", SilenceUsage: true, SilenceErrors: true}
	output := flags.OutputText
	rootCmd.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	// cmdUcm.New() registers itself under GroupID "development" so cobra
	// expects that group on the parent. Mirrors cmd/cmd.New's wiring.
	rootCmd.AddGroup(&cobra.Group{ID: "development", Title: "Development"})

	ucmCmd := New()
	rootCmd.AddCommand(ucmCmd)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	return rootCmd, ucmCmd, out, errOut
}

// setupTestEnvironment isolates the test from the developer's databricks
// config + restricts PATH so the SDK's auth resolution stays fully offline.
// We don't write a databrickscfg or set profile vars: every fixture under
// cmd/ucm/testdata declares its own workspace.host, and the SDK falls back
// to env-based auth (DATABRICKS_TOKEN / DATABRICKS_AUTH_TYPE) for the
// host-only profile resolution path. Mock-backed verbs override the
// resulting client via TestProcessHook.
func setupTestEnvironment(t *testing.T) {
	t.Helper()
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(tempHomeDir, "missing-databrickscfg"))
	t.Setenv(homeEnvVar, tempHomeDir)
	t.Setenv("DATABRICKS_TOKEN", "test-token")
	t.Setenv("DATABRICKS_AUTH_TYPE", "pat")
	// Clear any inherited profile / metadata env vars so the SDK can't pick
	// up the developer's local credentials and so DATABRICKS_HOST doesn't
	// override the per-fixture workspace.host.
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_METADATA_SERVICE_URL", "")
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", `C:\Windows\System32`)
	} else {
		t.Setenv("PATH", "/usr/bin:/bin")
	}
}

// installTestProcessHook sets utils.TestProcessHook to the standard verb-test
// seed: a mock WorkspaceClient + pre-set CurrentUser. The previous hook is
// restored on test cleanup so a parallel test file's seed doesn't leak across
// runs. Per-test customisations (extra mock expectations, extra warnings)
// chain on top by reading and re-wrapping utils.TestProcessHook.
func installTestProcessHook(t *testing.T) {
	t.Helper()
	prev := utils.TestProcessHook
	utils.TestProcessHook = func(ctx context.Context, u *ucmpkg.Ucm) {
		if prev != nil {
			prev(ctx, u)
		}
		seedFakeWorkspaceContext(t, u)
	}
	t.Cleanup(func() { utils.TestProcessHook = prev })
}

// seedFakeWorkspaceContext primes the loaded Ucm with the minimal state
// downstream mutators and verbs expect from a real deployment: a CurrentUser
// (so PopulateCurrentUser short-circuits) and a mock WorkspaceClient with the
// API expectations the deploy / destroy precheck paths exercise on the happy
// path. Per-test overrides can re-set expectations on the same mock by
// reading u.WorkspaceClientE().
func seedFakeWorkspaceContext(t *testing.T, u *ucmpkg.Ucm) {
	t.Helper()
	if u == nil {
		return
	}
	if u.CurrentUser == nil {
		u.CurrentUser = &config.User{
			ShortName: "test-user",
			User:      &iam.User{UserName: "test-user@example.com"},
		}
	}
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(mock.Anything, mock.Anything).
		Return(&workspace.ObjectInfo{}, nil).Maybe()
	m.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.Anything).
		Return(&catalog.EffectivePermissionsList{
			PrivilegeAssignments: []catalog.EffectivePrivilegeAssignment{{
				Principal:  "test-user@example.com",
				Privileges: []catalog.EffectivePrivilege{{Privilege: catalog.PrivilegeManage}},
			}},
		}, nil).Maybe()
	u.SetWorkspaceClient(m.WorkspaceClient)
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

// stripAuthHooks recursively clears PersistentPreRunE and PreRunE on cmd and
// all of its subcommands. The ucm verbs that need live auth (init, generate)
// wire root.MustWorkspaceClient as PreRunE; tests stand in their own Backend
// via buildPhaseOptions and a TestProcessHook so the real auth hook would just
// fail on a missing ~/.databrickscfg.
func stripAuthHooks(cmd *cobra.Command) {
	cmd.PersistentPreRunE = nil
	cmd.PersistentPreRun = nil
	cmd.PreRunE = nil
	cmd.PreRun = nil
	for _, sub := range cmd.Commands() {
		stripAuthHooks(sub)
	}
}
