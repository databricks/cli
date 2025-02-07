package worker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

func TestTelemetryWorker(t *testing.T) {
	server := testutil.StartServer(t)
	count := int64(0)

	server.Handle("POST /telemetry-ext", func(r *http.Request) (any, error) {
		// auth token should be set.
		require.Equal(t, "Bearer foobar", r.Header.Get("Authorization"))

		// reqBody should contain all the logs.
		reqBody := telemetry.RequestBody{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		require.Equal(t, []string{
			"{\"frontend_log_event_id\":\"aaaa\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE1\"}}}}",
			"{\"frontend_log_event_id\":\"bbbb\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE2\"}}}}",
			"{\"frontend_log_event_id\":\"cccc\",\"entry\":{\"databricks_cli_log\":{\"cli_test_event\":{\"name\":\"VALUE3\"}}}}",
		}, reqBody.ProtoLogs)

		count++
		return telemetry.ResponseBody{
			NumProtoSuccess: count,
		}, nil
	})

	in := telemetry.WorkerInput{
		Logs: []protos.FrontendLog{
			{
				FrontendLogEventID: "aaaa",
				Entry: protos.FrontendLogEntry{
					DatabricksCliLog: protos.DatabricksCliLog{
						CliTestEvent: &protos.CliTestEvent{
							Name: protos.DummyCliEnumValue1,
						},
					},
				},
			},
			{
				FrontendLogEventID: "bbbb",
				Entry: protos.FrontendLogEntry{
					DatabricksCliLog: protos.DatabricksCliLog{
						CliTestEvent: &protos.CliTestEvent{
							Name: protos.DummyCliEnumValue2,
						},
					},
				},
			},
			{
				FrontendLogEventID: "cccc",
				Entry: protos.FrontendLogEntry{
					DatabricksCliLog: protos.DatabricksCliLog{
						CliTestEvent: &protos.CliTestEvent{
							Name: protos.DummyCliEnumValue3,
						},
					},
				},
			},
		},
		AuthConfig: &config.Config{
			Host:  server.URL,
			Token: "foobar",
		},
	}

	inBytes, err := json.Marshal(in)
	require.NoError(t, err)

	stdinReader := bytes.NewReader(inBytes)

	cmd := New()
	cmd.SetIn(stdinReader)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Telemetry worker should retry until all logs are uploaded.
	require.Equal(t, int64(3), count)
}
