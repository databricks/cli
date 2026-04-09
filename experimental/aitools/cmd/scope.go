package aitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

// promptScopeSelection is a package-level var so tests can replace it with a mock.
var promptScopeSelection = defaultPromptScopeSelection

// promptUpdateScopeSelection is a package-level var for the update scope prompt (3 options: global/project/both).
var promptUpdateScopeSelection = defaultPromptUpdateScopeSelection

// promptUninstallScopeSelection is a package-level var for the uninstall scope prompt (2 options: global/project).
var promptUninstallScopeSelection = defaultPromptUninstallScopeSelection

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

const scopeBoth = "both"

// detectInstalledScopes checks which scopes have a .state.json file present.
func detectInstalledScopes(ctx context.Context) (global, project bool, err error) {
	globalDir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return false, false, err
	}
	globalState, err := installer.LoadState(globalDir)
	if err != nil {
		return false, false, err
	}

	projectDir, err := installer.ProjectSkillsDir(ctx)
	if err != nil {
		return false, false, err
	}
	projectState, err := installer.LoadState(projectDir)
	if err != nil {
		return false, false, err
	}

	return globalState != nil, projectState != nil, nil
}

// resolveScopeForUpdate resolves scopes for the update command.
// Returns one or more scopes to update. When both flags are set, only installed scopes are returned.
func resolveScopeForUpdate(ctx context.Context, projectFlag, globalFlag bool) ([]string, error) {
	if projectFlag && globalFlag {
		hasGlobal, hasProject, err := detectInstalledScopes(ctx)
		if err != nil {
			return nil, err
		}
		var scopes []string
		if hasGlobal {
			scopes = append(scopes, installer.ScopeGlobal)
		}
		if hasProject {
			scopes = append(scopes, installer.ScopeProject)
		}
		if len(scopes) == 0 {
			return nil, errors.New("no skills installed. Run 'databricks experimental aitools install' to install")
		}
		return scopes, nil
	}
	if projectFlag {
		return withExplicitScopeCheck(ctx, installer.ScopeProject)
	}
	if globalFlag {
		return withExplicitScopeCheck(ctx, installer.ScopeGlobal)
	}

	// No flags: auto-detect.
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	if err != nil {
		return nil, err
	}

	switch {
	case hasGlobal && hasProject:
		if !cmdio.IsPromptSupported(ctx) {
			return nil, errors.New("skills are installed in both global and project scopes; use --global, --project, or both flags to specify which to update")
		}
		scopes, err := promptUpdateScopeSelection(ctx)
		if err != nil {
			return nil, err
		}
		return scopes, nil

	case hasGlobal:
		return []string{installer.ScopeGlobal}, nil

	case hasProject:
		return []string{installer.ScopeProject}, nil

	default:
		// Fall through to global scope so the installer layer can detect
		// legacy installs (skills on disk without .state.json) and provide
		// appropriate migration guidance.
		return []string{installer.ScopeGlobal}, nil
	}
}

// resolveScopeForUninstall resolves the scope for the uninstall command.
// Unlike update, uninstall never allows "both" scopes at once.
func resolveScopeForUninstall(ctx context.Context, projectFlag, globalFlag bool) (string, error) {
	if projectFlag && globalFlag {
		return "", errors.New("cannot uninstall both scopes at once; run uninstall separately for --global and --project")
	}
	if projectFlag {
		scopes, err := withExplicitScopeCheck(ctx, installer.ScopeProject)
		if err != nil {
			return "", err
		}
		return scopes[0], nil
	}
	if globalFlag {
		scopes, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal)
		if err != nil {
			return "", err
		}
		return scopes[0], nil
	}

	// No flags: auto-detect.
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	if err != nil {
		return "", err
	}

	switch {
	case hasGlobal && hasProject:
		if !cmdio.IsPromptSupported(ctx) {
			return "", errors.New("skills are installed in both global and project scopes; use --global or --project to specify which to uninstall")
		}
		scope, err := promptUninstallScopeSelection(ctx)
		if err != nil {
			return "", err
		}
		return scope, nil

	case hasGlobal:
		return installer.ScopeGlobal, nil

	case hasProject:
		return installer.ScopeProject, nil

	default:
		// Fall through to global scope so the installer layer can detect
		// legacy installs (skills on disk without .state.json) and provide
		// appropriate migration guidance.
		return installer.ScopeGlobal, nil
	}
}

