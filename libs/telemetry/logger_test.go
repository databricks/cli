package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// longUploadTimeout replaces the default upload timeout in tests that check
// something other than the timeout. On a stalled CI runner, creating the client
// alone has taken over 3s, failing the upload with a timeout instead.
const longUploadTimeout = "60"

// uploadWatchdog is how long a test waits for an Upload or Send that must stop
// at its deadline before failing as hung. A runner stall fires that deadline and
// a short watchdog at the same moment, so it is far longer than any stall seen.
const uploadWatchdog = 60 * time.Second

func TestTelemetryUploadRetriesOnPartialSuccess(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return ResponseBody{
				NumProtoSuccess: 1,
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	ctx := WithNewLogger(t.Context())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})

	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})
	ctx = env.Set(ctx, timeoutEnvVar, longUploadTimeout)

	err := Upload(ctx, protos.ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func uploadRetriesFor(t *testing.T, statusCode int) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return testserver.Response{
				StatusCode: statusCode,
				Body: apierr.APIError{
					StatusCode: statusCode,
					Message:    "Some error",
				},
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

	ctx := WithNewLogger(t.Context())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})
	ctx = env.Set(ctx, timeoutEnvVar, longUploadTimeout)

	err := Upload(ctx, protos.ExecutionContext{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestTelemetryUploadRetriesForStatusCodes(t *testing.T) {
	// These retries happen in the CLI itself since the SDK does not automatically
	// retry for 5xx errors.
	uploadRetriesFor(t, 500)
	uploadRetriesFor(t, 504)

	// These retries happen on the SDK layer.
	// ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
	uploadRetriesFor(t, 503)
	uploadRetriesFor(t, 429)
}

func TestTelemetryUploadMaxRetries(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	count := 0

	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		return ResponseBody{
			NumProtoSuccess: 1,
		}
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")
	ctx := WithNewLogger(t.Context())

	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue2,
		},
	})

	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
	})
	ctx = env.Set(ctx, timeoutEnvVar, longUploadTimeout)

	err := Upload(ctx, protos.ExecutionContext{})
	assert.EqualError(t, err, "failed to upload telemetry logs after three attempts")
	assert.Equal(t, 3, count)
}

// The SDK fetches tokens with context.Background(), so a token source that
// retries ignores the deadline Upload passes to the request. The upload
// timeout must still bound Upload.
func TestTelemetryUploadTimeoutBoundsTokenAcquisition(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	ts := auth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		<-release
		return nil, errors.New("token endpoint unavailable")
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "0.1")
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:        server.URL,
		Credentials: config.NewTokenSourceStrategy("blocking", ts),
	})

	assertUploadTimesOut(t, ctx)
}

// A token that expires between attempts is fetched again on the retry; that
// fetch must be bounded by the upload timeout too.
func TestTelemetryUploadTimeoutBoundsTokenRefreshOnRetry(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// The token expires 500ms after it is issued, so the time Upload takes to
	// get the first token does not matter.
	var expiry atomic.Pointer[time.Time]
	var count atomic.Int32
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count.Add(1)
		// Fail the first attempt once the token has expired, so the retry
		// has to fetch a new one.
		time.Sleep(time.Until(*expiry.Load()) + 50*time.Millisecond)
		return testserver.Response{
			StatusCode: 500,
			Body:       apierr.APIError{StatusCode: 500, Message: "Some error"},
		}
	})

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	var calls atomic.Int32
	ts := auth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		if calls.Add(1) == 1 {
			e := time.Now().Add(500 * time.Millisecond)
			expiry.Store(&e)
			return &oauth2.Token{AccessToken: "token", Expiry: e}, nil
		}
		<-release
		return nil, errors.New("token endpoint unavailable")
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "2")
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:        server.URL,
		Credentials: config.NewTokenSourceStrategy("expiring", ts),
	})

	assertUploadTimesOut(t, ctx)
	assert.Equal(t, int32(1), count.Load())
}

// A config that the command never resolved is resolved when Upload creates
// its client, which includes host metadata discovery with
// context.Background(). The upload timeout must bound that too.
func TestTelemetryUploadTimeoutBoundsConfigResolution(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "0.1")
	Log(ctx, protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{
			Name: protos.DummyCliEnumValue1,
		},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:  server.URL,
		Token: "token",
		HostMetadataResolver: func(context.Context, string) (*config.HostMetadata, error) {
			<-release
			return nil, errors.New("host unreachable")
		},
	})

	assertUploadTimesOut(t, ctx)
}

// assertUploadTimesOut runs Upload and asserts it fails with its timeout,
// failing the test instead of hanging if Upload does not return.
func assertUploadTimesOut(t *testing.T, ctx context.Context) {
	errc := make(chan error, 1)
	go func() { errc <- Upload(ctx, protos.ExecutionContext{}) }()
	select {
	case err := <-errc:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(uploadWatchdog):
		t.Fatal("Upload did not return after its timeout")
	}
}

