package project

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentFromCommand(t *testing.T) {
	var cmd cobra.Command
	cmd.Flags().String("environment", "", "specify environment")
	cmd.Flags().Set("environment", "env-from-arg")
	t.Setenv(bricksEnv, "")

	value := getEnvironment(&cmd)
	assert.Equal(t, "env-from-arg", value)
}

func TestEnvironmentFromEnvironment(t *testing.T) {
	var cmd cobra.Command
	cmd.Flags().String("environment", "", "specify environment")
	cmd.Flags().Set("environment", "")
	t.Setenv(bricksEnv, "env-from-env")

	value := getEnvironment(&cmd)
	assert.Equal(t, "env-from-env", value)
}

func TestEnvironmentDefault(t *testing.T) {
	var cmd cobra.Command
	cmd.Flags().String("environment", "", "specify environment")
	cmd.Flags().Set("environment", "")
	t.Setenv(bricksEnv, "")

	value := getEnvironment(&cmd)
	assert.Equal(t, DefaultEnvironment, value)
}
