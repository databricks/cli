package debug

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

type targetInfo struct {
	Name    string      `json:"name"`
	Default bool        `json:"default,omitempty"`
	Mode    config.Mode `json:"mode,omitempty"`
	Host    string      `json:"host,omitempty"`
}

type listTargetsOutput struct {
	Targets []targetInfo `json:"targets"`
}

func collectTargets(targets map[string]*config.Target) []targetInfo {
	names := slices.Sorted(maps.Keys(targets))

	result := make([]targetInfo, 0, len(names))
	for _, name := range names {
		t := targets[name]
		info := targetInfo{
			Name:    name,
			Default: t.Default,
			Mode:    t.Mode,
		}
		if t.Workspace != nil {
			info.Host = t.Workspace.Host
		}
		result = append(result, info)
	}
	return result
}

// NewListTargetsCommand returns a command that lists all bundle targets
// with their name, default, mode, and workspace host fields.
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

		targets := collectTargets(b.Config.Targets)

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
