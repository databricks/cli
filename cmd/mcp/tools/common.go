package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// GetCLIPath returns the path to the current CLI executable.
// This supports development testing with ./cli.
func GetCLIPath() string {
	return os.Args[0]
}

// ValidateProjectPath checks if a project path exists and is a directory.
func ValidateProjectPath(projectPath string) error {
	if projectPath == "" {
		return errors.New("project_path is required")
	}

	pathInfo, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory does not exist: %s", projectPath)
		}
		return fmt.Errorf("failed to access project path: %w", err)
	}

	if !pathInfo.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	return nil
}

// ValidateDatabricksProject checks if a directory is a valid Databricks project.
// It ensures the directory exists and contains a databricks.yml file.
func ValidateDatabricksProject(projectPath string) error {
	if err := ValidateProjectPath(projectPath); err != nil {
		return err
	}

	databricksYml := filepath.Join(projectPath, "databricks.yml")
	if _, err := os.Stat(databricksYml); os.IsNotExist(err) {
		return fmt.Errorf("not a Databricks project: databricks.yml not found in %s\n\nUse the init_project tool to create a new project first", projectPath)
	}

	return nil
}
