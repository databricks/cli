package generate

import (
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func ConvertAppToValue(app *apps.App, sourceCodePath string) (structyaml.Map, error) {
	// The majority of fields of the app struct are read-only.
	// We copy the relevant fields manually.
	dv := structyaml.M(
		"name", app.Name,
		"description", app.Description,
	)

	// For a git-backed app, emit git_repository + git_source instead of a
	// workspace source_code_path. Otherwise the generated bundle would silently
	// down-convert the app to workspace source and point source_code_path at a
	// local directory that has nothing downloaded into it.
	if app.GitRepository != nil {
		dv.Add("git_repository", gitRepositoryValue(app.GitRepository))
		if gs := gitSourceValue(app); len(gs) > 0 {
			dv.Add("git_source", gs)
		}
	} else {
		dv.Add("source_code_path", sourceCodePath)
	}

	if ar := structyaml.Value(app.Resources); ar != nil {
		dv.Add("resources", ar)
	}

	return dv, nil
}

func gitRepositoryValue(r *apps.GitRepository) structyaml.Map {
	m := structyaml.M(
		"url", r.Url,
		"provider", r.Provider,
	)
	if r.AutoDeploy {
		m.Add("auto_deploy", r.AutoDeploy)
	}
	return m
}

// gitSourceValue returns the reference the app deploys from (branch, tag, or
// commit, plus an optional repo-relative source_code_path). It prefers the
// configured git_source and falls back to the default source of the app's most
// recent deployment. System-populated fields (resolved_commit and the nested
// git_repository) are intentionally omitted.
func gitSourceValue(app *apps.App) structyaml.Map {
	src := app.GitSource
	if src == nil {
		src = app.DefaultGitSource
	}
	if src == nil {
		return nil
	}

	var m structyaml.Map
	switch {
	case src.Branch != "":
		m.Add("branch", src.Branch)
	case src.Tag != "":
		m.Add("tag", src.Tag)
	case src.Commit != "":
		m.Add("commit", src.Commit)
	}
	if src.SourceCodePath != "" {
		m.Add("source_code_path", src.SourceCodePath)
	}
	return m
}
