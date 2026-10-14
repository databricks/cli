package generate

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/utils"
)

func ConvertAppToValue(app *apps.App, sourceCodePath string) (dyn.Value, error) {
	ar, err := convert.FromTyped(app.Resources, dyn.NilValue)
	if err != nil {
		return dyn.NilValue, err
	}

	// The majority of fields of the app struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"name":        dyn.NewValue(utils.Deref(app.Name), []diag.Location{{Line: 1}}),
		"description": dyn.NewValue(utils.Deref(app.Description), []diag.Location{{Line: 2}}),
	}

	// For a git-backed app, emit git_repository + git_source instead of a
	// workspace source_code_path. Otherwise the generated bundle would silently
	// down-convert the app to workspace source and point source_code_path at a
	// local directory that has nothing downloaded into it.
	if app.GitRepository != nil {
		dv["git_repository"] = gitRepositoryValue(app.GitRepository)
		if gs := gitSourceValue(app); gs.Kind() != dyn.KindNil {
			dv["git_source"] = gs
		}
	} else {
		dv["source_code_path"] = dyn.NewValue(sourceCodePath, []diag.Location{{Line: 4}})
	}

	if ar.Kind() != dyn.KindNil {
		dv["resources"] = ar.WithLocations([]diag.Location{{Line: 5}})
	}

	return dyn.V(dv), nil
}

func gitRepositoryValue(r *apps.GitRepository) dyn.Value {
	m := map[string]dyn.Value{
		"url":      dyn.NewValue(utils.Deref(r.Url), []diag.Location{{Line: 1}}),
		"provider": dyn.NewValue(utils.Deref(r.Provider), []diag.Location{{Line: 2}}),
	}
	if utils.Deref(r.AutoDeploy) {
		m["auto_deploy"] = dyn.NewValue(true, []diag.Location{{Line: 3}})
	}
	return dyn.NewValue(m, []diag.Location{{Line: 3}})
}

// gitSourceValue returns the reference the app deploys from (branch, tag, or
// commit, plus an optional repo-relative source_code_path). It prefers the
// configured git_source and falls back to the default source of the app's most
// recent deployment. System-populated fields (resolved_commit and the nested
// git_repository) are intentionally omitted.
func gitSourceValue(app *apps.App) dyn.Value {
	src := app.GitSource
	if src == nil {
		src = app.DefaultGitSource
	}
	if src == nil {
		return dyn.NilValue
	}

	m := map[string]dyn.Value{}
	switch {
	case utils.Deref(src.Branch) != "":
		m["branch"] = dyn.NewValue(*src.Branch, []diag.Location{{Line: 1}})
	case utils.Deref(src.Tag) != "":
		m["tag"] = dyn.NewValue(*src.Tag, []diag.Location{{Line: 1}})
	case utils.Deref(src.Commit) != "":
		m["commit"] = dyn.NewValue(*src.Commit, []diag.Location{{Line: 1}})
	}
	if sourceCodePath := utils.Deref(src.SourceCodePath); sourceCodePath != "" {
		m["source_code_path"] = dyn.NewValue(sourceCodePath, []diag.Location{{Line: 2}})
	}
	if len(m) == 0 {
		return dyn.NilValue
	}
	return dyn.NewValue(m, []diag.Location{{Line: 4}})
}
