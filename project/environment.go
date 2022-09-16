package project

import (
	"os"

	"github.com/spf13/cobra"
)

const bricksEnv = "BRICKS_ENV"

const defaultEnvironment = "development"

// Workspace defines configurables at the workspace level.
type Workspace struct {
	Profile string `json:"profile,omitempty"`
}

// Environment defines all configurables for a single environment.
type Environment struct {
	Workspace Workspace `json:"workspace"`
}

// getEnvironment returns the name of the environment to operate in.
func getEnvironment(cmd *cobra.Command) (value string) {
	// The command line flag takes precedence.
	flag := cmd.Flag("environment")
	if flag != nil {
		value = flag.Value.String()
		if value != "" {
			return
		}
	}

	// If it's not set, use the environment variable.
	value = os.Getenv(bricksEnv)
	if value != "" {
		return
	}

	return defaultEnvironment
}