// withExplicitScopeCheck validates that the explicitly requested scope has an installation.
// Returns a helpful error with CWD guidance for project scope and cross-scope hints.
func withExplicitScopeCheck(ctx context.Context, scope string) ([]string, error) {
	var dir string
	var err error

	if scope == installer.ScopeProject {
		dir, err = installer.ProjectSkillsDir(ctx)
	} else {
		dir, err = installer.GlobalSkillsDir(ctx)
	}
	if err != nil {
		return nil, err
	}

	state, err := installer.LoadState(dir)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, scopeNotInstalledError(ctx, scope)
	}

	return []string{scope}, nil
}

// scopeNotInstalledError builds a detailed error for when the requested scope has no installation.
// Includes cross-scope hints when the other scope is installed.
func scopeNotInstalledError(ctx context.Context, scope string) error {
	var msg string
	if scope == installer.ScopeProject {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("no project-scoped skills found: %w", err)
		}
		expectedPath := filepath.ToSlash(filepath.Join(cwd, ".databricks", "aitools", "skills"))
		msg = fmt.Sprintf(
			"no project-scoped skills found in the current directory.\n\n"+
				"Project-scoped skills are detected based on your working directory.\n"+
				"Make sure you are in the project root where you originally ran\n"+
				"'databricks experimental aitools install --project'.\n\n"+
				"Expected location: %s/", expectedPath)
	} else {
		msg = "no globally-scoped skills installed. Run 'databricks experimental aitools install --global' to install"
	}

	// Check if the other scope is installed and add a hint.
	hint := crossScopeHint(ctx, scope)
	if hint != "" {
		msg += "\n\n" + hint
	}

	return errors.New(msg)
}

// crossScopeHint returns a hint string if the opposite scope has an installation.
func crossScopeHint(ctx context.Context, requestedScope string) string {
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	if err != nil {
		return ""
	}
	if requestedScope == installer.ScopeProject && hasGlobal {
		return "Global skills are installed. Run without --project to update those."
	}
	if requestedScope == installer.ScopeGlobal && hasProject {
		return "Project-scoped skills are installed. Run without --global to update those."
	}
	return ""
}

func defaultPromptUpdateScopeSelection(ctx context.Context) ([]string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return nil, err
	}
	globalPath := filepath.Join(homeDir, ".databricks", "aitools", "skills")

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	projectPath := filepath.Join(cwd, ".databricks", "aitools", "skills")

	globalLabel := "Global (" + globalPath + "/)"
	projectLabel := "Project (" + projectPath + "/)"
	bothLabel := "Both global and project"

	var scope string
	err = huh.NewSelect[string]().
		Title("Which installation should be updated?").
		Options(
			huh.NewOption(globalLabel, installer.ScopeGlobal),
			huh.NewOption(projectLabel, installer.ScopeProject),
			huh.NewOption(bothLabel, scopeBoth),
		).
		Value(&scope).
		Run()
	if err != nil {
		return nil, err
	}

	if scope == scopeBoth {
		return []string{installer.ScopeGlobal, installer.ScopeProject}, nil
	}
	return []string{scope}, nil
}

func defaultPromptUninstallScopeSelection(ctx context.Context) (string, error) {
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

	globalLabel := "Global (" + globalPath + "/)"
	projectLabel := "Project (" + projectPath + "/)"

	var scope string
	err = huh.NewSelect[string]().
		Title("Which installation should be uninstalled?").
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
