package terraform

import (
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// fakeRunner is a stand-in for *tfexec.Terraform used by the init/plan/apply
// /destroy tests. It records call counts per method and lets tests inject
// return values for Plan.HasChanges and errors.
type fakeRunner struct {
	mu sync.Mutex

	InitCalls    int
	PlanCalls    int
	ApplyCalls   int
	DestroyCalls int
	SetEnvCalls  int

	// LastEnv captures the map passed to the most recent SetEnv call.
	LastEnv map[string]string
	// LastApplyOpts captures the options passed to the most recent Apply call.
	LastApplyOpts []tfexec.ApplyOption

	// PlanHasChanges is returned by Plan.
	PlanHasChanges bool
	// ApplyErr, InitErr, DestroyErr, PlanErr make the next corresponding
	// call return the given error.
	InitErr    error
	PlanErr    error
	ApplyErr   error
	DestroyErr error

	// ApplyHook is invoked synchronously inside Apply before returning. Used
	// by the lock contention test to hold the lock while a second goroutine
	// tries to acquire it.
	ApplyHook func(ctx context.Context)
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
