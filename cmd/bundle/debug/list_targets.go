package debug

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

type targetInfo struct {
	Name        string      `json:"name"`
	Default     bool        `json:"default,omitempty"`
	Mode        config.Mode `json:"mode,omitempty"`
	Host        string      `json:"host,omitempty"`
	WorkspaceID string      `json:"workspace_id,omitempty"`

	// Engine is only reported for targets that use the removed Terraform engine.
	Engine engine.EngineType `json:"engine,omitempty"`
}

type listTargetsOutput struct {
	Targets []targetInfo `json:"targets"`
}

// collectTargets returns the targets sorted by name. The engine of a target is
// its own bundle.engine if set, otherwise the top-level bundle.engine.
func collectTargets(cfg *config.Root) []targetInfo {
	targets := cfg.Targets
	rootEngine := cfg.Bundle.Engine
	names := slices.Sorted(maps.Keys(targets))

	result := make([]targetInfo, 0, len(names))
	for _, name := range names {
		t := targets[name]
		host, workspaceID := targetHostAndWorkspaceID(&cfg.Workspace, t)
		info := targetInfo{
			Name:        name,
			Host:        host,
			WorkspaceID: workspaceID,
		}

		// YAML decoding can leave a nil entry in the map when a target is
		// declared with a null value. Skip rather than dereference and panic.
		if t == nil {
			info.Engine = terraformOnly(rootEngine)
			result = append(result, info)
			continue
		}
		info.Default = t.Default
		info.Mode = t.Mode
		targetEngine := rootEngine
		if t.Bundle != nil && t.Bundle.Engine != engine.EngineNotSet {
			targetEngine = t.Bundle.Engine
		}
		info.Engine = terraformOnly(targetEngine)
		result = append(result, info)
	}
	return result
}

// targetHostAndWorkspaceID returns the host and workspace ID of target t, which
// may be nil, normalized the same way authentication normalizes them. As in
// MergeTargetOverrides, the target's values take precedence over the top-level
// ones, except that an explicit empty value cannot be told apart from an unset
// one, so it does not clear the top-level value.
func targetHostAndWorkspaceID(rootWorkspace *config.Workspace, t *config.Target) (string, string) {
	w := config.Workspace{
		Host:        rootWorkspace.Host,
		WorkspaceID: rootWorkspace.WorkspaceID,
	}
	if t != nil && t.Workspace != nil {
		w.Host = cmp.Or(t.Workspace.Host, w.Host)
		w.WorkspaceID = cmp.Or(t.Workspace.WorkspaceID, w.WorkspaceID)
	}
	w.NormalizeHostURL()
	return w.Host, w.WorkspaceID
}

func terraformOnly(e engine.EngineType) engine.EngineType {
	if e == engine.EngineTerraform {
		return e
	}
	return engine.EngineNotSet
}

// NewListTargetsCommand returns a command that lists all bundle targets
// with their name, default, mode, workspace host and ID, and (if terraform) engine fields.
func NewListTargetsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "list-targets",
		Short:  "List all available bundle targets",
		Args:   root.NoArgs,
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)
		logdiag.SetSeverity(ctx, diag.Warning)

		b := bundle.MustLoad(ctx)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Load(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		targets := collectTargets(&b.Config)

		switch root.OutputType(cmd) {
		case flags.OutputText:
			for _, t := range targets {
				parts := []string{t.Name}
				if t.Default {
					parts = append(parts, "(default)")
				}
				if t.Mode != "" {
					parts = append(parts, string(t.Mode))
				}
				if t.Host != "" {
					parts = append(parts, t.Host)
				}
				if t.WorkspaceID != "" {
					parts = append(parts, "workspace_id:"+t.WorkspaceID)
				}
				if t.Engine != "" {
					parts = append(parts, "engine:"+string(t.Engine))
				}
				cmdio.LogString(ctx, strings.Join(parts, " "))
			}
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(listTargetsOutput{Targets: targets}, "", "  ")
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write(buf)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
