package dresources

import (
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
	testRoleParent = "projects/p/branches/main"
	testRoleName   = testRoleParent + "/roles/owner"
)

// roleCreateScript plays the given responses, in order, to the role create and get requests.
// A response is a postgres.Role (for a create: the finished operation that returns it) or a
// testserver.Response for an error.
type roleCreateScript struct {
	t       *testing.T
	creates []any
	gets    []any

	createCount int
	getCount    int
	operation   *postgres.Operation
}

func (s *roleCreateScript) client() *databricks.WorkspaceClient {
	server := testserver.New(s.t)
	server.Handle("POST", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}/roles", func(req testserver.Request) any {
		s.createCount++
		require.LessOrEqual(s.t, s.createCount, len(s.creates), "unexpected create request")
		resp := s.creates[s.createCount-1]
		if role, ok := resp.(postgres.Role); ok {
			body, err := json.Marshal(role)
			require.NoError(s.t, err)
			s.operation = &postgres.Operation{Name: testRoleName + "/operations/op", Done: true, Response: body}
			return s.operation
		}
		return resp
	})
	// The SDK polls the operation even when the create response says it is done.
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}/roles/{role_id}/operations/{operation_id}", func(req testserver.Request) any {
		require.NotNil(s.t, s.operation, "unexpected operation request")
		return s.operation
	})
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}/roles/{role_id}", func(req testserver.Request) any {
		s.getCount++
		require.LessOrEqual(s.t, s.getCount, len(s.gets), "unexpected get request")
		return s.gets[s.getCount-1]
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(s.t, err)
	return client
}

var (
	// What the cloud answers to a role create now and then, right after the branch is created.
	roleInternalError = branchError(500, "INTERNAL_ERROR", "The request could not be processed. Please try again later and contact Databricks if the problem persists.")
	roleNotFound      = branchError(404, "NOT_FOUND", "role not found")
	// The real API answers a duplicate role with 400, not 409 ALREADY_EXISTS.
	roleAlreadyExists = branchError(400, "BAD_REQUEST", "role with that name already exists")
)

func testRole(created time.Time) postgres.Role {
	return postgres.Role{Name: testRoleName, RoleId: "owner", Parent: testRoleParent, CreateTime: sdktime.New(created)}
}

// createTestRole runs DoCreate with fast retries.
func createTestRole(t *testing.T, s *roleCreateScript, replaceExisting bool) (string, *PostgresRoleRemote, error) {
	t.Helper()
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	r := (&ResourcePostgresRole{}).New(s.client())
	return r.DoCreate(ctx, &PostgresRoleState{RoleId: "owner", Parent: testRoleParent, ReplaceExisting: &replaceExisting})
}

func TestPostgresRoleCreateUsesRoleCreatedDespiteError(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{roleInternalError}, gets: []any{testRole(time.Now())}}

	id, remote, err := createTestRole(t, s, false)

	require.NoError(t, err)
	assert.Equal(t, testRoleName, id)
	assert.Equal(t, "owner", remote.RoleId)
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresRoleCreateRepeatsCreateWhileRoleIsMissing(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{roleInternalError, testRole(time.Now())}, gets: []any{roleNotFound}}

	id, _, err := createTestRole(t, s, false)

	require.NoError(t, err)
	assert.Equal(t, testRoleName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresRoleCreateUsesRoleWhenRepeatedCreateFindsIt(t *testing.T) {
	// The first request was still completing when the role was read, and has landed since.
	s := &roleCreateScript{
		t:       t,
		creates: []any{roleInternalError, roleAlreadyExists},
		gets:    []any{roleNotFound, testRole(time.Now())},
	}

	id, _, err := createTestRole(t, s, false)

	require.NoError(t, err)
	assert.Equal(t, testRoleName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

func TestPostgresRoleCreateGivesUpWhenRoleStaysMissing(t *testing.T) {
	creates := []any{roleInternalError}
	var gets []any
	for range retry.MaxRetries {
		creates = append(creates, roleInternalError)
		gets = append(gets, roleNotFound)
	}
	gets = append(gets, roleNotFound)
	s := &roleCreateScript{t: t, creates: creates, gets: gets}

	_, _, err := createTestRole(t, s, false)

	require.ErrorContains(t, err, "The request could not be processed")
	assert.Equal(t, retry.MaxRetries+1, s.createCount)
	assert.Equal(t, retry.MaxRetries+1, s.getCount)
}

func TestPostgresRoleCreateDoesNotTakeOverOlderRole(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{roleInternalError}, gets: []any{testRole(time.Now().Add(-time.Hour))}}

	_, _, err := createTestRole(t, s, false)

	require.ErrorContains(t, err, "The request could not be processed")
	require.ErrorContains(t, err, "role "+testRoleName+" exists but was not created by this deployment")
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresRoleCreateFailsWhenRoleCannotBeRead(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{roleInternalError}, gets: []any{branchError(403, "PERMISSION_DENIED", "no access")}}

	_, _, err := createTestRole(t, s, false)

	require.ErrorContains(t, err, "The request could not be processed")
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresRoleCreateLeavesReplaceToCallerRetry(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{roleInternalError}}

	_, _, err := createTestRole(t, s, true)

	// Marked safe to retry for DeploymentUnit.Create, without reading the role.
	_, ok := errors.AsType[*RetrySafeError](err)
	assert.True(t, ok, "want a RetrySafeError, got %v", err)
	assert.Equal(t, 0, s.getCount)
}

func TestPostgresRoleCreateDoesNotRecoverFromNonTransientError(t *testing.T) {
	s := &roleCreateScript{t: t, creates: []any{branchError(400, "INVALID_PARAMETER_VALUE", "bad spec")}}

	_, _, err := createTestRole(t, s, false)

	require.ErrorContains(t, err, "bad spec")
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 0, s.getCount)
}
