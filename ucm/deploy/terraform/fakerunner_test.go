package terraform

import (
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// fakeRunner is a stand-in for *tfexec.Terraform used by the init/plan/apply
// /destroy tests. It records call counts per method and lets tests inject
// return values for Plan.HasChanges and errors.
type fakeRunner struct {
	mu sync.Mutex

	InitCalls         int
	PlanCalls         int
	ShowPlanFileCalls int
	ApplyCalls        int
	DestroyCalls      int
	ImportCalls       int
	StateRmCalls      int
	SetEnvCalls       int

	// LastEnv captures the map passed to the most recent SetEnv call.
	LastEnv map[string]string
	// LastApplyOpts captures the options passed to the most recent Apply call.
	LastApplyOpts []tfexec.ApplyOption
	// LastImportAddress and LastImportId capture the args passed to the most
	// recent Import call.
	LastImportAddress string
	LastImportId      string
	// LastStateRmAddress captures the address passed to the most recent
	// StateRm call.
	LastStateRmAddress string

	// PlanHasChanges is returned by Plan.
	PlanHasChanges bool
	// ShowPlanResult is returned by ShowPlanFile. nil yields an empty plan
	// (no resource changes) so callers don't need to set it for zero-diff
	// tests.
	ShowPlanResult *tfjson.Plan
	// ApplyErr, InitErr, DestroyErr, PlanErr, ShowPlanFileErr, ImportErr,
	// StateRmErr make the next corresponding call return the given error.
	InitErr         error
	PlanErr         error
	ShowPlanFileErr error
	ApplyErr        error
	DestroyErr      error
	ImportErr       error
	StateRmErr      error

	// ApplyHook is invoked synchronously inside Apply before returning. Used
	// by the lock contention test to hold the lock while a second goroutine
	// tries to acquire it.
	ApplyHook func(ctx context.Context)
	// ImportHook mirrors ApplyHook for the Import path.
	ImportHook func(ctx context.Context)
	// StateRmHook mirrors ApplyHook for the StateRm path.
	StateRmHook func(ctx context.Context)
}

func (f *fakeRunner) Init(_ context.Context, _ ...tfexec.InitOption) error {
	f.mu.Lock()
	f.InitCalls++
	err := f.InitErr
	f.mu.Unlock()
	return err
}

func (f *fakeRunner) Plan(_ context.Context, _ ...tfexec.PlanOption) (bool, error) {
	f.mu.Lock()
	f.PlanCalls++
	err := f.PlanErr
	changes := f.PlanHasChanges
	f.mu.Unlock()
	return changes, err
}

func (f *fakeRunner) ShowPlanFile(_ context.Context, _ string, _ ...tfexec.ShowOption) (*tfjson.Plan, error) {
	f.mu.Lock()
	f.ShowPlanFileCalls++
	err := f.ShowPlanFileErr
	plan := f.ShowPlanResult
	f.mu.Unlock()
	if plan == nil {
		plan = &tfjson.Plan{}
	}
	return plan, err
}

func (f *fakeRunner) Apply(ctx context.Context, opts ...tfexec.ApplyOption) error {
	f.mu.Lock()
	f.ApplyCalls++
	f.LastApplyOpts = opts
	hook := f.ApplyHook
	err := f.ApplyErr
	f.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return err
}

func (f *fakeRunner) Destroy(_ context.Context, _ ...tfexec.DestroyOption) error {
	f.mu.Lock()
	f.DestroyCalls++
	err := f.DestroyErr
	f.mu.Unlock()
	return err
}

func (f *fakeRunner) Import(ctx context.Context, address, id string, _ ...tfexec.ImportOption) error {
	f.mu.Lock()
	f.ImportCalls++
	f.LastImportAddress = address
	f.LastImportId = id
	hook := f.ImportHook
	err := f.ImportErr
	f.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return err
}

func (f *fakeRunner) StateRm(ctx context.Context, address string, _ ...tfexec.StateRmCmdOption) error {
	f.mu.Lock()
	f.StateRmCalls++
	f.LastStateRmAddress = address
	hook := f.StateRmHook
	err := f.StateRmErr
	f.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return err
}

func (f *fakeRunner) SetEnv(e map[string]string) error {
	f.mu.Lock()
	f.SetEnvCalls++
	f.LastEnv = e
	f.mu.Unlock()
	return nil
}

// newFakeRunnerFactory returns a tfRunnerFactory that always hands back the
// same fakeRunner — so tests can observe calls after the wrapper returns.
func newFakeRunnerFactory(r *fakeRunner) tfRunnerFactory {
	return func(_, _ string) (tfRunner, error) {
		if r == nil {
			return nil, errors.New("fakeRunner is nil")
		}
		return r, nil
	}
}
