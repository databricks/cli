package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestLogDeployTelemetryRecordsDeploymentHistory(t *testing.T) {
	tests := []struct {
		name        string
		engine      engine.EngineType
		wantEnabled bool
	}{
		{
			name:        "deployment history enabled",
			engine:      engine.EngineDirectWithHistory,
			wantEnabled: true,
		},
		{
			name:        "deployment history disabled",
			engine:      engine.EngineDirect,
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := telemetry.WithNewLogger(t.Context())
			b := &bundle.Bundle{}
			b.Config.Bundle.Engine = tt.engine

			LogDeployTelemetry(ctx, b, "")

			assert.Contains(t, b.Metrics.BoolValues, protos.BoolMapEntry{
				Key:   metrics.DeploymentHistoryEnabled,
				Value: tt.wantEnabled,
			})
		})
	}
}