func TestLogWithoutLoggerDropsEvent(t *testing.T) {
	// A context without a telemetry logger must not panic; the event is dropped.
	assert.NotPanics(t, func() {
		Log(t.Context(), protos.DatabricksCliLog{
			CliTestEvent: &protos.CliTestEvent{
				Name: protos.DummyCliEnumValue1,
			},
		})
	})
}

func TestStateLineageHash(t *testing.T) {
	ctx := WithNewLogger(t.Context())
	assert.Empty(t, StateLineageHash(ctx))

	SetStateLineageHash(ctx, "G0ZNUjK18Pa")
	assert.Equal(t, "G0ZNUjK18Pa", StateLineageHash(ctx))
}

func TestStateLineageHashWithoutLogger(t *testing.T) {
	SetStateLineageHash(t.Context(), "G0ZNUjK18Pa")
	assert.Empty(t, StateLineageHash(t.Context()))
}

func TestBufferedEvents(t *testing.T) {
	assert.Nil(t, BufferedEvents(t.Context()))

	ctx := WithNewLogger(t.Context())
	assert.Empty(t, BufferedEvents(ctx))
	Log(ctx, protos.DatabricksCliLog{AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: protos.AitoolsHintKindInstallText}})
	got := BufferedEvents(ctx)
	require.Len(t, got, 1)
	assert.Equal(t, protos.AitoolsHintKindInstallText, got[0].AitoolsHintEvent.HintKind)
}

func TestUploadTimeout(t *testing.T) {
	hint := protos.FrontendLog{Entry: protos.FrontendLogEntry{DatabricksCliLog: protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: protos.AitoolsHintKindInstallText},
	}}}
	other := protos.FrontendLog{Entry: protos.FrontendLogEntry{DatabricksCliLog: protos.DatabricksCliLog{
		CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1},
	}}}

	for _, tc := range []struct {
		name string
		env  string
		logs []protos.FrontendLog
		want time.Duration
	}{
		{name: "only hint events", logs: []protos.FrontendLog{hint, hint}, want: hintOnlyUploadTimeout},
		{name: "hint and other events", logs: []protos.FrontendLog{hint, other}, want: defaultUploadTimeout},
		{name: "other events", logs: []protos.FrontendLog{other}, want: defaultUploadTimeout},
		{name: "env overrides hint events", env: "0.5", logs: []protos.FrontendLog{hint}, want: 500 * time.Millisecond},
		{name: "env overrides other events", env: "5", logs: []protos.FrontendLog{other}, want: 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), timeoutEnvVar, tc.env)
			got, err := uploadTimeout(ctx, tc.logs)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	_, err := uploadTimeout(env.Set(t.Context(), timeoutEnvVar, "soon"), []protos.FrontendLog{other})
	assert.ErrorContains(t, err, "failed to parse timeout")
}

// An upload of only hint events gives up after hintOnlyUploadTimeout, not
// the default timeout.
func TestTelemetryUploadHintOnlyUsesShortTimeout(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		<-release
		return ResponseBody{NumProtoSuccess: 1}
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "")
	Log(ctx, protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: protos.AitoolsHintKindInstallText},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})

	start := time.Now()
	err := Upload(ctx, protos.ExecutionContext{})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), defaultUploadTimeout)
}

// A hint-only upload runs for a config that never authenticated: inside a
// bundle directory the config stored for telemetry is not the one the
// command's client authenticated, so it has no resolved credentials yet.
func TestTelemetryUploadHintOnlyWithUnresolvedConfig(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var calls atomic.Int32
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		calls.Add(1)
		return ResponseBody{NumProtoSuccess: 1}
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, longUploadTimeout)
	Log(ctx, protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: protos.AitoolsHintKindInstallText},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})

	require.NoError(t, Upload(ctx, protos.ExecutionContext{}))
	assert.Equal(t, int32(1), calls.Load())
}

// A hint-only upload whose credentials keep failing (an auth type preset by
// the profile, with a token source that cannot refresh) gives up within the
// hint-only timeout and sends nothing.
func TestTelemetryUploadHintOnlyWithFailingAuthIsBounded(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var calls atomic.Int32
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		calls.Add(1)
		return ResponseBody{NumProtoSuccess: 1}
	})

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	ts := auth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		<-release
		return nil, errors.New("refresh token revoked")
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "")
	Log(ctx, protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: protos.AitoolsHintKindInstallText},
	})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:        server.URL,
		AuthType:    "databricks-cli",
		Credentials: config.NewTokenSourceStrategy("databricks-cli", ts),
	})

	start := time.Now()
	err := Upload(ctx, protos.ExecutionContext{})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), defaultUploadTimeout)
	assert.Equal(t, int32(0), calls.Load())
}

