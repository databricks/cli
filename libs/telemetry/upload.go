package telemetry

import (
	"context"
	"encoding/json"
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
	// File containing output from the upload process.
	UploadLogsFileEnvVar = "DATABRICKS_CLI_TELEMETRY_UPLOAD_LOGS_FILE"

	// File containing the PID of the telemetry upload process.
	PidFileEnvVar = "DATABRICKS_CLI_TELEMETRY_PID_FILE"

	// Environment variable to disable telemetry. If this is set to any value, telemetry
	// will be disabled.
	DisableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"

	// Max time to try and upload the telemetry logs. Useful for testing.
	UploadTimeoutEnvVar = "DATABRICKS_CLI_TELEMETRY_UPLOAD_TIMEOUT"
)

type UploadConfig struct {
	Logs []protos.FrontendLog `json:"logs"`
}

// Upload reads telemetry logs from stdin and uploads them to the telemetry endpoint.
// This function is always expected to be called in a separate child process from
// the main CLI process.
func Upload(ctx context.Context) (*ResponseBody, error) {
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
	if v, ok := os.LookupEnv(UploadTimeoutEnvVar); ok {
		maxUploadTime, err = time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse time limit %s: %s\n", UploadTimeoutEnvVar, err)
		}
	}

	// Set a maximum total time to try telemetry uploads.
	ctx, cancel := context.WithTimeout(ctx, maxUploadTime)
	defer cancel()

	resp := &ResponseBody{}

	// Retry uploading logs a maximum of 3 times incase the uploads are partially successful.
	for range 3 {
		// Log the CLI telemetry events.
		err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, nil, RequestBody{
			UploadTime: time.Now().UnixMilli(),
			// There is a bug in the `/telemetry-ext` API which requires us to
			// send an empty array for the `Items` field. Otherwise the API returns
			// a 500.
			Items:     []string{},
			ProtoLogs: protoLogs,
		}, resp)
		if err != nil {
			return nil, fmt.Errorf("Failed to upload telemetry logs: %s\n", err)
		}

		// Skip retrying if the upload fails with an error.
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("Failed to upload telemetry logs: %s\n", resp.Errors)
		}

		// All logs were uploaded successfully.
		if resp.NumProtoSuccess == int64(len(in.Logs)) {
			return resp, nil
		}

		// Add a delay of 1 second before retrying. We avoid retrying immediately
		// to avoid overwhelming the telemetry endpoint.
		// We only return incase of partial successful uploads. The SDK layer takes
		// care of retrying in case of retriable status codes.
		//
		// TODO: I think I was wrong about the SDKs automatically doing retries.
		// Look into this more and confirm with ankit what the 5xx status codes are.
		// TODO: Confirm that the timeout of a request here is indeed one minute.
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("Failed to upload all telemetry logs after 4 tries. Only %d/%d logs uploaded", resp.NumProtoSuccess, len(in.Logs))
}
