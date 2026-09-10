package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func ConvertAppToValue(app *apps.App, sourceCodePath string) (dyn.Value, error) {
	ar, err := convert.FromTyped(app.Resources, dyn.NilValue)
	if err != nil {
		return dyn.NilValue, err
	}

	// The majority of fields of the app struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"name":        dyn.NewValue(app.Name, []dyn.Location{{Line: 1}}),
		"description": dyn.NewValue(app.Description, []dyn.Location{{Line: 2}}),
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
		dv["source_code_path"] = dyn.NewValue(sourceCodePath, []dyn.Location{{Line: 4}})
	}

	if ar.Kind() != dyn.KindNil {
		dv["resources"] = ar.WithLocations([]dyn.Location{{Line: 5}})
	}

	return dyn.V(dv), nil
}

func gitRepositoryValue(r *apps.GitRepository) dyn.Value {
	m := map[string]dyn.Value{
		"url":      dyn.NewValue(r.Url, []dyn.Location{{Line: 1}}),
		"provider": dyn.NewValue(r.Provider, []dyn.Location{{Line: 2}}),
	}
	if r.AutoDeploy {
		m["auto_deploy"] = dyn.NewValue(r.AutoDeploy, []dyn.Location{{Line: 3}})
	}
	return dyn.NewValue(m, []dyn.Location{{Line: 3}})
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
	case src.Branch != "":
		m["branch"] = dyn.NewValue(src.Branch, []dyn.Location{{Line: 1}})
	case src.Tag != "":
		m["tag"] = dyn.NewValue(src.Tag, []dyn.Location{{Line: 1}})
	case src.Commit != "":
		m["commit"] = dyn.NewValue(src.Commit, []dyn.Location{{Line: 1}})
	}
	if src.SourceCodePath != "" {
		m["source_code_path"] = dyn.NewValue(src.SourceCodePath, []dyn.Location{{Line: 2}})
	}
	if len(m) == 0 {
		return dyn.NilValue
	}
	return dyn.NewValue(m, []dyn.Location{{Line: 4}})
}
