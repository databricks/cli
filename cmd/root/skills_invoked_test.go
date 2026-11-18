package root

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/stretchr/testify/assert"
)

func TestWithSkillsInvoked(t *testing.T) {
	tests := []struct {
		name             string
		skills           string
		disableTelemetry string
		want             []string
	}{
		{"unset", "", "", nil},
		{"set", "databricks-core,databricks-jobs", "", []string{"databricks-core", "databricks-jobs"}},
		{"telemetry disabled", "databricks-core", "1", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set both: an empty value in the context overrides the host environment.
			ctx := env.Set(t.Context(), envSkillsInvoked, tt.skills)
			ctx = env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", tt.disableTelemetry)
			assert.Equal(t, tt.want, skillsinvoked.FromContext(withSkillsInvoked(ctx)))
		})
	}
}
