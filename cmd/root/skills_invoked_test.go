package root

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSkillsInvokedFlagAndEnvPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		flags []string
		want  []string
	}{
		{"neither", "", nil, nil},
		{"env only", "from-env", nil, []string{"from-env"}},
		{"flag only", "", []string{"--skills-invoked", "from-flag"}, []string{"from-flag"}},
		{"flag wins over env", "from-env", []string{"--skills-invoked", "from-flag"}, []string{"from-flag"}},
		{"invalid flag does not fall back to env", "from-env", []string{"--skills-invoked", "!!"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			initSkillsInvokedFlag(cmd)
			require.NoError(t, cmd.ParseFlags(tt.flags))

			ctx := t.Context()
			if tt.env != "" {
				ctx = env.Set(ctx, envSkillsInvoked, tt.env)
			}
			assert.Equal(t, tt.want, skillsinvoked.FromContext(withSkillsInvoked(ctx, cmd)))
		})
	}
}
