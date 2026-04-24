package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
)

// BindRequest bundles the operator-supplied inputs for a single bind. Name is
// the UC identifier of the existing object (e.g. "team_alpha" for a catalog,
// "team_alpha.bronze" for a schema); Key is the ucm.yml map key the binding
// will be recorded under. Kind reuses the ImportKind vocabulary since bind
// and import share the same per-resource primitives.
type BindRequest struct {
	Kind ImportKind
	Name string
	Key  string
}

// UnbindRequest mirrors BindRequest but only needs the Kind+Key pair — unbind
// drops the recorded state entry without touching the remote UC object, so
// the UC name is immaterial.
type UnbindRequest struct {
	Kind ImportKind
	Key  string
}

// Bind resolves the deployment engine and attaches an existing Unity Catalog
// object to the ucm-declared key in req.Key. The direct engine records a
// state entry; the terraform engine runs `terraform import`. Errors are
// reported via logdiag; callers must check logdiag.HasError before
// continuing. The terraform path pushes state on success; the direct path
// rewrites resources.json in place. Mirrors bundle/phases/bind.go in shape.
func Bind(ctx context.Context, u *ucm.Ucm, opts Options, req BindRequest) {
	log.Infof(ctx, "Phase: bind %s %s", req.Kind, req.Name)

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	if err := validateResourceDeclared(u, ImportRequest(req)); err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if setting.Type.IsDirect() {
		bindDirect(ctx, u, opts, req)
		return
	}
	bindTerraform(ctx, u, opts, req)
}

// Unbind resolves the deployment engine and drops the recorded binding for
// req.Key. The direct engine deletes the state entry in resources.json; the
// terraform engine runs `terraform state rm`. The remote UC object is never
// touched — unbind is a state-only operation. Mirrors
// bundle/phases/bind.go::Unbind in shape.
func Unbind(ctx context.Context, u *ucm.Ucm, opts Options, req UnbindRequest) {
	log.Infof(ctx, "Phase: unbind %s %s", req.Kind, req.Key)

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		unbindDirect(ctx, u, opts, req)
		return
	}
	unbindTerraform(ctx, u, opts, req)
}

func bindTerraform(ctx context.Context, u *ucm.Ucm, opts Options, req BindRequest) {
	factory := opts.terraformFactoryOrDefault()
	tf, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("build terraform wrapper: %w", err))
		return
	}

	if err := tf.Render(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("render terraform config: %w", err))
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	if err := tf.Import(ctx, u, terraformAddress(ImportRequest(req)), req.Name); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform import: %w", err))
		return
	}

	if err := deploy.Push(ctx, u, opts.Backend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}

func bindDirect(ctx context.Context, u *ucm.Ucm, opts Options, req BindRequest) {
	ucm.ApplyContext(ctx, u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	if logdiag.HasError(ctx) {
		return
	}

	factory := opts.directClientFactoryOrDefault()
	client, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("resolve direct client: %w", err))
		return
	}

	statePath := direct.StatePath(u)
	state, err := direct.LoadState(statePath)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return
	}

	if err := direct.ImportResource(ctx, u, client, state, string(req.Kind), req.Name, req.Key); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct bind: %w", err))
		return
	}

	if err := direct.SaveState(statePath, state); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", err))
	}
}

func unbindTerraform(ctx context.Context, u *ucm.Ucm, opts Options, req UnbindRequest) {
	factory := opts.terraformFactoryOrDefault()
	tf, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("build terraform wrapper: %w", err))
		return
	}

	if err := tf.Render(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("render terraform config: %w", err))
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	address := terraformAddress(ImportRequest{Kind: req.Kind, Key: req.Key})
	if err := tf.StateRm(ctx, u, address); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform state rm: %w", err))
		return
	}

	if err := deploy.Push(ctx, u, opts.Backend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}

func unbindDirect(ctx context.Context, u *ucm.Ucm, opts Options, req UnbindRequest) {
	_ = opts // direct-engine unbind does not need Options.Backend
	statePath := direct.StatePath(u)
	state, err := direct.LoadState(statePath)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return
	}

	if err := direct.UnbindResource(ctx, state, string(req.Kind), req.Key); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct unbind: %w", err))
		return
	}

	if err := direct.SaveState(statePath, state); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", err))
	}
}
