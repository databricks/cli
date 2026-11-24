package dresources

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFeatureName = "main.default.feat"

// featureCreateScript plays the given responses, in order, to the feature create and get
// requests. A response is an ml.Feature or a testserver.Response for an error.
type featureCreateScript struct {
	t       *testing.T
	creates []any
	gets    []any
	// onGet runs when a get request arrives, before its response is sent.
	onGet func()

	createCount int
	getCount    int
}

func (s *featureCreateScript) client() *databricks.WorkspaceClient {
	server := testserver.New(s.t)
	server.Handle("POST", "/api/2.0/feature-engineering/features", func(req testserver.Request) any {
		s.createCount++
		require.LessOrEqual(s.t, s.createCount, len(s.creates), "unexpected create request")
		return s.creates[s.createCount-1]
	})
	server.Handle("GET", "/api/2.0/feature-engineering/features/{full_name}", func(req testserver.Request) any {
		s.getCount++
		require.LessOrEqual(s.t, s.getCount, len(s.gets), "unexpected get request")
		if s.onGet != nil {
			s.onGet()
		}
		return s.gets[s.getCount-1]
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(s.t, err)
	return client
}

func featureError(status int, code, message string) testserver.Response {
	return testserver.Response{StatusCode: status, Body: map[string]string{"error_code": code, "message": message}}
}

var (
	// Real 504s from the feature-engineering gateway carry an empty message, which the SDK does
	// not retry.
	featureTimeout       = featureError(504, "DEADLINE_EXCEEDED", "")
	featureNotFound      = featureError(404, "NOT_FOUND", "Feature 'main.default.feat' not found.")
	featureAlreadyExists = featureError(409, "ALREADY_EXISTS", "Feature 'main.default.feat' already exists.")
	featureDenied        = featureError(403, "PERMISSION_DENIED", "denied")
	featureInvalid       = featureError(400, "INVALID_PARAMETER_VALUE", "invalid")
)

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	require.True(t, ok, "not an API error: %v", err)
	assert.Equal(t, status, apiErr.StatusCode)
}

func testFeature(created time.Time) ml.Feature {
	return ml.Feature{FullName: testFeatureName, Description: "d", CreatedAt: sdktime.New(created)}
}

// createTestFeature runs DoCreate with fast retries.
func createTestFeature(t *testing.T, s *featureCreateScript, ctx context.Context) (string, *ml.Feature, error) {
	t.Helper()
	ctx = env.Set(ctx, retry.IntervalMsVariable, "1")
	r := (&ResourceFeature{}).New(s.client())
	return r.DoCreate(ctx, &ml.Feature{FullName: testFeatureName, Description: "d"})
}

func TestCreateRecoveryUsesResourceCreatedDespiteTimeout(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{testFeature(time.Now())}}
	id, remote, err := createTestFeature(t, s, t.Context())
	require.NoError(t, err)
	assert.Equal(t, testFeatureName, id)
	assert.Equal(t, testFeatureName, remote.FullName)
	assert.Equal(t, 1, s.createCount)
}

func TestCreateRecoveryRejectsResourceThatPredatesRequest(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{testFeature(time.Now().Add(-time.Hour))}}
	_, _, err := createTestFeature(t, s, t.Context())
	// The 504 has an empty message, so nothing follows the explanation; the bundle error output
	// appends the status code and details from the wrapped error.
	require.EqualError(t, err, "feature main.default.feat exists but was not created by this deployment")
	requireStatus(t, err, 504)
}

func TestCreateRecoveryAllowsClockSkew(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{testFeature(time.Now().Add(-30 * time.Second))}}
	_, _, err := createTestFeature(t, s, t.Context())
	require.NoError(t, err)
}

func TestCreateRecoveryKeepsCreateErrorMessage(t *testing.T) {
	timeout := featureError(504, "DEADLINE_EXCEEDED", "The request timed out.")
	s := &featureCreateScript{t: t, creates: []any{timeout}, gets: []any{testFeature(time.Now().Add(-time.Hour))}}
	_, _, err := createTestFeature(t, s, t.Context())
	require.EqualError(t, err, "feature main.default.feat exists but was not created by this deployment: The request timed out.")
	requireStatus(t, err, 504)
}

