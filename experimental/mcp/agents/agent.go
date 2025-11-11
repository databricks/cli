package agents

import (
	"os"
	"os/exec"
)

// Agent represents a coding agent that can have MCP servers installed.
type Agent struct {
	Name        string
	DisplayName string
	Detected    bool
	Installer   func() error
}

// IsOnPath checks if a command is available on the system PATH.
func IsOnPath(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetHomeDir returns the user's home directory.
func GetHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}
