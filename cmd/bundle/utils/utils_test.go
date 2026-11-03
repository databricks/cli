package utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetForceLock(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		registerFlag bool
		configured   bool
		forceLock    bool
		expected     bool
	}{
		{
			name:         "flag set to true overrides configured false",
			args:         []string{"--force-lock"},
			registerFlag: true,
			forceLock:    true,
			expected:     true,
		},
		{
			name:         "flag set to false overrides configured true",
			args:         []string{"--force-lock=false"},
			registerFlag: true,
			configured:   true,
			expected:     false,
		},
		{
			name:         "unset flag preserves configured true",
			registerFlag: true,
			configured:   true,
			expected:     true,
		},
		{
			name:         "unset flag leaves configured false alone",
			registerFlag: true,
			expected:     false,
		},
		{
			name:       "unregistered flag preserves configured true",
			configured: true,
			expected:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "deploy"}
			var forceLock bool
			if tc.registerFlag {
				cmd.Flags().BoolVar(&forceLock, "force-lock", false, "")
			}
			require.NoError(t, cmd.ParseFlags(tc.args))

			b := &bundle.Bundle{Config: config.Root{}}
			b.Config.Bundle.Deployment.Lock.Force = tc.configured

			SetForceLock(cmd, b, tc.forceLock)

			assert.Equal(t, tc.expected, b.Config.Bundle.Deployment.Lock.Force)
		})
	}
}