func TestCreateRecoveryRejectsResourceWithoutCreateTime(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{ml.Feature{FullName: testFeatureName}}}
	_, _, err := createTestFeature(t, s, t.Context())
	require.EqualError(t, err, "feature main.default.feat exists but has no create time, so it is not known to be created by this deployment")
	requireStatus(t, err, 504)
}

func TestCreateRecoveryRepeatsCreateWhileMissing(t *testing.T) {
	s := &featureCreateScript{
		t:       t,
		creates: []any{featureTimeout, testFeature(time.Now())},
		gets:    []any{featureNotFound},
	}
	id, _, err := createTestFeature(t, s, t.Context())
	require.NoError(t, err)
	assert.Equal(t, testFeatureName, id)
	assert.Equal(t, 2, s.createCount)
}

func TestCreateRecoveryLooksAgainAfterAlreadyExists(t *testing.T) {
	s := &featureCreateScript{
		t:       t,
		creates: []any{featureTimeout, featureAlreadyExists},
		gets:    []any{featureNotFound, testFeature(time.Now())},
	}
	id, _, err := createTestFeature(t, s, t.Context())
	require.NoError(t, err)
	assert.Equal(t, testFeatureName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

func TestCreateRecoveryGivesUpAfterMaxRetries(t *testing.T) {
	creates := []any{featureTimeout}
	var gets []any
	for range retry.MaxRetries {
		creates = append(creates, featureTimeout)
		gets = append(gets, featureNotFound)
	}
	gets = append(gets, featureNotFound)
	s := &featureCreateScript{t: t, creates: creates, gets: gets}
	_, _, err := createTestFeature(t, s, t.Context())
	requireStatus(t, err, 504)
	assert.Equal(t, 1+retry.MaxRetries, s.createCount)
	assert.Equal(t, 1+retry.MaxRetries, s.getCount)
}

func TestCreateRecoveryReturnsCreateErrorWhenReadFails(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{featureDenied}}
	_, _, err := createTestFeature(t, s, t.Context())
	requireStatus(t, err, 504)
	assert.Equal(t, 1, s.createCount)
}

func TestCreateRecoveryReturnsNonTransientRepeatError(t *testing.T) {
	s := &featureCreateScript{
		t:       t,
		creates: []any{featureTimeout, featureInvalid},
		gets:    []any{featureNotFound},
	}
	_, _, err := createTestFeature(t, s, t.Context())
	require.ErrorContains(t, err, "invalid")
}

func TestCreateRecoveryIgnoresNonTransientError(t *testing.T) {
	s := &featureCreateScript{t: t, creates: []any{featureAlreadyExists}}
	_, _, err := createTestFeature(t, s, t.Context())
	require.ErrorContains(t, err, "already exists")
	assert.Equal(t, 0, s.getCount)
}

func TestCreateRecoveryStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	s := &featureCreateScript{t: t, creates: []any{featureTimeout}, gets: []any{featureNotFound}, onGet: cancel}
	_, _, err := createTestFeature(t, s, ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, s.createCount)
}

func TestIsUncertainCreateError(t *testing.T) {
	ctx := t.Context()
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"504", &apierr.APIError{StatusCode: 504, ErrorCode: "DEADLINE_EXCEEDED", Message: "Storage timeout"}, true},
		{"499", &apierr.APIError{StatusCode: 499, ErrorCode: "CANCELLED"}, true},
		{"stream reset", fmt.Errorf("response body: %w", errors.New("stream error: stream ID 5; INTERNAL_ERROR; received from peer")), true},
		{"499 with mapped code", &apierr.APIError{StatusCode: 499, ErrorCode: "UNKNOWN"}, true},
		{"API error quoting a stream error", &apierr.APIError{StatusCode: 400, ErrorCode: "INVALID_PARAMETER_VALUE", Message: "upstream: stream error: stream ID 1; PROTOCOL_ERROR"}, false},
		{"400", &apierr.APIError{StatusCode: 400, ErrorCode: "INVALID_PARAMETER_VALUE"}, false},
		{"409", &apierr.APIError{StatusCode: 409, ErrorCode: "ALREADY_EXISTS"}, false},
		{"other", errors.New("boom"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isUncertainCreateError(ctx, tc.err))
		})
	}
}
