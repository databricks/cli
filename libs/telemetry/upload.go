package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	// File containing output from the upload process.
	UploadLogsFileEnvVar = "DATABRICKS_CLI_TELEMETRY_LOGFILE"

	// File containing the PID of the telemetry upload process.
	PidFileEnvVar = "DATABRICKS_CLI_TELEMETRY_PIDFILE"

	// Environment variable to disable telemetry. If this is set to any value, telemetry
	// will be disabled.
	DisableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"
)

type UploadConfig struct {
	Logs []protos.FrontendLog `json:"logs"`
}

// The API requires the logs to be JSON encoded strings. This function reads the
// logs from stdin and returns them as a slice of JSON encoded strings.
func readLogs(stdin io.Reader) ([]string, error) {
	b, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %s", err)
	}

	in := UploadConfig{}
	err = json.Unmarshal(b, &in)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %s", err)
	}

	if len(in.Logs) == 0 {
		return nil, errors.New("No logs to upload")
	}

	protoLogs := make([]string, len(in.Logs))
	for i, log := range in.Logs {
		b, err := json.Marshal(log)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal log: %s", err)
		}
		protoLogs[i] = string(b)
	}

	return protoLogs, nil
}

// Upload reads telemetry logs from stdin and uploads them to the telemetry endpoint.
// This function is always expected to be called in a separate child process from
// the main CLI process.
func Upload(ctx context.Context) (*ResponseBody, error) {
	logs, err := readLogs(os.Stdin)
	if err != nil {
		return nil, err
	}

	// Parent process is responsible for setting environment variables to
	// configure authentication.
	apiClient, err := client.New(&config.Config{})
	if err != nil {
		return nil, fmt.Errorf("Failed to create API client: %s\n", err)
	}

	var resp *ResponseBody

	// Only try uploading logs for a maximum of 3 times.
	for i := range 3 {
		resp, err = attempt(ctx, apiClient, logs)

		// All logs were uploaded successfully.
		if err == nil && resp.NumProtoSuccess >= int64(len(logs)) {
			return resp, nil
		}

		// Partial success. Retry.
		if err == nil && resp.NumProtoSuccess < int64(len(logs)) {
			log.Warnf(ctx, "Attempt %d was a partial success. Number of logs uploaded: %d out of %d\n", i+1, resp.NumProtoSuccess, len(logs))
			time.Sleep(2 * time.Second)
			continue
		}

		// We retry for all 5xx responses. We explicitly omit 503 in the predicate here
		// because it is already automatically retried in the SDK layer.
		// ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 && apiErr.StatusCode != 503 {
			log.Warnf(ctx, "Attempt %d failed due to a server side error. Retrying status code: %d\n", i+1, apiErr.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}
	}

	return resp, errors.New("upload did not succeed after three attempts")
}

func attempt(ctx context.Context, apiClient *client.DatabricksClient, protoLogs []string) (*ResponseBody, error) {
	resp := &ResponseBody{}
	err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, nil, RequestBody{
		UploadTime: time.Now().UnixMilli(),
		// There is a bug in the `/telemetry-ext` API which requires us to
		// send an empty array for the `Items` field. Otherwise the API returns
		// a 500.
		Items:     []string{},
		ProtoLogs: protoLogs,
	}, resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("uploading telemetry failed: %v", resp.Errors)
	}

	return resp, nil
}
