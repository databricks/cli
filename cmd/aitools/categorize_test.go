package aitools

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestClassifyInstallError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want protos.AitoolsErrorCategory
	}{
		{
			name: "nil is success",
			err:  nil,
			want: protos.AitoolsErrorCategoryUnspecified,
		},
		{
			name: "blocked cli not on path",
			err:  &installer.BlockedError{Agent: "claude-code", Reason: installer.ReasonCLINotOnPath},
			want: protos.AitoolsErrorCategoryCLINotOnPath,
		},
		{
			name: "blocked install failed",
			err:  &installer.BlockedError{Agent: "codex", Reason: installer.ReasonInstallFailed},
			want: protos.AitoolsErrorCategoryPluginInstallFailed,
		},
		{
			name: "blocked no plugin is uncategorized",
			err:  &installer.BlockedError{Agent: "codex", Reason: installer.ReasonNoPlugin},
			want: protos.AitoolsErrorCategoryUncategorized,
		},
		{
			name: "wrapped skill not found",
			err:  fmt.Errorf("resolve failed: %w", &installer.SkillError{Skill: "databricks", Reason: installer.ReasonSkillNotFound, Detail: "not found"}),
			want: protos.AitoolsErrorCategorySkillNotFound,
		},
		{
			name: "version incompatible",
			err:  &installer.SkillError{Skill: "databricks", Reason: installer.ReasonVersionIncompatible, Detail: "requires CLI version 0.5 (running 0.4)"},
			want: protos.AitoolsErrorCategoryVersionIncompatible,
		},
		{
			name: "skill error with unknown reason is uncategorized",
			err:  &installer.SkillError{Skill: "databricks", Reason: "some-future-reason"},
			want: protos.AitoolsErrorCategoryUncategorized,
		},
		{
			name: "blocked error joined with another error is still classified",
			err:  errors.Join(&installer.BlockedError{Agent: "codex", Reason: installer.ReasonInstallFailed}, errors.New("other")),
			want: protos.AitoolsErrorCategoryPluginInstallFailed,
		},
		{
			name: "unrecognized error is uncategorized",
			err:  errors.New("boom"),
			want: protos.AitoolsErrorCategoryUncategorized,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyInstallError(tc.err))
		})
	}
}
