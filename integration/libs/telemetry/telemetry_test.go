package telemetry

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
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
	// The SDK retries 429 responses for its full retry timeout (5 minutes).
	// A 429 that outlasts it means the shared test workspace is throttling
	// /telemetry-ext because of traffic from other concurrent CI runs; it
	// says nothing about whether the endpoint accepts these logs.
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.StatusCode == http.StatusTooManyRequests {
		t.Skipf("/telemetry-ext is rate limited on this workspace: %s", aerr.Message)
	}
	require.NoError(t, err)

	assert.Equal(t, telemetry.ResponseBody{
		Errors:          []telemetry.LogError{},
		NumProtoSuccess: int64(2),
	}, respB)
}
