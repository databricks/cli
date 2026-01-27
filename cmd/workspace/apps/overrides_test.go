package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCommandHasDestroyAlias(t *testing.T) {
	cmd := New()
	require.NotNil(t, cmd)

	// Find the delete command
	deleteCmd, _, err := cmd.Find([]string{"delete"})
	require.NoError(t, err)
	require.NotNil(t, deleteCmd)

	// Verify that "destroy" is an alias for "delete"
	assert.Contains(t, deleteCmd.Aliases, "destroy", "delete command should have 'destroy' as an alias")

	// Verify that we can also find the command using the "destroy" alias
	destroyCmd, _, err := cmd.Find([]string{"destroy"})
	require.NoError(t, err)
	require.NotNil(t, destroyCmd)

	// Verify that both commands are the same
	assert.Equal(t, deleteCmd, destroyCmd, "delete and destroy should resolve to the same command")
}
