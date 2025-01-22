package telemetry_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// // Wrapper to capture the response from the API client since that's not directly
// // accessible from the logger.
// type apiClientWrapper struct {
// 	response  *telemetry.ResponseBody
// 	apiClient *client.DatabricksClient
// }
// §
// func (wrapper *apiClientWrapper) Do(ctx context.Context, method, path string,
// 	headers map[string]string, request, response any,
// 	visitors ...func(*http.Request) error,
// ) error {
// 	err := wrapper.apiClient.Do(ctx, method, path, headers, request, response, visitors...)
// 	wrapper.response = response.(*telemetry.ResponseBody)
// 	return err
// }

// func TestTelemetryLogger(t *testing.T) {
// 	events := []telemetry.DatabricksCliLog{
// 		{
// 			CliTestEvent: &protos.CliTestEvent{
// 				Name: protos.DummyCliEnumValue1,
// 			},
// 		},
// 		{
// 			BundleInitEvent: &protos.BundleInitEvent{
// 				Uuid:         uuid.New().String(),
// 				TemplateName: "abc",
// 				TemplateEnumArgs: []protos.BundleInitTemplateEnumArg{
// 					{
// 						Key:   "a",
// 						Value: "b",
// 					},
// 					{
// 						Key:   "c",
// 						Value: "d",
// 					},
// 				},
// 			},
// 		},
// 	}

// 	assert.Equal(t, len(events), reflect.TypeOf(telemetry.DatabricksCliLog{}).NumField(),
// 		"Number of events should match the number of fields in DatabricksCliLog. Please add a new event to this test.")

// 	ctx, w := acc.WorkspaceTest(t)
// 	ctx = telemetry.WithDefaultLogger(ctx)

// 	// Extend the maximum wait time for the telemetry flush just for this test.
// 	oldV := telemetry.MaxAdditionalWaitTime
// 	telemetry.MaxAdditionalWaitTime = 1 * time.Hour
// 	t.Cleanup(func() {
// 		telemetry.MaxAdditionalWaitTime = oldV
// 	})

// 	for _, event := range events {
// 		telemetry.Log(ctx, event)
// 	}

// 	apiClient, err := client.New(w.W.Config)
// 	require.NoError(t, err)

// 	// Flush the events.
// 	wrapper := &apiClientWrapper{
// 		apiClient: apiClient,
// 	}
// 	telemetry.Flush(ctx, wrapper)

// 	// Assert that the events were logged.
// 	assert.Equal(t, telemetry.ResponseBody{
// 		NumProtoSuccess: int64(len(events)),
// 		Errors:          []telemetry.LogError{},
// 	}, *wrapper.response)
// }

func TestTelemetryCliPassesAuthCredentials(t *testing.T) {
	server := testutil.StartServer(t)
	count := int64(0)

	server.Handle("POST /telemetry-ext", func(r *http.Request) (any, error) {
		// auth token should be set. Since the telemetry-worker command does not
		// load the profile, the token must have been passed explicitly.
		require.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))

		// reqBody should contain all the logs.
		reqBody := telemetry.RequestBody{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		logs := []protos.FrontendLog{}
		for _, log := range reqBody.ProtoLogs {
			var l protos.FrontendLog
			err := json.Unmarshal([]byte(log), &l)
			require.NoError(t, err)

			logs = append(logs, l)
		}

		assert.Len(t, logs, 3)
		assert.Equal(t, protos.DummyCliEnum("VALUE1"), logs[0].Entry.DatabricksCliLog.CliTestEvent.Name)
		assert.Equal(t, protos.DummyCliEnum("VALUE2"), logs[1].Entry.DatabricksCliLog.CliTestEvent.Name)
		assert.Equal(t, protos.DummyCliEnum("VALUE3"), logs[2].Entry.DatabricksCliLog.CliTestEvent.Name)

		count++

		// TODO: Add (or keep) the API testing tests that ensure that the telemetry API is working correctly.
		return telemetry.ResponseBody{
			NumProtoSuccess: count,
		}, nil
	})

	// Setup databrickscfg file.
	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, ".databrickscfg"), fmt.Sprintf(`
[myprofile]
host = %s
token = mytoken`, server.URL))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(tmpDir, ".databrickscfg"))
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "myprofile")

	execPath := testutil.BuildCLI(t)
	cmd := exec.Command(execPath, "send-test-event")
	err := cmd.Run()
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return count == 3
	}, 10*time.Second, 1*time.Second)
}

func TestTelemetry(t *testing.T) {
	server := testutil.StartServer(t)
	count := int64(0)

	server.Handle("POST /telemetry-ext", func(r *http.Request) (any, error) {
		// auth token should be set.
		require.Equal(t, "Bearer foobar", r.Header.Get("Authorization"))

		// reqBody should contain all the logs.
		reqBody := telemetry.RequestBody{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		logs := []protos.FrontendLog{}
		for _, log := range reqBody.ProtoLogs {
			var l protos.FrontendLog
			err := json.Unmarshal([]byte(log), &l)
			require.NoError(t, err)

			logs = append(logs, l)
		}

		assert.Len(t, logs, 3)
		assert.Equal(t, protos.DummyCliEnum("VALUE1"), logs[0].Entry.DatabricksCliLog.CliTestEvent.Name)
		assert.Equal(t, protos.DummyCliEnum("VALUE2"), logs[1].Entry.DatabricksCliLog.CliTestEvent.Name)
		assert.Equal(t, protos.DummyCliEnum("VALUE3"), logs[2].Entry.DatabricksCliLog.CliTestEvent.Name)

		count++

		// TODO: Add (or keep) the API testing tests that ensure that the telemetry API is working correctly.
		return telemetry.ResponseBody{
			NumProtoSuccess: count,
		}, nil
	})

	// TODO: Also see how much extra time does spawning a process take?
	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "foobar")

	execPath := testutil.BuildCLI(t)
	cmd := exec.Command(execPath, "send-test-event")
	err := cmd.Run()
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return count == 3
	}, 10*time.Second, 1*time.Second)
}

func TestTelemetryDoesNotBlock(t *testing.T) {
	server := testutil.StartServer(t)
	count := int64(0)

	fire := make(chan struct{})

	server.Handle("POST /telemetry-ext", func(r *http.Request) (any, error) {
		// Block until the channel is closed.
		<-fire

		require.Equal(t, "Bearer foobar", r.Header.Get("Authorization"))

		count++
		return telemetry.ResponseBody{
			NumProtoSuccess: 3,
		}, nil
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "foobar")

	execPath := testutil.BuildCLI(t)
	cmd := exec.Command(execPath, "send-test-event")
	err := cmd.Run()
	require.NoError(t, err)

	// No API calls should have been made yet. Even though the main process has
	// finished, the telemetry worker should be running in the background.
	assert.Equal(t, int64(0), count)

	// Close the channel to allow the API call to go through.
	close(fire)
	assert.Eventually(t, func() bool {
		return count == 1
	}, 10*time.Second, 1*time.Second)
}