// While a timed-out upload's goroutine may still use the config, a second
// upload on the same logger does not touch it.
func TestTelemetryUploadSkipsWhileEarlierUploadRuns(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	ts := auth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		<-release
		return nil, errors.New("token endpoint unavailable")
	})

	ctx := WithNewLogger(t.Context())
	ctx = env.Set(ctx, timeoutEnvVar, "0.1")
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{
		Host:        server.URL,
		Credentials: config.NewTokenSourceStrategy("blocking", ts),
	})

	assertUploadTimesOut(t, ctx)
	err := Upload(ctx, protos.ExecutionContext{})
	assert.ErrorContains(t, err, "a previous telemetry upload is still running")
}

func TestWithinReturnsPanicAsError(t *testing.T) {
	var running atomic.Int32
	_, err := within(t.Context(), &running, func() (int, error) { panic("boom") })
	assert.EqualError(t, err, "panic: boom")
	assert.Eventually(t, func() bool { return running.Load() == 0 }, time.Second, time.Millisecond)
}

func TestNotSent(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "never sent", err: fmt.Errorf("wrapped: %w", &notSentError{errors.New("no client")}), want: true},
		{name: "nothing accepted", err: &notAcceptedError{Accepted: 0, Sent: 1}, want: true},
		{name: "some accepted", err: &notAcceptedError{Accepted: 1, Sent: 2}},
		{name: "rejected", err: &apierr.APIError{StatusCode: 400}, want: true},
		{name: "server error", err: &apierr.APIError{StatusCode: 500}},
		{name: "connection reset", err: errors.New("connection reset by peer")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NotSent(tt.err))
		})
	}
}

func testFeedbackLog() []protos.FrontendLog {
	return []protos.FrontendLog{{
		FrontendLogEventID: "id",
		Entry: protos.FrontendLogEntry{
			FastFeedbackLog: &protos.FastFeedbackLog{Text: "hello"},
		},
	}}
}

func TestSendMakesOneAttempt(t *testing.T) {
	tests := []struct {
		name            string
		numProtoSuccess int64
		statusCode      int
		reset           bool
		wantErr         bool
		errContains     string
		wantNotSent     bool
	}{
		{name: "all accepted", numProtoSuccess: 1},
		{name: "none accepted", wantErr: true, errContains: "only 0 of 1 logs were accepted", wantNotSent: true},
		{name: "server error", statusCode: 500, wantErr: true, errContains: "server error"},
		{name: "SDK retryable error", statusCode: 503, wantErr: true, errContains: "server error"},
		{name: "rejected", statusCode: 403, wantErr: true, errContains: "server error", wantNotSent: true},
		{name: "connection reset", reset: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			var count atomic.Int32
			var body RequestBody
			if tt.reset {
				server.HandleRaw(http.MethodPost, "/telemetry-ext", func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					conn, _, err := w.(http.Hijacker).Hijack()
					if !assert.NoError(t, err) {
						return
					}
					// A zero linger makes Close send a TCP reset, which the SDK
					// retries unless retries are off.
					assert.NoError(t, conn.(*net.TCPConn).SetLinger(0))
					assert.NoError(t, conn.Close())
				})
			} else {
				server.Handle(http.MethodPost, "/telemetry-ext", func(req testserver.Request) any {
					count.Add(1)
					assert.NoError(t, json.Unmarshal(req.Body, &body))
					if tt.statusCode != 0 {
						return testserver.Response{
							StatusCode: tt.statusCode,
							Body:       apierr.APIError{StatusCode: tt.statusCode, Message: "server error"},
						}
					}
					return ResponseBody{NumProtoSuccess: tt.numProtoSuccess}
				})
			}

			err := Send(t.Context(), &config.Config{Host: server.URL, Token: "token"}, testFeedbackLog())
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.ErrorContains(t, err, tt.errContains)
				}
				assert.Equal(t, tt.wantNotSent, NotSent(err))
			}
			assert.Equal(t, int32(1), count.Load())
			if !tt.reset {
				assert.Equal(t, []string{`{"frontend_log_event_id":"id","entry":{"fast_feedback_log":{"can_be_contacted":false,"text":"hello"}}}`}, body.ProtoLogs)
			}
		})
	}
}

