package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/spf13/cobra"

	"github.com/databricks/databricks-sdk-go/client"
)

// TODO CONTINUE:
// 2. Add end to end integration tests and mocked tests for telemetry upload.
// 3. Verify that tha auth configuration is resolved. Enforce that somehow?

// TODO: What happens here with OAuth? This then would end up spawning a new process
// to resolve the auth token. Options:
// 1. Check in with miles that this is fine.
// 2. See if we can directly pass a token with enough lifetime left to this
//    worker process.
// 3. Right before spawning the child process make sure to refresh the token.

// TODO: Print errors to stderr and assert in tests that the stderr is empty.

// We need to spawn a separate process to upload telemetry logs in order to avoid
// increasing the latency of CLI commands.
//
// TODO: Add check to ensure this does not become a fork bomb. Maybe a unit test
// as well.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "telemetry-worker",
		Short:  "Upload telemetry logs from stdin to Databricks",
		Args:   root.NoArgs,
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Running telemetry worker\n")

		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %s\n", err)
		}

		in := telemetry.WorkerInput{}
		err = json.Unmarshal(b, &in)
		if err != nil {
			return fmt.Errorf("failed to unmarshal input: %s\n", err)
		}

		fmt.Printf("worker input: %#v\n", in)

		logs := in.Logs

		// No logs to upload.
		if len(logs) == 0 {
			return fmt.Errorf("No logs to upload: %s\n", err)
		}

		// The API expects logs to be JSON strings. Serialize the logs to a string
		// to be set in the request body.
		var protoLogs []string
		for _, event := range logs {
			s, err := json.Marshal(event)
			if err != nil {
				return err
			}

			protoLogs = append(protoLogs, string(s))
		}

		apiClient, err := client.New(in.AuthConfig)
		if err != nil {
			return fmt.Errorf("Failed to create API client: %s\n", err)
		}

		// Set a maximum total time to try telemetry uploads.
		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()

		resp := &telemetry.ResponseBody{}
		for {
			select {
			case <-ctx.Done():
				return errors.New("Failed to flush telemetry log due to timeout")

			default:
				// Proceed
			}

			// Log the CLI telemetry events.
			err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", nil, telemetry.RequestBody{
				UploadTime: time.Now().Unix(),
				ProtoLogs:  protoLogs,

				// A bug in the telemetry API requires us to send an empty items array.
				// Otherwise we get an opaque 500 internal server error.
				// TODO: Do I need to do this even though "omitempty" is not set?
				Items: []string{},
			}, resp)
			if err != nil {
				// The SDK automatically performs retries for 429s and 503s. Thus if we
				// see an error here, do not retry logging the telemetry.
				return fmt.Errorf("Error making the API request to /telemetry-ext: %v", err)
			}

			// If not all the logs were successfully sent, we'll retry and log everything
			// again.
			//
			// Note: This will result in server side duplications but that's fine since
			// we can always deduplicate in the data pipeline itself.
			if len(logs) > int(resp.NumProtoSuccess) {
				continue
			}

			// TODO: Add an integration acceptance test for this.
			fmt.Println("Successfully flushed telemetry events")
			b, err := json.Marshal(resp)
			if err != nil {
				return fmt.Errorf("Failed to marshal response: %s\n", err)
			}
			fmt.Println("Response: ", string(b))
			return nil
		}
	}

	return cmd
}
