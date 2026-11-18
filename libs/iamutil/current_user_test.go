package iamutil

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIdentityMeWorkspace returns a workspace client whose identity/me is answered by handler.
func newIdentityMeWorkspace(t *testing.T, handler func(req testserver.Request) testserver.Response) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, func(req testserver.Request) any {
		return handler(req)
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

func TestMeToUser(t *testing.T) {
	tests := []struct {
		name    string
		me      identityMe
		want    *iam.User
		wantErr bool
	}{
		{
			name: "user",
			me: identityMe{User: &iamv2.User{
				UserId:            "1000012345",
				Username:          "alice@example.com",
				FullName:          iamv2.UserFullName{GivenName: "Alice", FamilyName: "Smith"},
				AccountUserStatus: iamv2.StateActive,
				ExternalId:        "ext-1",
			}},
			want: &iam.User{
				Id:          "1000012345",
				UserName:    "alice@example.com",
				Name:        &iam.Name{GivenName: "Alice", FamilyName: "Smith"},
				DisplayName: "Alice Smith",
				Active:      true,
				ExternalId:  "ext-1",
				Emails:      []iam.ComplexValue{{Value: "alice@example.com", Primary: true}},
			},
		},
		{
			name: "user without full name and inactive",
			me: identityMe{User: &iamv2.User{
				UserId:            "1000012345",
				Username:          "alice@example.com",
				AccountUserStatus: iamv2.StateInactive,
			}},
			want: &iam.User{
				Id:       "1000012345",
				UserName: "alice@example.com",
				Emails:   []iam.ComplexValue{{Value: "alice@example.com", Primary: true}},
			},
		},
		{
			name: "service principal reports application id as user name",
			me: identityMe{ServicePrincipal: &iamv2.ServicePrincipal{
				ServicePrincipalId: "2000022222",
				ApplicationId:      "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee",
				DisplayName:        "deploy-sp",
				AccountSpStatus:    iamv2.StateActive,
			}},
			want: &iam.User{
				Id:          "2000022222",
				UserName:    "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee",
				DisplayName: "deploy-sp",
				Active:      true,
			},
		},
		{
			name: "assumed group",
			me:   identityMe{Group: &iamv2.Group{GroupId: "3000033333", GroupName: "data-engineers"}},
			want: &iam.User{
				Id:          "3000033333",
				UserName:    "data-engineers",
				DisplayName: "data-engineers",
			},
		},
		{
			name:    "no principal",
			me:      identityMe{},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := meToUser(&tc.me)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetCurrentUserRetriesOn500(t *testing.T) {
	var calls atomic.Int32
	w := newIdentityMeWorkspace(t, func(testserver.Request) testserver.Response {
		if calls.Add(1) == 1 {
			return testserver.Response{StatusCode: http.StatusInternalServerError, Body: map[string]any{"error_code": "TEMPORARILY_UNAVAILABLE"}}
		}
		return testserver.Response{Body: map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user"}}}
	})

	user, err := GetCurrentUser(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", user.UserName)
	assert.Equal(t, int32(2), calls.Load())
}

func TestGetCurrentUserDoesNotRetryOn403(t *testing.T) {
	var calls atomic.Int32
	w := newIdentityMeWorkspace(t, func(testserver.Request) testserver.Response {
		calls.Add(1)
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})

	_, err := GetCurrentUser(t.Context(), w)
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	// A non-500 error must fail immediately, without a retry.
	assert.Equal(t, int32(1), calls.Load())
}

func TestListCurrentUserParentGroupsFollowsPages(t *testing.T) {
	var pages []string
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(req testserver.Request) any {
		token := req.URL.Query().Get("page_token")
		pages = append(pages, token)
		// The fixture bodies use the proto keys; a wrong key decodes to no groups.
		if token == "" {
			return map[string]any{
				"me_transitive_parent_groups": []map[string]any{{"group_id": "11", "group_name": "eng"}, {"group_id": "12", "group_name": "ops"}},
				"next_page_token":             "page2",
			}
		}
		return map[string]any{
			"me_transitive_parent_groups": []map[string]any{{"group_id": "13", "group_name": "users"}},
		}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := listCurrentUserParentGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, []parentGroup{
		{GroupId: "11", GroupName: "eng"},
		{GroupId: "12", GroupName: "ops"},
		{GroupId: "13", GroupName: "users"},
	}, groups)
	assert.Equal(t, []string{"", "page2"}, pages)
}

func TestCurrentUserGroupsResolvesNamesSorted(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	server.AddGroups("test-token", []string{"zeta", "alpha"}, nil)

	groups, err := CurrentUserGroups(t.Context(), w)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "alpha", groups[0].Display)
	assert.Equal(t, "zeta", groups[1].Display)
	assert.NotEmpty(t, groups[0].Value)
}

// Workspace-local groups such as admins and users are named by the parent-groups
// response itself, so they resolve without any account group read.
func TestCurrentUserGroupsIncludesWorkspaceLocalGroups(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		return map[string]any{"me_transitive_parent_groups": []map[string]any{
			{"group_id": "86546749069356", "group_name": "users"},
			{"group_id": "79621542766305", "group_name": "account users"},
			{"group_id": "90000000000001", "group_name": "admins"},
		}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := CurrentUserGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, []iam.ComplexValue{
		{Value: "79621542766305", Display: "account users"},
		{Value: "90000000000001", Display: "admins"},
		{Value: "86546749069356", Display: "users"},
	}, groups)
}

// A failed parent-groups read fails the lookup, so no partial list is returned as the
// complete membership.
func TestCurrentUserGroupsFailsWhenParentGroupsFail(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := CurrentUserGroups(t.Context(), w)
	require.ErrorContains(t, err, "denied")
	assert.Nil(t, groups)
}

func TestGetUserPrincipalRejectsAssumedGroup(t *testing.T) {
	w := newIdentityMeWorkspace(t, func(testserver.Request) testserver.Response {
		return testserver.Response{Body: map[string]any{"group": map[string]any{"group_id": "3000033333", "group_name": "data-engineers"}}}
	})

	user, err := GetUserPrincipal(t.Context(), w)
	require.ErrorIs(t, err, ErrGroupPrincipal)
	assert.Nil(t, user)

	// Commands that only identify the caller still get the group.
	group, err := GetCurrentUser(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "data-engineers", group.UserName)
}

func TestGetUserPrincipalReturnsUser(t *testing.T) {
	w := newIdentityMeWorkspace(t, func(testserver.Request) testserver.Response {
		return testserver.Response{Body: map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user"}}}
	})

	user, err := GetUserPrincipal(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", user.UserName)
}

// GetCurrentUserOnce makes a single attempt, so a probe fails fast on a transient 500.
func TestGetCurrentUserOnceDoesNotRetryOn500(t *testing.T) {
	var calls atomic.Int32
	w := newIdentityMeWorkspace(t, func(testserver.Request) testserver.Response {
		calls.Add(1)
		return testserver.Response{StatusCode: http.StatusInternalServerError, Body: map[string]any{"error_code": "TEMPORARILY_UNAVAILABLE"}}
	})

	_, err := GetCurrentUserOnce(t.Context(), w)
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load())
}
