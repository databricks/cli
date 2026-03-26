package aitools

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

// promptScopeSelection is a package-level var so tests can replace it with a mock.
var promptScopeSelection = defaultPromptScopeSelection

// resolveScope validates --project and --global flags and returns the scope.
func resolveScope(project, global bool) (string, error) {
	if project && global {
		return "", errors.New("cannot use --global and --project together")
	}
	if project {
		return installer.ScopeProject, nil
	}
	return installer.ScopeGlobal, nil
}

// resolveScopeWithPrompt resolves scope with optional interactive prompt.
// When neither flag is set: interactive mode shows a prompt (default: global),
// non-interactive mode uses global.
func resolveScopeWithPrompt(ctx context.Context, project, global bool) (string, error) {
	if project || global {
		return resolveScope(project, global)
	}

	// No flag: prompt if interactive, default to global otherwise.
	if cmdio.IsPromptSupported(ctx) {
		return promptScopeSelection(ctx)
	}
	return installer.ScopeGlobal, nil
}

func defaultPromptScopeSelection(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	globalPath := filepath.Join(homeDir, ".databricks", "aitools", "skills")

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	projectPath := filepath.Join(cwd, ".databricks", "aitools", "skills")

	globalLabel := "User global (" + globalPath + "/)\n    Available to you across all projects."
	projectLabel := "Project (" + projectPath + "/)\n    Checked into the repo, shared with everyone on the project."

	var scope string
	err = huh.NewSelect[string]().
		Title("Where should skills be installed?").
		Options(
			huh.NewOption(globalLabel, installer.ScopeGlobal),
			huh.NewOption(projectLabel, installer.ScopeProject),
		).
		Value(&scope).
		Run()
	if err != nil {
		return "", err
	}

	return scope, nil
}
