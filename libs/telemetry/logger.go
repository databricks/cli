package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
)

const (
	// Environment variable to disable telemetry. If this is set to any value, telemetry
	// will be disabled.
	disableEnvVar = "DATABRICKS_CLI_DISABLE_TELEMETRY"

	// Timeout in seconds for uploading telemetry logs.
	timeoutEnvVar = "DATABRICKS_CLI_TELEMETRY_TIMEOUT"
)

func Log(ctx context.Context, event protos.DatabricksCliLog) {
	// A missing logger means telemetry was never initialized on this context
	// (e.g. a command invoked outside the normal cmd/root setup). Dropping the
	// event is the right call: telemetry is best-effort and must never crash a
	// command.
	l, ok := loggerFromContext(ctx)
	if !ok {
		log.Debugf(ctx, "telemetry logger not found in the context; dropping event")
		return
	}
	l.log(event)
}

type logger struct {
	logs             []protos.FrontendLog
	stateLineageHash string

	// running counts the goroutines that Upload started and stopped waiting
	// for, which may still be using the config (see within).
	running atomic.Int32
}

// SetStateLineageHash records the hashed lineage of the bundle state the command read, so that
// the command's execution context can report it. Like Log, it is a no-op without a logger.
func SetStateLineageHash(ctx context.Context, hash string) {
	if l, ok := loggerFromContext(ctx); ok {
		l.stateLineageHash = hash
	}
}

// StateLineageHash returns the value recorded by SetStateLineageHash, or "" if none was.
func StateLineageHash(ctx context.Context) string {
	if l, ok := loggerFromContext(ctx); ok {
		return l.stateLineageHash
	}
	return ""
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

	// hintOnlyUploadTimeout bounds an upload whose events are all install
	// hint events. Most commands that show a hint log nothing else, so
	// without the hint they would not upload at all; they wait less for it at
	// exit.
	hintOnlyUploadTimeout = 1 * time.Second

	waitBetweenRetries = 200 * time.Millisecond
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
			return fmt.Errorf("failed to marshal log: %w", err)
		}
		protoLogs[i] = string(b)
	}

	if !cmdctx.HasConfigUsed(ctx) {
		log.Debugf(ctx, "no auth config available; skipping telemetry upload")
		return nil
	}

	// The ssh agent shim uploads before it execs. If the exec fails, cmd/root
	// uploads again at exit, while a goroutine of the timed-out first upload
	// may still be resolving or authenticating the same config.
	if l.running.Load() > 0 {
		return errors.New("a previous telemetry upload is still running")
	}

	timeout, err := uploadTimeout(ctx, l.logs)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		log.Infof(ctx, "context has no deadline. This is unexpected. Not uploading telemetry logs")
		return nil
	}

	apiClient, err := within(ctx, &l.running, func() (*client.DatabricksClient, error) {
		return newAPIClient(cmdctx.ConfigUsed(ctx), &l.running)
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
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
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode >= 500 {
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

// uploadTimeout returns how long Upload may take: DATABRICKS_CLI_TELEMETRY_TIMEOUT
// (in seconds) when set, else hintOnlyUploadTimeout when every log is an
// install hint event, else defaultUploadTimeout.
func uploadTimeout(ctx context.Context, logs []protos.FrontendLog) (time.Duration, error) {
	if v := env.Get(ctx, timeoutEnvVar); v != "" {
		timeout, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse timeout: %w", err)
		}
		return time.Duration(timeout * float64(time.Second)), nil
	}
	if onlyHintEvents(logs) {
		return hintOnlyUploadTimeout, nil
	}
	return defaultUploadTimeout, nil
}

// onlyHintEvents reports whether every log is an install hint event.
func onlyHintEvents(logs []protos.FrontendLog) bool {
	for _, l := range logs {
		if l.Entry.DatabricksCliLog.AitoolsHintEvent == nil {
			return false
		}
	}
	return true
}

// within runs f on its own goroutine and returns its result, or ctx's error
// once ctx is done. Upload uses it for the SDK calls that ignore ctx: the SDK
// resolves the config (including host metadata discovery) and fetches auth
// tokens with context.Background(). The requests themselves honor ctx and run
// on the caller's goroutine. A panic in f is returned as an error, so it is
// logged like any other upload failure instead of crashing the process.
//
// After a timeout the goroutine keeps running until f returns or the process
// ends, and running counts it until then. It may still be writing to the
// config while resolving it, or refreshing an OAuth token that is lost if the
// process exits first. cmd/root runs Upload as the last step before the
// process exits; the ssh agent shim runs it just before exec, and Upload skips
// a second upload while running is non-zero.
func within[T any](ctx context.Context, running *atomic.Int32, f func() (T, error)) (T, error) {
	type result struct {
		v   T
		err error
	}
	resc := make(chan result, 1)
	running.Add(1)
	go func() {
		defer running.Add(-1)
		defer func() {
			if r := recover(); r != nil {
				resc <- result{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		v, err := f()
		resc <- result{v, err}
	}()
	select {
	case res := <-resc:
		return res.v, res.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// newAPIClient is client.New with authenticate as the auth visitor. The SDK
// runs the auth visitor on every request, including its own retries.
func newAPIClient(cfg *config.Config, running *atomic.Int32) (*client.DatabricksClient, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	clientCfg.AuthVisitor = func(r *http.Request) error {
		return authenticate(r, cfg, running)
	}
	return client.NewWithClient(cfg, httpclient.NewApiClient(clientCfg))
}

// authenticate sets the auth headers on r, giving up when r's context, which
// carries the upload deadline, is done. It authenticates a clone so that an
// abandoned goroutine never writes to r.
func authenticate(r *http.Request, cfg *config.Config, running *atomic.Int32) error {
	authReq := r.Clone(r.Context())
	_, err := within(r.Context(), running, func() (struct{}, error) {
		return struct{}{}, cfg.Authenticate(authReq)
	})
	if err != nil {
		return fmt.Errorf("authenticating telemetry upload: %w", err)
	}
	r.Header = authReq.Header
	return nil
}

func attempt(ctx context.Context, apiClient *client.DatabricksClient, protoLogs []string) (*ResponseBody, error) {
	// Without the workspace routing header, telemetry on unified hosts is
	// recorded in a central shard instead of the user's workspace.
	resp := &ResponseBody{}
	err := apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", auth.WorkspaceIDHeaders(apiClient.Config), nil, RequestBody{
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

// BufferedEvents returns the events logged on ctx, or nil when ctx has no
// logger. Upload does not clear them. Tests in other packages use it to assert
// what a command logs without uploading.
func BufferedEvents(ctx context.Context) []protos.DatabricksCliLog {
	l, ok := loggerFromContext(ctx)
	if !ok {
		return nil
	}
	out := make([]protos.DatabricksCliLog, len(l.logs))
	for i, fl := range l.logs {
		out[i] = fl.Entry.DatabricksCliLog
	}
	return out
}
