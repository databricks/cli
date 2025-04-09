package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/google/uuid"
)

const (
	// Environment variable to disable telemetry. If this is set to any value, telemetry
	// will be disabled.
	disableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"

	// Timeout in seconds for uploading telemetry logs.
	timeoutEnvVar = "DATABRICKS_CLI_TELEMETRY_TIMEOUT"
)

func Log(ctx context.Context, event protos.DatabricksCliLog) {
	fromContext(ctx).log(event)
}

type logger struct {
	logs []protos.FrontendLog
}

func (l *logger) log(event protos.DatabricksCliLog) {
	l.logs = append(l.logs, protos.FrontendLog{
		FrontendLogEventID: uuid.New().String(),
		Entry: protos.FrontendLogEntry{
			DatabricksCliLog: event,
		},
	})
}

const (
	defaultUploadTimeout = 3 * time.Second
	waitBetweenRetries   = 200 * time.Millisecond
)

func Upload(ctx context.Context, ec protos.ExecutionContext) error {
	l := fromContext(ctx)
	if len(l.logs) == 0 {
		log.Debugf(ctx, "no telemetry logs to upload")
		return nil
	}

	// Telemetry is disabled. We don't upload logs.
	if env.Get(ctx, disableEnvVar) != "" {
		log.Debugf(ctx, "telemetry upload is disabled. Not uploading any logs.")
		return nil
	}

	// Set the execution context for all logs.
	for i := range l.logs {
		l.logs[i].Entry.DatabricksCliLog.ExecutionContext = &ec
	}

	protoLogs := make([]string, len(l.logs))
	for i, log := range l.logs {
		b, err := json.Marshal(log)
		if err != nil {
			return fmt.Errorf("failed to marshal log: %s", err)
		}
		protoLogs[i] = string(b)
	}

	apiClient, err := client.New(cmdctx.ConfigUsed(ctx))
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	uploadTimeout := defaultUploadTimeout
	if v := env.Get(ctx, timeoutEnvVar); v != "" {
		timeout, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("failed to parse timeout: %w", err)
		}

		uploadTimeout = time.Duration(timeout * float64(time.Second))
	}

	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		log.Infof(ctx, "context has no deadline. This is unexpected. Not uploading telemetry logs")
		return nil
	}

	// Only try uploading logs for a maximum of 3 times.
	for i := range 3 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("uploading telemetry logs timed out: %w", ctx.Err())
		default:
			// proceed
		}

		resp, err := attempt(ctx, apiClient, protoLogs)

		// All logs were uploaded successfully.
		if err == nil && resp.NumProtoSuccess >= int64(len(protoLogs)) {
			log.Debugf(ctx, "All %d logs uploaded successfully", len(protoLogs))
			return nil
		}

		// Partial success. Retry.
		if err == nil && resp.NumProtoSuccess < int64(len(protoLogs)) {
			log.Debugf(ctx, "Attempt %d was a partial success. Number of logs uploaded: %d out of %d", i+1, resp.NumProtoSuccess, len(protoLogs))

			remainingTime := time.Until(deadline)
			if remainingTime < waitBetweenRetries {
				log.Debugf(ctx, "not enough time to retry before context deadline.")
				break
			}

			time.Sleep(waitBetweenRetries)
			continue
		}

		// Do not retry if the context deadline was exceeded. This means that our
		// timeout of three seconds was triggered and we should not try again.
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debugf(ctx, "Attempt %d failed due to a timeout. Will not retry", i+1)
			return fmt.Errorf("uploading telemetry logs timed out: %w", err)
		}

		// We retry for all 5xx responses. Note that the SDK only retries for 503 and 429
		// (as of 6th March 2025) so we need some additional logic here to retry for other
		// 5xx responses.
		// Note: We should never see a 503 or 429 here because the SDK default timeout
		// of 1 minute is more than the 3 second timeout we set above.
		//
		// SDK ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
		//
		// The UI infra team (who owns the /telemetry-ext API) recommends retrying for
		// all 5xx responses.
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
			log.Infof(ctx, "Attempt %d failed due to a server side error. Retrying status code: %d", i+1, apiErr.StatusCode)

			remainingTime := time.Until(deadline)
			if remainingTime < waitBetweenRetries {
				log.Debugf(ctx, "not enough time to retry before context deadline.")
				break
			}

			time.Sleep(waitBetweenRetries)
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
