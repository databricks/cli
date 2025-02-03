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
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

// File containing debug logs from the upload process.
const UploadLogsFileEnvVar = "DATABRICKS_TELEMETRY_UPLOAD_LOGS_FILE"

type UploadConfig struct {
	Logs []protos.FrontendLog `json:"logs"`
}

// Upload reads telemetry logs from stdin and uploads them to the telemetry endpoint.
// This function is always expected to be called in a separate child process from
// the main CLI process.
func Upload() error {
	var err error

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %s\n", err)
	}

	in := UploadConfig{}
	err = json.Unmarshal(b, &in)
	if err != nil {
		printJson(in)
		return fmt.Errorf("failed to unmarshal input: %s\n", err)
	}

	if len(in.Logs) == 0 {
		return fmt.Errorf("No logs to upload: %s\n", err)
	}

	protoLogs := make([]string, len(in.Logs))
	for i, log := range in.Logs {
		b, err := json.Marshal(log)
		if err != nil {
			return fmt.Errorf("failed to marshal log: %s\n", err)
		}
		protoLogs[i] = string(b)
	}

	// Parent process is responsible for setting environment variables to
	// configure authentication.
	apiClient, err := client.New(&config.Config{})
	if err != nil {
		return fmt.Errorf("Failed to create API client: %s\n", err)
	}

	// Set a maximum total time to try telemetry uploads.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp := &ResponseBody{}
	for {
		select {
		case <-ctx.Done():
			return errors.New("Failed to flush telemetry log due to timeout")

		default:
			// Proceed
		}

		// Log the CLI telemetry events.
		err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, nil, RequestBody{
			UploadTime: time.Now().UnixMilli(),
			Items:      []string{},
			ProtoLogs:  protoLogs,
		}, resp)
		var apierr *apierr.APIError
		if errors.As(err, &apierr) && apierr.StatusCode == http.StatusNotFound {
			return errors.New("telemetry endpoint not found")
		}
		if err != nil {
			return fmt.Errorf("Failed to upload telemetry logs: %s\n", err)
		}

		if len(resp.Errors) > 0 {
			return fmt.Errorf("Failed to upload telemetry logs: %s\n", resp.Errors)
		}

		if resp.NumProtoSuccess == int64(len(in.Logs)) {
			fmt.Println("Successfully uploaded telemetry logs")
			fmt.Println("Response: ")
			printJson(resp)
			break
		}
	}

	return nil
}

func printJson(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("failed to marshal JSON: %s\n", err)
	}
	fmt.Println(string(b))
}
