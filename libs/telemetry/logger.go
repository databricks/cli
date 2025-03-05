package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/google/uuid"
)

// Environment variable to disable telemetry. If this is set to any value, telemetry
// will be disabled.
const DisableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"

func Log(ctx context.Context, event protos.DatabricksCliLog) {
	fromContext(ctx).log(event)
}

func SetExecutionContext(ctx context.Context, ec protos.ExecutionContext) {
	fromContext(ctx).setExecutionContext(ec)
}

func HasLogs(ctx context.Context) bool {
	return len(fromContext(ctx).getLogs()) > 0
}

type logger struct {
	logs []protos.FrontendLog
}

func (l *logger) log(event protos.DatabricksCliLog) {
	if l.logs == nil {
		l.logs = make([]protos.FrontendLog, 0)
	}
	l.logs = append(l.logs, protos.FrontendLog{
		FrontendLogEventID: uuid.New().String(),
		Entry: protos.FrontendLogEntry{
			DatabricksCliLog: event,
		},
	})
}

func (l *logger) getLogs() []protos.FrontendLog {
	return l.logs
}

func (l *logger) setExecutionContext(ec protos.ExecutionContext) {
	for i := range l.logs {
		l.logs[i].Entry.DatabricksCliLog.ExecutionContext = &ec
	}
}

func Upload(ctx context.Context, cfg *config.Config) error {
	l := fromContext(ctx)
	if len(l.logs) == 0 {
		return errors.New("no logs to upload")
	}

	protoLogs := make([]string, len(l.logs))
	for i, log := range l.logs {
		b, err := json.Marshal(log)
		if err != nil {
			return fmt.Errorf("failed to marshal log: %s", err)
		}
		protoLogs[i] = string(b)
	}

	apiClient, err := client.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	ctx, _ = context.WithTimeout(ctx, 3*time.Second)
	var resp *ResponseBody

	// Only try uploading logs for a maximum of 3 times.
	for i := range 3 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("uploading telemetry logs timed out: %w", ctx.Err())
		default:
			// proceed
		}

		resp, err = attempt(ctx, apiClient, protoLogs)

		// All logs were uploaded successfully.
		if err == nil && resp.NumProtoSuccess >= int64(len(protoLogs)) {
			return nil
		}

		// Partial success. Retry.
		if err == nil && resp.NumProtoSuccess < int64(len(protoLogs)) {
			log.Debugf(ctx, "Attempt %d was a partial success. Number of logs uploaded: %d out of %d\n", i+1, resp.NumProtoSuccess, len(protoLogs))
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// We retry for all 5xx responses. We explicitly omit 503 in the predicate here
		// because it is already automatically retried in the SDK layer.
		// ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 && apiErr.StatusCode != 503 {
			log.Debugf(ctx, "Attempt %d failed due to a server side error. Retrying status code: %d\n", i+1, apiErr.StatusCode)
			time.Sleep(200 * time.Millisecond)
			continue
		}
	}

	return errors.New("failed to upload telemetry logs after three attempts")
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
