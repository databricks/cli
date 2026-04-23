package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/scripts"
)

// Engine-resolution source labels. Kept private and parallel to the labels
// used by cmd/ucm/utils.ResolveEngineSetting so diagnostics remain consistent
// with whichever entry point the caller exercised. Duplicated here (not
// imported) because cmd/ucm/utils → phases makes utils → phases a cycle.
const (
	engineSourceConfig  = "config"
	engineSourceEnv     = "env"
	engineSourceDefault = "default"
)

// resolveEngine mirrors cmd/ucm/utils.ResolveEngineSetting. It lives here to
// break the cmd/ucm/utils → phases import cycle that would otherwise form
// when the CLI layer wants to compose both the load/validate phases and the
// deploy phases from a single utils.ProcessUcm-style helper.
func resolveEngine(ctx context.Context, u *ucm.Ucm) (engine.EngineSetting, error) {
	configEngine := engine.EngineNotSet
	if u != nil {
		configEngine = u.Config.Ucm.Engine
	}

	if configEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:       configEngine,
			Source:     engineSourceConfig,
			ConfigType: configEngine,
		}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:   envEngine,
			Source: engineSourceEnv,
		}, nil
	}

	return engine.EngineSetting{
		Type:   engine.Default,
		Source: engineSourceDefault,
	}, nil
}

// Initialize fills in the workspace-path defaults, resource URLs, and
// deployment engine, then pulls remote state into the per-target local cache
// so downstream phases (build/plan/deploy/destroy) can read a consistent
// baseline. Errors are reported via logdiag; callers must check
// logdiag.HasError before proceeding.
//
// Mirrors the tail of bundle/phases/Initialize: DefineDefaultWorkspacePaths
// (depends on Workspace.RootPath set earlier by
// cmd/ucm/utils.ProcessUcm via DefineDefaultWorkspaceRoot + ExpandWorkspaceRoot)
// feeds the state path used by summary/debug, and InitializeURLs populates
// resource URLs so `ucm summary` / debug renderers can link them.
//
// Initialize does NOT retain the deploy lock across phases — state.Pull
// acquires and releases its own lock for the duration of the pull. The
// subsequent terraform Apply/Destroy acquires a fresh lock covering the write
// half of the deploy.
func Initialize(ctx context.Context, u *ucm.Ucm, opts Options) engine.EngineSetting {
	log.Info(ctx, "Phase: initialize")

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPreInit))
	if logdiag.HasError(ctx) {
		return engine.EngineSetting{}
	}

	ucm.ApplySeqContext(ctx, u,
		mutator.DefineDefaultWorkspacePaths(),
		mutator.InitializeURLs(),
	)
	if logdiag.HasError(ctx) {
		return engine.EngineSetting{}
	}

	setting, err := resolveEngine(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("resolve engine: %w", err))
		return engine.EngineSetting{}
	}
	log.Debugf(ctx, "initialize: engine=%s source=%s", setting.Type, setting.Source)

	if setting.Type.IsDirect() {
		// Direct engine state is a local-only artefact; there is no remote
		// tfstate to pull. Initialize's remaining work collapses to the
		// post_init script hook so downstream phases can still branch on
		// setting.Type.
		ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPostInit))
		return setting
	}

	pullBackend := opts.Backend
	pullBackend.ForceLock = opts.ForceLock
	if err := deploy.Pull(ctx, u, pullBackend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("pull remote state: %w", err))
		return setting
	}

	ucm.ApplyContext(ctx, u, scripts.Execute(config.ScriptPostInit))
	return setting
}
