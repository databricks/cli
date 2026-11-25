package dresources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBranchParent = "projects/p"
	testBranchName   = "projects/p/branches/dev"
	// The message the backend sends with a slow create; the SDK does not retry it.
	testBranchTimeoutMessage = "The create request timed out and may still be completing. Check whether the resource was created before retrying."
)

// branchCreateScript plays the given responses, in order, to the branch create and get requests.
// A response is a postgres.Branch (for a create: the finished operation that returns it) or a
// testserver.Response for an error.
type branchCreateScript struct {
	t       *testing.T
	creates []any
	gets    []any
	// onGet runs when a get request arrives, before its response is sent.
	onGet func()

	createCount int
	getCount    int
	operation   *postgres.Operation
}

func (s *branchCreateScript) client() *databricks.WorkspaceClient {
	server := testserver.New(s.t)
	server.Handle("POST", "/api/2.0/postgres/projects/{project_id}/branches", func(req testserver.Request) any {
		s.createCount++
		require.LessOrEqual(s.t, s.createCount, len(s.creates), "unexpected create request")
		resp := s.creates[s.createCount-1]
		if branch, ok := resp.(postgres.Branch); ok {
			body, err := json.Marshal(branch)
			require.NoError(s.t, err)
			s.operation = &postgres.Operation{Name: testBranchName + "/operations/op", Done: true, Response: body}
			return s.operation
		}
		return resp
	})
	// The SDK polls the operation even when the create response says it is done.
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}/operations/{operation_id}", func(req testserver.Request) any {
		require.NotNil(s.t, s.operation, "unexpected operation request")
		return s.operation
	})
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}", func(req testserver.Request) any {
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

func branchError(status int, code, message string) testserver.Response {
	return testserver.Response{StatusCode: status, Body: map[string]string{"error_code": code, "message": message}}
}

var (
	branchTimeout       = branchError(504, "DEADLINE_EXCEEDED", testBranchTimeoutMessage)
	branchNotFound      = branchError(404, "NOT_FOUND", "branch not found")
	branchAlreadyExists = branchError(409, "ALREADY_EXISTS", "branch with such id already exists")
)

func testBranch(state postgres.BranchStatusState, created time.Time) postgres.Branch {
	return postgres.Branch{
		Name:       testBranchName,
		BranchId:   "dev",
		Parent:     testBranchParent,
		CreateTime: sdktime.New(created),
		Status:     &postgres.BranchStatus{CurrentState: state},
	}
}

// createTestBranch runs DoCreate with fast retries and polling.
func createTestBranch(t *testing.T, s *branchCreateScript, ctx context.Context, replaceExisting bool) (string, *PostgresBranchRemote, error) {
	t.Helper()
	ctx = env.Set(ctx, retry.IntervalMsVariable, "1")
	setForTest(t, &branchReadyPollInterval, time.Millisecond)
	r := (&ResourcePostgresBranch{}).New(s.client())
	return r.DoCreate(ctx, &PostgresBranchState{BranchId: "dev", Parent: testBranchParent, ReplaceExisting: &replaceExisting})
}

func setForTest[T any](t *testing.T, v *T, value T) {
	old := *v
	*v = value
	t.Cleanup(func() { *v = old })
}

func TestPostgresBranchCreateWaitsForBranchCreatedDespiteTimeout(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{testBranch(postgres.BranchStatusStateInit, time.Now()), testBranch(postgres.BranchStatusStateReady, time.Now())},
	}

	id, remote, err := createTestBranch(t, s, t.Context(), false)

	require.NoError(t, err)
	assert.Equal(t, testBranchName, id)
	assert.Equal(t, postgres.BranchStatusStateReady, remote.Status.CurrentState)
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

func TestPostgresBranchCreateReturnsInitBranchAfterReadyTimeout(t *testing.T) {
	setForTest(t, &branchReadyTimeout, 0)
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{testBranch(postgres.BranchStatusStateInit, time.Now())},
	}

	id, remote, err := createTestBranch(t, s, t.Context(), false)

	// Returned so that state records it: the branch exists and is ours.
	require.NoError(t, err)
	assert.Equal(t, testBranchName, id)
	assert.Equal(t, postgres.BranchStatusStateInit, remote.Status.CurrentState)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresBranchCreateRepeatsCreateWhileBranchIsMissing(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout, testBranch(postgres.BranchStatusStateReady, time.Now())},
		gets:    []any{branchNotFound},
	}

	id, _, err := createTestBranch(t, s, t.Context(), false)

	require.NoError(t, err)
	assert.Equal(t, testBranchName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresBranchCreateUsesBranchWhenRepeatedCreateFindsIt(t *testing.T) {
	// The first request was still completing when the branch was read, and has landed since.
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout, branchAlreadyExists},
		gets:    []any{branchNotFound, testBranch(postgres.BranchStatusStateReady, time.Now())},
	}

	id, _, err := createTestBranch(t, s, t.Context(), false)

	require.NoError(t, err)
	assert.Equal(t, testBranchName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

func TestPostgresBranchCreateGivesUpWhenBranchStaysMissing(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout, branchTimeout, branchTimeout},
		gets:    []any{branchNotFound, branchNotFound, branchNotFound},
	}

	_, _, err := createTestBranch(t, s, t.Context(), false)

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	assert.Equal(t, 1+retry.MaxRetries, s.createCount)
	assert.Equal(t, 1+retry.MaxRetries, s.getCount)
}

func TestPostgresBranchCreateDoesNotUseBranchThatPredatesRequest(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{testBranch(postgres.BranchStatusStateReady, time.Now().Add(-time.Hour))},
	}

	_, _, err := createTestBranch(t, s, t.Context(), false)

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	require.ErrorContains(t, err, "branch "+testBranchName+" exists but was not created by this deployment")
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresBranchCreateFailsOnBranchThatWillNotBecomeReady(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{testBranch(postgres.BranchStatusStateDeleted, time.Now())},
	}

	_, _, err := createTestBranch(t, s, t.Context(), false)

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	require.ErrorContains(t, err, `exists in state "DELETED"`)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresBranchCreateReturnsCreateErrorWhenReadFails(t *testing.T) {
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{branchError(403, "PERMISSION_DENIED", "no access")},
	}

	_, _, err := createTestBranch(t, s, t.Context(), false)

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	assert.NotContains(t, err.Error(), "no access")
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresBranchCreateReturnsCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	s := &branchCreateScript{
		t:       t,
		creates: []any{branchTimeout},
		gets:    []any{branchNotFound},
		onGet:   cancel,
	}

	_, _, err := createTestBranch(t, s, ctx, false)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresBranchCreateLeavesReplaceToCallerRetry(t *testing.T) {
	s := &branchCreateScript{t: t, creates: []any{branchTimeout}}

	_, _, err := createTestBranch(t, s, t.Context(), true)

	// Marked safe to retry for DeploymentUnit.Create, without reading the branch.
	_, ok := errors.AsType[*RetrySafeError](err)
	assert.True(t, ok, "want a RetrySafeError, got %v", err)
	assert.Equal(t, 0, s.getCount)
}

func TestPostgresBranchCreateDoesNotRecoverFromNonTransientError(t *testing.T) {
	s := &branchCreateScript{t: t, creates: []any{branchError(400, "INVALID_PARAMETER_VALUE", "bad spec")}}

	_, _, err := createTestBranch(t, s, t.Context(), false)

	require.ErrorContains(t, err, "bad spec")
	_, ok := errors.AsType[*RetrySafeError](err)
	assert.False(t, ok)
	assert.Equal(t, 0, s.getCount)
}
