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

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	// File containing debug logs from the upload process.
	UploadLogsFileEnvVar = "DATABRICKS_CLI_TELEMETRY_UPLOAD_LOGS_FILE"

	// File containing the PID of the telemetry upload process.
	PidFileEnvVar = "DATABRICKS_CLI_TELEMETRY_PID_FILE"

	// Environment variable to disable telemetry. If this is set to any value, telemetry
	// will be disabled.
	DisableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"

	// Max time to try and upload the telemetry logs. Useful for testing.
	UploadTimeout = "DATABRICKS_CLI_TELEMETRY_UPLOAD_TIMEOUT"
)

type UploadConfig struct {
	Logs []protos.FrontendLog `json:"logs"`
}

// Upload reads telemetry logs from stdin and uploads them to the telemetry endpoint.
// This function is always expected to be called in a separate child process from
// the main CLI process.
func Upload() (*ResponseBody, error) {
	var err error

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %s\n", err)
	}

	in := UploadConfig{}
	err = json.Unmarshal(b, &in)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %s\n", err)
	}

	if len(in.Logs) == 0 {
		return nil, fmt.Errorf("No logs to upload: %s\n", err)
	}

	protoLogs := make([]string, len(in.Logs))
	for i, log := range in.Logs {
		b, err := json.Marshal(log)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal log: %s\n", err)
		}
		protoLogs[i] = string(b)
	}

	// Parent process is responsible for setting environment variables to
	// configure authentication.
	apiClient, err := client.New(&config.Config{})
	if err != nil {
		return nil, fmt.Errorf("Failed to create API client: %s\n", err)
	}

	maxUploadTime := 30 * time.Second
	if v, ok := os.LookupEnv(UploadTimeout); ok {
		maxUploadTime, err = time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse time limit %s: %s\n", UploadTimeout, err)
		}
	}

	// Set a maximum total time to try telemetry uploads.
	ctx, cancel := context.WithTimeout(context.Background(), maxUploadTime)
	defer cancel()

	resp := &ResponseBody{}
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Failed to flush telemetry log due to timeout")

		default:
			// Proceed
		}

		// Log the CLI telemetry events.
		err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, nil, RequestBody{
			UploadTime: time.Now().UnixMilli(),
			Items:      []string{},
			ProtoLogs:  protoLogs,
		}, resp)
		if err != nil {
			return nil, fmt.Errorf("Failed to upload telemetry logs: %s\n", err)
		}

		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("Failed to upload telemetry logs: %s\n", resp.Errors)
		}

		if resp.NumProtoSuccess == int64(len(in.Logs)) {
			return resp, nil
		}
	}
}
