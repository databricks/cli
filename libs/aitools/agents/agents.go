package agents

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// Agent defines a supported coding agent.
type Agent struct {
	Name        string
	DisplayName string
	// ConfigDir returns the agent's config directory (e.g., ~/.claude).
	// Used for detection and as base for skills directory.
	ConfigDir func(ctx context.Context) (string, error)
	// SkillsSubdir is the subdirectory within ConfigDir for skills (default: "skills").
	SkillsSubdir string
	// SupportsProjectScope indicates whether this agent supports project-scoped skills.
	// When true, skills can be installed relative to the project root.
	SupportsProjectScope bool
	// ProjectConfigDir is the config directory name relative to a project root
	// (e.g., ".claude"). Only used when SupportsProjectScope is true.
	ProjectConfigDir string
}

// Detected returns true if the agent is installed on the system.
func (a *Agent) Detected(ctx context.Context) bool {
	dir, err := a.ConfigDir(ctx)
	if err != nil {
		return false
	}
	_, err = os.Stat(dir)
	return err == nil
}

// SkillsDir returns the full path to the agent's skills directory.
func (a *Agent) SkillsDir(ctx context.Context) (string, error) {
	configDir, err := a.ConfigDir(ctx)
	if err != nil {
		return "", err
	}
	subdir := a.SkillsSubdir
	if subdir == "" {
		subdir = "skills"
	}
	return filepath.Join(configDir, subdir), nil
}

// homeSubdir returns a function that computes ~/subpath.
func homeSubdir(subpath ...string) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		parts := append([]string{home}, subpath...)
		return filepath.Join(parts...), nil
	}
}

// ProjectSkillsDir returns the project-scoped skills directory for this agent.
// Only valid for agents where SupportsProjectScope is true.
func (a *Agent) ProjectSkillsDir(cwd string) string {
	subdir := a.SkillsSubdir
	if subdir == "" {
		subdir = "skills"
	}
	return filepath.Join(cwd, a.ProjectConfigDir, subdir)
}

// Registry contains all supported agents.
var Registry = []Agent{
	{
		Name:                 "claude-code",
		DisplayName:          "Claude Code",
		ConfigDir:            homeSubdir(".claude"),
		SupportsProjectScope: true,
		ProjectConfigDir:     ".claude",
	},
	{
		Name:                 "cursor",
		DisplayName:          "Cursor",
		ConfigDir:            homeSubdir(".cursor"),
		SupportsProjectScope: true,
		ProjectConfigDir:     ".cursor",
	},
	{
		Name:        "codex",
		DisplayName: "Codex CLI",
		ConfigDir:   homeSubdir(".codex"),
	},
	{
		Name:        "opencode",
		DisplayName: "OpenCode",
		ConfigDir:   homeSubdir(".config", "opencode"),
	},
	{
		Name:        "copilot",
		DisplayName: "GitHub Copilot",
		ConfigDir:   homeSubdir(".copilot"),
	},
	{
		Name:         "antigravity",
		DisplayName:  "Antigravity",
		ConfigDir:    homeSubdir(".gemini", "antigravity"),
		SkillsSubdir: "global_skills",
	},
}

// DetectInstalled returns all agents that are installed on the system.
func DetectInstalled(ctx context.Context) []*Agent {
	var installed []*Agent
	for i := range Registry {
		if Registry[i].Detected(ctx) {
			installed = append(installed, &Registry[i])
		}
	}
	return installed
}
