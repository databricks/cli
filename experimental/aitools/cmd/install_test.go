package mcp

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCommandsDelegateToSkillsInstall(t *testing.T) {
	originalInstallAllSkills := installAllSkills
	originalInstallSkill := installSkill
	t.Cleanup(func() {
		installAllSkills = originalInstallAllSkills
		installSkill = originalInstallSkill
	})

	tests := []struct {
		name           string
		newCmd         func() *cobra.Command
		args           []string
		wantAllCalls   int
		wantSkillCalls []string
	}{
		{
			name:         "skills install installs all skills",
			newCmd:       newSkillsInstallCmd,
			wantAllCalls: 1,
		},
		{
			name:           "skills install forwards skill name",
			newCmd:         newSkillsInstallCmd,
			args:           []string{"bundle/review"},
			wantSkillCalls: []string{"bundle/review"},
		},
		{
			name:         "top level install installs all skills",
			newCmd:       newInstallCmd,
			wantAllCalls: 1,
		},
		{
			name:           "top level install forwards skill name",
			newCmd:         newInstallCmd,
			args:           []string{"bundle/review"},
			wantSkillCalls: []string{"bundle/review"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allCalls := 0
			var skillCalls []string

			installAllSkills = func(context.Context) error {
				allCalls++
				return nil
			}
			installSkill = func(_ context.Context, skillName string) error {
				skillCalls = append(skillCalls, skillName)
				return nil
			}

			cmd := tt.newCmd()
			cmd.SetContext(t.Context())

			err := cmd.RunE(cmd, tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.wantAllCalls, allCalls)
			assert.Equal(t, tt.wantSkillCalls, skillCalls)
		})
	}
}