func TestSendFailingBeforeConnectingIsNotSent(t *testing.T) {
	// The client does not trust the certificate of this server. It starts
	// first so that it cannot take the port of the closed listener below.
	untrusted := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(untrusted.Close)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	refused := "http://" + listener.Addr().String()
	require.NoError(t, listener.Close())

	// The SDK retries its host metadata lookup for minutes when it fails.
	noMetadata := func(context.Context, string) (*config.HostMetadata, error) {
		return nil, nil
	}

	tests := []struct {
		name string
		host string
	}{
		{name: "connection refused", host: refused},
		{name: "untrusted certificate", host: untrusted.URL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Host: tt.host, Token: "token", HostMetadataResolver: noMetadata}
			err := Send(t.Context(), cfg, testFeedbackLog())
			require.Error(t, err)
			assert.True(t, NotSent(err), "error: %v", err)
		})
	}
}

func TestSendTimeoutBoundsTokenAcquisition(t *testing.T) {
	server := testserver.New(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	ts := auth.TokenSourceFn(func(context.Context) (*oauth2.Token, error) {
		<-release
		return nil, errors.New("token endpoint unavailable")
	})
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	errc := make(chan error, 1)
	go func() {
		errc <- Send(ctx, &config.Config{
			Host:        server.URL,
			Credentials: config.NewTokenSourceStrategy("blocking", ts),
		}, testFeedbackLog())
	}()
	select {
	case err := <-errc:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		// The token was never fetched, so nothing was sent.
		assert.True(t, NotSent(err))
	case <-time.After(uploadWatchdog):
		t.Fatal("Send did not return after its context deadline")
	}
}

// TestUploadUsesConfigSetByCommand checks that the upload authenticates with the config the command set, in place of the
// one cmdctx.ConfigUsed holds.
func TestUploadUsesConfigSetByCommand(t *testing.T) {
	bound := testserver.New(t)
	t.Cleanup(bound.Close)
	other := testserver.New(t)
	t.Cleanup(other.Close)
	var boundCalls, otherCalls int
	bound.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		boundCalls++
		return ResponseBody{NumProtoSuccess: 1}
	})
	other.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		otherCalls++
		return ResponseBody{NumProtoSuccess: 1}
	})

	ctx := WithNewLogger(t.Context())
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: other.URL, Token: "other"})
	SetUploadConfig(ctx, &config.Config{Host: bound.URL, Token: "bound"})

	cfg, ok := UploadConfig(ctx)
	require.True(t, ok)
	assert.Equal(t, bound.URL, cfg.Host)

	require.NoError(t, Upload(ctx, protos.ExecutionContext{}))
	assert.Equal(t, 1, boundCalls)
	assert.Zero(t, otherCalls)
}

// TestUploadConfigWithoutLogger checks that the upload slot is a no-op without a logger, as Log is.
func TestUploadConfigWithoutLogger(t *testing.T) {
	ctx := t.Context()
	SetUploadConfig(ctx, &config.Config{Host: "https://ws.test"})
	_, ok := UploadConfig(ctx)
	assert.False(t, ok)
}

// TestUploadSkippedDespiteConfigUsed checks that a skipped upload sends nothing, even when the run has a config from
// cmdctx.ConfigUsed and one set by the command before it skipped.
func TestUploadSkippedDespiteConfigUsed(t *testing.T) {
	bound := testserver.New(t)
	t.Cleanup(bound.Close)
	used := testserver.New(t)
	t.Cleanup(used.Close)
	var calls int
	for _, server := range []*testserver.Server{bound, used} {
		server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
			calls++
			return ResponseBody{NumProtoSuccess: 1}
		})
	}

	ctx := WithNewLogger(t.Context())
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: used.URL, Token: "used"})
	SetUploadConfig(ctx, &config.Config{Host: bound.URL, Token: "bound"})
	SkipUpload(ctx)

	_, ok := UploadConfig(ctx)
	assert.False(t, ok)
	require.NoError(t, Upload(ctx, protos.ExecutionContext{}))
	assert.Zero(t, calls)
}

// TestUploadSendsEachAcceptedEventOnce checks that an upload sends only the events logged since the earlier upload was
// accepted, so an event is not uploaded twice.
func TestUploadSendsEachAcceptedEventOnce(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var uploaded []int
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		var body RequestBody
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		uploaded = append(uploaded, len(body.ProtoLogs))
		return ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
	})

	ctx := WithNewLogger(t.Context())
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	require.NoError(t, Upload(ctx, protos.ExecutionContext{}))
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue2}})
	require.NoError(t, Upload(ctx, protos.ExecutionContext{}))

	assert.Equal(t, []int{1, 1}, uploaded)
	assert.Empty(t, BufferedEvents(ctx))
}

// TestUploadKeepsEventsNotAccepted checks that an upload the service does not accept leaves its events buffered, for a
// later upload to send.
func TestUploadKeepsEventsNotAccepted(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		return ResponseBody{NumProtoSuccess: 0}
	})

	ctx := WithNewLogger(t.Context())
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})
	Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	require.Error(t, Upload(ctx, protos.ExecutionContext{}))

	assert.Len(t, BufferedEvents(ctx), 1)
}
