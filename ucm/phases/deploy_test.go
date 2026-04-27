package phases_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/deploy"
	ucmterraform "github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/cli/ucm/deployplan"
	"github.com/databricks/cli/ucm/metadata"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// destructiveCatalogPlan returns a PlanResult whose plan recreates a single
// catalog — enough to drive approvalForDeploy down its "destructive actions
// present" branch.
func destructiveCatalogPlan() *ucmterraform.PlanResult {
	return &ucmterraform.PlanResult{
		HasChanges: true,
		Plan: &deployplan.Plan{
			Plan: map[string]*deployplan.PlanEntry{
				"resources.catalogs.main": {Action: deployplan.Recreate},
			},
		},
	}
}

// nonDestructivePlan returns a PlanResult whose plan only creates resources —
// approvalForDeploy must treat this as "nothing to confirm" and proceed.
func nonDestructivePlan() *ucmterraform.PlanResult {
	return &ucmterraform.PlanResult{
		HasChanges: true,
		Plan: &deployplan.Plan{
			Plan: map[string]*deployplan.PlanEntry{
				"resources.catalogs.main": {Action: deployplan.Create},
			},
		},
	}
}

// readRemoteSeq returns the Seq of the remote ucm-state.json, or -1 when the
// file has not been written yet. Used to assert Push advanced the remote.
func readRemoteSeq(t *testing.T, f *fixture) int {
	t.Helper()
	rc, err := f.remote.Read(t.Context(), deploy.UcmStateFileName)
	if err != nil {
		return -1
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	var s deploy.State
	require.NoError(t, json.Unmarshal(data, &s))
	return s.Seq
}

// readRemoteMetadata returns the parsed ucm-metadata.json from the fixture's
// remote state dir, or nil when it has not been written.
func readRemoteMetadata(t *testing.T, f *fixture) *metadata.Metadata {
	t.Helper()
	rc, err := f.remote.Read(t.Context(), metadata.MetadataFileName)
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	var md metadata.Metadata
	require.NoError(t, json.Unmarshal(data, &md))
	return &md
}

func TestDeployHappyPath(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.RenderCalls)
	assert.Equal(t, 1, f.tf.InitCalls)
	assert.Equal(t, 1, f.tf.ApplyCalls)
	// Post-apply Push must advance the remote Seq from 0 to 1.
	assert.Equal(t, 1, readRemoteSeq(t, f))
	// Metadata upload follows Push: ucm-metadata.json must be present and
	// identify this deployment by name + target.
	md := readRemoteMetadata(t, f)
	require.NotNil(t, md)
	assert.Equal(t, metadata.Version, md.Version)
	assert.Equal(t, "test", md.Ucm.Name)
	assert.Equal(t, "dev", md.Ucm.Target)
	assert.False(t, md.Timestamp.IsZero())
}

// TestDeployDirectEngineSkipsTerraform asserts that when the direct engine
// is selected Deploy routes through direct.Apply rather than the terraform
// wrapper: the terraform fake's counters stay at zero, no remote state is
// pushed, and no error is raised by the happy empty-config path.
func TestDeployDirectEngineSkipsTerraform(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Ucm.Engine = engine.EngineDirect
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		TerraformFactory:    fakeTfFactory(f.tf),
		DirectClientFactory: fakeDirectClientFactory(),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
	assert.Equal(t, -1, readRemoteSeq(t, f), "direct engine must never touch remote state")
}

func TestDeployBailsOnApplyError(t *testing.T) {
	f := newFixture(t)
	f.tf.ApplyErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	// Apply failed → Push must NOT run; remote ucm-state.json must stay
	// absent because Pull only writes locally on first-run.
	assert.Equal(t, -1, readRemoteSeq(t, f))
}

func TestDeployBailsOnInitializeError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	// Zero Backend → Pull fails → the rest of Deploy short-circuits.
	phases.Deploy(ctx, f.u, phases.Options{TerraformFactory: fakeTfFactory(f.tf)})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
}

func TestDeployBailsOnBuildError(t *testing.T) {
	f := newFixture(t)
	f.tf.RenderErr = errSentinel
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
}

// TestDeploySkipsPromptWhenNoDestructiveActions asserts that a plan with only
// creates never blocks on approval: Apply must run and remote state must
// advance, regardless of whether a prompt would have been available.
func TestDeploySkipsPromptWhenNoDestructiveActions(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = nonDestructivePlan()
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.ApplyCalls)
}

// TestDeploySkipsPromptWhenAutoApprove asserts --auto-approve bypasses the
// prompt even when the plan contains destructive catalog actions.
func TestDeploySkipsPromptWhenAutoApprove(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = destructiveCatalogPlan()
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
		AutoApprove:      true,
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.ApplyCalls)
}

// TestDeployErrorsWhenPromptingNotSupported asserts Deploy returns an error
// asking for --auto-approve when stdin is not a TTY and the plan is
// destructive. Apply must not fire.
func TestDeployErrorsWhenPromptingNotSupported(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = destructiveCatalogPlan()
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "--auto-approve")
	assert.Equal(t, 0, f.tf.ApplyCalls)
}

// TestDeployPromptsAndAccepts drives the interactive prompt path: with a
// destructive plan and a TTY-shaped cmdio context fed "y\n" on stdin, Deploy
// must call Apply and push state.
func TestDeployPromptsAndAccepts(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = destructiveCatalogPlan()
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, tio := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	defer tio.Done()
	go func() {
		_, _ = tio.Stdin.WriteString("y\n")
		_ = tio.Stdin.Flush()
	}()
	// Drain prompt output so the cmdio writer never blocks on a full pipe.
	go func() { _, _ = io.Copy(io.Discard, tio.Stderr) }()

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 1, f.tf.ApplyCalls)
}

// TestDeployAbortsWhenUserLacksManage asserts that when the permission
// precheck finds the deploy principal lacks MANAGE on a declared catalog,
// Deploy emits a diagnostic and short-circuits before Build/Apply.
func TestDeployAbortsWhenUserLacksManage(t *testing.T) {
	f := newFixture(t)
	f.u.Config.Resources.Catalogs = map[string]*resources.Catalog{
		"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}},
	}
	f.u.CurrentUser = &config.User{User: &iam.User{UserName: "alice@example.com"}}
	f.mockWS.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.Anything).
		Return(&catalog.EffectivePermissionsList{
			PrivilegeAssignments: []catalog.EffectivePrivilegeAssignment{{
				Principal:  "alice@example.com",
				Privileges: []catalog.EffectivePrivilege{{Privilege: catalog.PrivilegeUseCatalog}},
			}},
		}, nil)

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "MANAGE")
	assert.Equal(t, 0, f.tf.ApplyCalls)
}

// TestDeployAbortsWhenPromptDeclined drives the same interactive path with
// "n\n" on stdin and asserts Apply is never called and no error surfaces —
// the cancel is a clean exit, not a failure.
func TestDeployAbortsWhenPromptDeclined(t *testing.T) {
	f := newFixture(t)
	f.tf.PlanResult = destructiveCatalogPlan()
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ctx, tio := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	defer tio.Done()
	var stderr strings.Builder
	go func() {
		_, _ = tio.Stdin.WriteString("n\n")
		_ = tio.Stdin.Flush()
	}()
	go func() { _, _ = io.Copy(&stderr, tio.Stderr) }()

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, 0, f.tf.ApplyCalls)
}
