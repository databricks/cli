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

const testProjectName = "projects/p"

// projectCreateScript plays the given responses, in order, to the project create and get requests.
// A response is a postgres.Project (for a create: the finished operation that returns it) or a
// testserver.Response for an error.
type projectCreateScript struct {
	t       *testing.T
	creates []any
	gets    []any
	// branchGets are the responses to reads of the default branch.
	branchGets []any

	createCount    int
	getCount       int
	branchGetCount int
	operation      *postgres.Operation
}

func (s *projectCreateScript) client() *databricks.WorkspaceClient {
	server := testserver.New(s.t)
	server.Handle("POST", "/api/2.0/postgres/projects", func(req testserver.Request) any {
		s.createCount++
		require.LessOrEqual(s.t, s.createCount, len(s.creates), "unexpected create request")
		resp := s.creates[s.createCount-1]
		if project, ok := resp.(postgres.Project); ok {
			body, err := json.Marshal(project)
			require.NoError(s.t, err)
			s.operation = &postgres.Operation{Name: testProjectName + "/operations/op", Done: true, Response: body}
			return s.operation
		}
		return resp
	})
	// The SDK polls the operation even when the create response says it is done.
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/operations/{operation_id}", func(req testserver.Request) any {
		require.NotNil(s.t, s.operation, "unexpected operation request")
		return s.operation
	})
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}", func(req testserver.Request) any {
		s.getCount++
		require.LessOrEqual(s.t, s.getCount, len(s.gets), "unexpected get request")
		return s.gets[s.getCount-1]
	})
	server.Handle("GET", "/api/2.0/postgres/projects/{project_id}/branches/{branch_id}", func(req testserver.Request) any {
		s.branchGetCount++
		require.LessOrEqual(s.t, s.branchGetCount, len(s.branchGets), "unexpected branch get request")
		return s.branchGets[s.branchGetCount-1]
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(s.t, err)
	return client
}

const testDefaultBranchName = testProjectName + "/branches/production"

// testProjectWithDefaultBranch is a project whose status names its default branch.
func testProjectWithDefaultBranch(created time.Time) postgres.Project {
	project := testProject(created)
	project.Status = &postgres.ProjectStatus{DefaultBranch: testDefaultBranchName}
	return project
}

func testDefaultBranch(state postgres.BranchStatusState) postgres.Branch {
	return postgres.Branch{Name: testDefaultBranchName, Status: &postgres.BranchStatus{CurrentState: state}}
}

var (
	projectNotFound = branchError(404, "NOT_FOUND", "project not found")
	// The real API answers a duplicate project with 400, not 409 ALREADY_EXISTS.
	projectAlreadyExists = branchError(400, "BAD_REQUEST", "A project with this resource name already exists in the workspace.")
)

func testProject(created time.Time) postgres.Project {
	return postgres.Project{Name: testProjectName, ProjectId: "p", CreateTime: sdktime.New(created)}
}

// createTestProject runs DoCreate with fast retries.
func createTestProject(t *testing.T, s *projectCreateScript, projectID string) (string, *PostgresProjectRemote, error) {
	t.Helper()
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	setForTest(t, &branchReadyPollInterval, time.Millisecond)
	r := (&ResourcePostgresProject{}).New(s.client())
	return r.DoCreate(ctx, &PostgresProjectState{ProjectId: projectID})
}

func TestPostgresProjectCreateUsesProjectCreatedDespiteTimeout(t *testing.T) {
	s := &projectCreateScript{t: t, creates: []any{branchTimeout}, gets: []any{testProject(time.Now())}}

	id, remote, err := createTestProject(t, s, "p")

	require.NoError(t, err)
	assert.Equal(t, testProjectName, id)
	assert.Equal(t, "p", remote.ProjectId)
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresProjectCreateRepeatsCreateWhileProjectIsMissing(t *testing.T) {
	s := &projectCreateScript{t: t, creates: []any{branchTimeout, testProject(time.Now())}, gets: []any{projectNotFound}}

	id, _, err := createTestProject(t, s, "p")

	require.NoError(t, err)
	assert.Equal(t, testProjectName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 1, s.getCount)
}

func TestPostgresProjectCreateUsesProjectWhenRepeatedCreateFindsIt(t *testing.T) {
	// The first request was still completing when the project was read, and has landed since.
	s := &projectCreateScript{
		t:       t,
		creates: []any{branchTimeout, projectAlreadyExists},
		gets:    []any{projectNotFound, testProject(time.Now())},
	}

	id, _, err := createTestProject(t, s, "p")

	require.NoError(t, err)
	assert.Equal(t, testProjectName, id)
	assert.Equal(t, 2, s.createCount)
	assert.Equal(t, 2, s.getCount)
}

func TestPostgresProjectCreateGivesUpWhenProjectStaysMissing(t *testing.T) {
	s := &projectCreateScript{
		t:       t,
		creates: []any{branchTimeout, branchTimeout, branchTimeout},
		gets:    []any{projectNotFound, projectNotFound, projectNotFound},
	}

	_, _, err := createTestProject(t, s, "p")

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	assert.Equal(t, 1+retry.MaxRetries, s.createCount)
	assert.Equal(t, 1+retry.MaxRetries, s.getCount)
}

func TestPostgresProjectCreateDoesNotUseProjectThatPredatesRequest(t *testing.T) {
	s := &projectCreateScript{t: t, creates: []any{branchTimeout}, gets: []any{testProject(time.Now().Add(-time.Hour))}}

	_, _, err := createTestProject(t, s, "p")

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	require.ErrorContains(t, err, "project "+testProjectName+" exists but was not created by this deployment")
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresProjectCreateReturnsCreateErrorWhenReadFails(t *testing.T) {
	s := &projectCreateScript{t: t, creates: []any{branchTimeout}, gets: []any{branchError(403, "PERMISSION_DENIED", "no access")}}

	_, _, err := createTestProject(t, s, "p")

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	assert.NotContains(t, err.Error(), "no access")
	assert.Equal(t, 1, s.createCount)
}

func TestPostgresProjectCreateDoesNotRecoverFromNonTransientError(t *testing.T) {
	s := &projectCreateScript{t: t, creates: []any{projectAlreadyExists}}

	_, _, err := createTestProject(t, s, "p")

	require.ErrorContains(t, err, "already exists")
	_, ok := errors.AsType[*RetrySafeError](err)
	assert.False(t, ok)
	assert.Equal(t, 0, s.getCount)
}

func TestPostgresProjectCreateWithoutProjectIdIsNotRepeated(t *testing.T) {
	// Without a project_id there is no name to look up.
	s := &projectCreateScript{t: t, creates: []any{branchTimeout}}

	_, _, err := createTestProject(t, s, "")

	require.ErrorContains(t, err, testBranchTimeoutMessage)
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 0, s.getCount)
}

func TestPostgresProjectCreateWaitsForDefaultBranchOfRecoveredProject(t *testing.T) {
	s := &projectCreateScript{
		t:          t,
		creates:    []any{branchTimeout},
		gets:       []any{testProjectWithDefaultBranch(time.Now())},
		branchGets: []any{testDefaultBranch(postgres.BranchStatusStateInit), testDefaultBranch(postgres.BranchStatusStateReady)},
	}

	id, _, err := createTestProject(t, s, "p")

	require.NoError(t, err)
	assert.Equal(t, testProjectName, id)
	assert.Equal(t, 1, s.createCount)
	assert.Equal(t, 2, s.branchGetCount)
}

func TestPostgresProjectCreateReturnsRecoveredProjectWhenDefaultBranchReadFails(t *testing.T) {
	// The project is ours: return it so that state records it, even though the wait ended early.
	s := &projectCreateScript{
		t:          t,
		creates:    []any{branchTimeout},
		gets:       []any{testProjectWithDefaultBranch(time.Now())},
		branchGets: []any{branchError(403, "PERMISSION_DENIED", "no access")},
	}

	id, _, err := createTestProject(t, s, "p")

	require.NoError(t, err)
	assert.Equal(t, testProjectName, id)
	assert.Equal(t, 1, s.branchGetCount)
}
