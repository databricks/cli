package agents

import (
	"os"
	"path/filepath"
	"runtime"
)

// Agent defines a coding agent that can have skills installed and optionally MCP server.
type Agent struct {
	Name        string
	DisplayName string
	// ConfigDir returns the agent's config directory (e.g., ~/.claude).
	// Used for detection and as base for skills directory.
	ConfigDir func() (string, error)
	// SkillsSubdir is the subdirectory within ConfigDir for skills (default: "skills").
	SkillsSubdir string
	// InstallMCP installs the Databricks MCP server for this agent.
	// Nil if agent doesn't support MCP or we haven't implemented it.
	InstallMCP func() error
}

// Detected returns true if the agent is installed on the system.
func (a *Agent) Detected() bool {
	dir, err := a.ConfigDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(dir)
	return err == nil
}

// SkillsDir returns the full path to the agent's skills directory.
func (a *Agent) SkillsDir() (string, error) {
	configDir, err := a.ConfigDir()
	if err != nil {
		return "", err
	}
	subdir := a.SkillsSubdir
	if subdir == "" {
		subdir = "skills"
	}
	return filepath.Join(configDir, subdir), nil
}

// getHomeDir returns home directory, handling Windows USERPROFILE.
func getHomeDir() (string, error) {
	if runtime.GOOS == "windows" {
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			return userProfile, nil
		}
	}
	return os.UserHomeDir()
}

// homeSubdir returns a function that computes ~/subpath.
func homeSubdir(subpath ...string) func() (string, error) {
	return func() (string, error) {
		home, err := getHomeDir()
		if err != nil {
			return "", err
		}
		parts := append([]string{home}, subpath...)
		return filepath.Join(parts...), nil
	}
}

// Registry contains all supported agents.
var Registry = []Agent{
	{
		Name:        "claude-code",
		DisplayName: "Claude Code",
		ConfigDir:   homeSubdir(".claude"),
		InstallMCP:  InstallClaude,
	},
	{
		Name:        "cursor",
		DisplayName: "Cursor",
		ConfigDir:   homeSubdir(".cursor"),
		InstallMCP:  InstallCursor,
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
		Name:        "antigravity",
		DisplayName: "Antigravity",
		ConfigDir:   homeSubdir(".gemini", "antigravity"),
	},
}

// DetectInstalled returns all agents that are installed on the system.
func DetectInstalled() []*Agent {
	var installed []*Agent
	for i := range Registry {
		if Registry[i].Detected() {
			installed = append(installed, &Registry[i])
		}
	}
	return installed
}

// GetByName returns an agent by name, or nil if not found.
func GetByName(name string) *Agent {
	for i := range Registry {
		if Registry[i].Name == name {
			return &Registry[i]
		}
	}
	return nil
}
