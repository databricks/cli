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

func TestListCurrentUserParentGroupIDsFollowsPages(t *testing.T) {
	var pages []string
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(req testserver.Request) any {
		token := req.URL.Query().Get("page_token")
		pages = append(pages, token)
		// The fixture bodies use the proto keys; a wrong key decodes to no groups.
		if token == "" {
			return map[string]any{
				"me_transitive_parent_groups": []map[string]any{{"group_id": "11"}, {"group_id": "12"}},
				"next_page_token":             "page2",
			}
		}
		return map[string]any{
			"me_transitive_parent_groups": []map[string]any{{"group_id": "13"}},
		}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	ids, err := listCurrentUserParentGroupIDs(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, []string{"11", "12", "13"}, ids)
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

// A parent ID that is missing from the account group list is skipped, not reported as
// an error: the other memberships are still known.
func TestCurrentUserGroupsSkipsParentIDMissingFromAccountList(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	server.AddGroups("test-token", []string{"testgroup"}, nil)
	server.JoinGroup("test-token", "missing-group")

	groups, err := CurrentUserGroups(t.Context(), w)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "testgroup", groups[0].Display)
	assert.Equal(t, "testgroup", groups[0].Display)
}

// The account group list is paged; every page is read, and names are matched by ID
// across pages.
func TestCurrentUserGroupsReadsEveryPageOfGroupList(t *testing.T) {
	var pages []string
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		return map[string]any{"me_transitive_parent_groups": []map[string]any{{"group_id": "11"}, {"group_id": "12"}, {"group_id": "13"}}}
	})
	server.Handle("GET", "/api/2.0/identity/groups", func(req testserver.Request) any {
		token := req.URL.Query().Get("page_token")
		pages = append(pages, token)
		if token == "" {
			return map[string]any{
				"groups":          []map[string]any{{"group_id": "11", "group_name": "b"}, {"group_id": "99", "group_name": "unrelated"}},
				"next_page_token": "page2",
			}
		}
		return map[string]any{"groups": []map[string]any{{"group_id": "12", "group_name": "a"}, {"group_id": "13", "group_name": "c"}}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := CurrentUserGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, []iam.ComplexValue{
		{Value: "12", Display: "a"},
		{Value: "11", Display: "b"},
		{Value: "13", Display: "c"},
	}, groups)
	assert.Equal(t, []string{"", "page2"}, pages)
}

// A failed account group list fails the lookup, so no partial list is returned as the
// complete membership.
func TestCurrentUserGroupsFailsWhenGroupListFails(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		return map[string]any{"me_transitive_parent_groups": []map[string]any{{"group_id": "11"}}}
	})
	server.Handle("GET", "/api/2.0/identity/groups", func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "denied"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := CurrentUserGroups(t.Context(), w)
	require.ErrorContains(t, err, "denied")
	assert.Nil(t, groups)
}
