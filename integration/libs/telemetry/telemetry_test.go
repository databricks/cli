package telemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryEndpoint(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	apiClient, err := client.New(w.Config)
	require.NoError(t, err)

	logs := []protos.FrontendLog{
		{
			FrontendLogEventID: uuid.New().String(),
			Entry: protos.FrontendLogEntry{
				DatabricksCliLog: protos.DatabricksCliLog{
					CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1},
				},
			},
		},
		{
			FrontendLogEventID: uuid.New().String(),
			Entry: protos.FrontendLogEntry{
				DatabricksCliLog: protos.DatabricksCliLog{
					CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue2},
				},
			},
		},
	}

	protoLogs := make([]string, len(logs))
	for i, log := range logs {
		b, err := json.Marshal(log)
		require.NoError(t, err)
		protoLogs[i] = string(b)
	}

	reqB := telemetry.RequestBody{
		UploadTime: time.Now().UnixMilli(),
		Items:      []string{},
		ProtoLogs:  protoLogs,
	}

	respB := telemetry.ResponseBody{}

	err = apiClient.Do(ctx, "POST", "/telemetry-ext", nil, nil, reqB, &respB)
	require.NoError(t, err)

	assert.Equal(t, telemetry.ResponseBody{
		Errors:          []telemetry.LogError{},
		NumProtoSuccess: int64(2),
	}, respB)
}
