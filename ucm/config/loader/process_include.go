// Package loader holds mutators that load ucm.yml (or partial) configuration
// files from disk and merge them into the root Config tree. Forked from
// bundle/config/loader with DAB-specific pieces (resource-format heuristics)
// dropped.
package loader

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

type processInclude struct {
	fullPath string
	relPath  string
}

// ProcessInclude loads the configuration at [fullPath] and merges it into the
// root ucm configuration.
func ProcessInclude(fullPath, relPath string) ucm.Mutator {
	return &processInclude{fullPath: fullPath, relPath: relPath}
}

func (m *processInclude) Name() string {
	return fmt.Sprintf("ProcessInclude(%s)", m.relPath)
}

func (m *processInclude) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	this, diags := config.Load(m.fullPath)
	if diags.HasError() {
		return diags
	}

	// Included files are not allowed to declare their own include section —
	// matches DAB.
	if len(this.Include) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Include section is defined outside root file",
			Detail: `An include section is defined in a file that is not ucm.yml.
Only includes defined in ucm.yml are applied.`,
			Locations: this.GetLocations("include"),
			Paths:     []dyn.Path{dyn.MustPathFromString("include")},
		})
	}

	if err := u.Config.Merge(this); err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
