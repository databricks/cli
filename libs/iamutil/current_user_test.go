package iamutil

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
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

// currentUserGroups returns the parent groups in the shape GetUserPrincipalAndGroups gives.
func currentUserGroups(ctx context.Context, w *databricks.WorkspaceClient) ([]iam.ComplexValue, error) {
	parents, err := listCurrentUserParentGroups(ctx, w)
	if err != nil {
		return nil, err
	}
	return groupValues(parents), nil
}

func TestCurrentUserGroupsResolvesNamesSorted(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	server.AddGroups("test-token", []string{"zeta", "alpha"}, nil)

	groups, err := currentUserGroups(t.Context(), w)
	require.NoError(t, err)
	require.Len(t, groups, 3)
	assert.Equal(t, "alpha", groups[0].Display)
	assert.Equal(t, "users", groups[1].Display)
	assert.Equal(t, "zeta", groups[2].Display)
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

	groups, err := currentUserGroups(t.Context(), w)
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

	groups, err := currentUserGroups(t.Context(), w)
	require.ErrorContains(t, err, "denied")
	assert.Nil(t, groups)
}

// The groups read is advisory and makes one attempt: a 500 is not retried, so a failing
// endpoint cannot stall a command for the 500 retry window. The groups become unknown.
func TestCurrentUserGroupsReadsOnceOn500(t *testing.T) {
	var calls atomic.Int32
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		calls.Add(1)
		return testserver.Response{StatusCode: http.StatusInternalServerError, Body: map[string]any{"error_code": "TEMPORARILY_UNAVAILABLE"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	groups, err := currentUserGroups(t.Context(), w)
	require.Error(t, err)
	assert.Nil(t, groups)
	assert.Equal(t, int32(1), calls.Load())
}

// Every request of the current-user read carries the workspace routing header when the
// profile sets a workspace ID, on identity/me, the parent groups and the SCIM fallback.
func TestCurrentUserRequestsCarryWorkspaceIDHeader(t *testing.T) {
	const wsID = "4242"
	newServer := func(me, parents, scim func(testserver.Request) any) *databricks.WorkspaceClient {
		server := testserver.New(t)
		t.Cleanup(server.Close)
		server.Handle("GET", identityMePath, me)
		server.Handle("GET", identityMeTransitiveParentGroupsPath, parents)
		server.Handle("GET", scimMePath, scim)
		w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token", WorkspaceID: wsID})
		require.NoError(t, err)
		return w
	}
	okUser := func(testserver.Request) any {
		return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user", "account_user_status": "ACTIVE"}}
	}
	emptyParents := func(testserver.Request) any { return map[string]any{"me_transitive_parent_groups": []map[string]any{}} }
	notEnabled := func(testserver.Request) any {
		return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND")
	}

	t.Run("identity/me", func(t *testing.T) {
		var seen []string
		record := func(h func(testserver.Request) any) func(testserver.Request) any {
			return func(req testserver.Request) any {
				seen = append(seen, req.Headers.Get(auth.WorkspaceIDHeader))
				return h(req)
			}
		}
		w := newServer(record(okUser), record(emptyParents), func(testserver.Request) any { return scimUserBody })
		_, err := GetUserPrincipalAndGroups(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, []string{wsID, wsID}, seen)
	})

	t.Run("SCIM fallback", func(t *testing.T) {
		var seen []string
		w := newServer(notEnabled, notEnabled, func(req testserver.Request) any {
			seen = append(seen, req.Headers.Get(auth.WorkspaceIDHeader))
			return scimUserBody
		})
		_, err := GetUserPrincipalAndGroups(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, []string{wsID}, seen)
	})

	t.Run("SCIM groups fallback", func(t *testing.T) {
		var seen []string
		w := newServer(okUser, notEnabled, func(req testserver.Request) any {
			seen = append(seen, req.Headers.Get(auth.WorkspaceIDHeader))
			return scimUserBody
		})
		_, err := GetUserPrincipalAndGroups(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, []string{wsID}, seen)
	})
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

// groupPairs returns "value display" per group, so tests compare what the fallback passes on
// and not the SDK's decoding details.
func groupPairs(groups []iam.ComplexValue) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g.Value+" "+g.Display)
	}
	return out
}

// scimUserBody is the SCIM /Me response of a workspace without identity/me. Its groups are
// unsorted, so a test can see that the fallback passes them on as SCIM returns them.
var scimUserBody = map[string]any{
	"id":       "77",
	"userName": "scim-user@databricks.com",
	"groups":   []map[string]any{{"value": "2", "display": "zeta"}, {"value": "1", "display": "alpha"}},
}

// notEnabledResponse is what identity/me answers in a workspace that does not serve it yet.
func notEnabledResponse(status int, errorCode string) testserver.Response {
	return testserver.Response{StatusCode: status, Body: map[string]any{"error_code": errorCode, "message": "Me API is not enabled."}}
}

// newWorkspaceWithSCIM returns a workspace whose identity/me, parent-groups and SCIM /Me are
// answered by the given handlers. The handlers count their own calls where a test needs it.
func newWorkspaceWithSCIM(t *testing.T, me, parents, scim func(testserver.Request) any) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, me)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, parents)
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", scim)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

// Only the not-enabled answers fall back to SCIM /Me. A not-enabled identity/me is the whole
// trigger, so the user comes from SCIM as it did before identity/me.
func TestGetCurrentUserFallsBackToSCIMWhenIdentityMeNotEnabled(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		errorCode string
	}{
		{name: "not found", status: http.StatusNotFound, errorCode: "ENDPOINT_NOT_FOUND"},
		{name: "not implemented", status: http.StatusNotImplemented, errorCode: "NOT_IMPLEMENTED"},
		{name: "feature disabled", status: http.StatusBadRequest, errorCode: "FEATURE_DISABLED"},
		{name: "not found without code", status: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var scimCalls atomic.Int32
			w := newWorkspaceWithSCIM(t,
				func(testserver.Request) any { return notEnabledResponse(tc.status, tc.errorCode) },
				func(testserver.Request) any { return notEnabledResponse(tc.status, tc.errorCode) },
				func(testserver.Request) any {
					scimCalls.Add(1)
					return scimUserBody
				})

			user, err := GetCurrentUser(t.Context(), w)
			require.NoError(t, err)
			assert.Equal(t, "scim-user@databricks.com", user.UserName)
			assert.Equal(t, int32(1), scimCalls.Load())
		})
	}
}

// Any other identity/me error fails the read as before and never falls back, so a
// permission problem or a server fault is not hidden behind SCIM.
func TestGetCurrentUserDoesNotFallBackOnOtherErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		errorCode string
		once      bool
	}{
		{name: "unauthenticated", status: http.StatusUnauthorized, errorCode: "UNAUTHENTICATED"},
		{name: "forbidden", status: http.StatusForbidden, errorCode: "PERMISSION_DENIED"},
		{name: "bad request", status: http.StatusBadRequest, errorCode: "INVALID_PARAMETER_VALUE"},
		{name: "server error", status: http.StatusInternalServerError, errorCode: "INTERNAL_ERROR", once: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var scimCalls atomic.Int32
			w := newWorkspaceWithSCIM(t,
				func(testserver.Request) any { return notEnabledResponse(tc.status, tc.errorCode) },
				func(testserver.Request) any { return notEnabledResponse(tc.status, tc.errorCode) },
				func(testserver.Request) any {
					scimCalls.Add(1)
					return scimUserBody
				})

			var err error
			if tc.once {
				_, err = GetCurrentUserOnce(t.Context(), w)
			} else {
				_, err = GetCurrentUser(t.Context(), w)
			}
			require.Error(t, err)
			assert.Equal(t, int32(0), scimCalls.Load())
		})
	}
}

// The assumed-group check does not run on the SCIM fallback, which cannot report a group.
// Without identity/me the caller is the SCIM user, as it was before identity/me.
func TestGetUserPrincipalFallbackHasNoAssumedGroupCheck(t *testing.T) {
	w := newWorkspaceWithSCIM(t,
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any { return scimUserBody })

	user, err := GetUserPrincipal(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "scim-user@databricks.com", user.UserName)
}

// Every request that reads the current user names the API that served it in the user agent,
// so request logs show the path taken. The SCIM call the CLI makes by design carries no tag.
func TestCurrentUserRequestsAreTaggedWithTheirAPI(t *testing.T) {
	var identityUA, scimUA string
	w := newWorkspaceWithSCIM(t,
		func(req testserver.Request) any {
			identityUA = req.Headers.Get("User-Agent")
			return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user", "account_user_status": "ACTIVE"}}
		},
		func(testserver.Request) any { return map[string]any{"me_transitive_parent_groups": []map[string]any{}} },
		func(req testserver.Request) any {
			scimUA = req.Headers.Get("User-Agent")
			return scimUserBody
		})
	_, err := GetCurrentUser(t.Context(), w)
	require.NoError(t, err)
	assert.Contains(t, identityUA, "current-user-api/identity-me")

	// The same workspace without identity/me: the fallback request carries the fallback tag.
	fallback := newWorkspaceWithSCIM(t,
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(req testserver.Request) any {
			scimUA = req.Headers.Get("User-Agent")
			return scimUserBody
		})
	_, err = GetCurrentUser(t.Context(), fallback)
	require.NoError(t, err)
	assert.Contains(t, scimUA, "current-user-api/scim-me-fallback")

	// A direct SCIM call, as the current-user command makes, is not tagged.
	_, err = w.CurrentUser.Me(t.Context(), iam.MeRequest{})
	require.NoError(t, err)
	assert.NotContains(t, scimUA, "current-user-api")
}

func TestListCurrentUserParentGroupsRejectsRepeatedPageToken(t *testing.T) {
	var calls atomic.Int32
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(req testserver.Request) any {
		calls.Add(1)
		// A server bug that keeps returning the same next page token must not loop forever.
		return map[string]any{
			"me_transitive_parent_groups": []map[string]any{{"group_id": "11", "group_name": "eng"}},
			"next_page_token":             "page2",
		}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	_, err = listCurrentUserParentGroups(t.Context(), w)
	assert.ErrorContains(t, err, "repeated page token")
	assert.Equal(t, int32(2), calls.Load())
}

// awaitRequest reports whether the request signalled on ch arrived within a few seconds.
// The concurrency tests wait on each other's arrival, so a serial implementation fails here.
func awaitRequest(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(10 * time.Second):
		return false
	}
}

// getWithin runs get and fails the test if it has not returned in time, so a read left
// running shows up as a failure instead of a hung test.
func getWithin[T any](t *testing.T, get func() (T, error)) (T, error) {
	t.Helper()
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := get()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, r.err
	case <-time.After(10 * time.Second):
		t.Fatal("the call did not return; a read it started was not cancelled")
		var zero T
		return zero, nil
	}
}

// blockUntilCancelled holds a request open until the client cancels it, as a slow
// groups read would, so a test can see whether the caller cancels it.
func blockUntilCancelled(req testserver.Request) {
	select {
	case <-req.Context.Done():
	case <-time.After(20 * time.Second):
	}
}

// Both reads are in flight at once: each handler waits for the other's request to arrive,
// so a serial implementation times out here. The order of arrival is not asserted.
func TestGetUserPrincipalAndGroupsRunsBothReadsConcurrently(t *testing.T) {
	meSeen := make(chan struct{})
	groupsSeen := make(chan struct{})
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, func(testserver.Request) any {
		close(meSeen)
		if !awaitRequest(groupsSeen) {
			return testserver.Response{StatusCode: http.StatusGatewayTimeout}
		}
		return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user", "account_user_status": "ACTIVE"}}
	})
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		close(groupsSeen)
		if !awaitRequest(meSeen) {
			return testserver.Response{StatusCode: http.StatusGatewayTimeout}
		}
		return map[string]any{"me_transitive_parent_groups": []map[string]any{
			{"group_id": "22", "group_name": "users"},
			{"group_id": "11", "group_name": "eng"},
		}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	res, err := getWithin(t, func() (*CurrentUserAndGroups, error) {
		return GetUserPrincipalAndGroups(t.Context(), w)
	})
	require.NoError(t, err)
	require.NoError(t, res.GroupsErr)
	assert.Equal(t, "test-user", res.User.UserName)
	assert.Equal(t, []iam.ComplexValue{
		{Value: "11", Display: "eng"},
		{Value: "22", Display: "users"},
	}, res.Groups)
}

// identity/me not enabled wins over the groups read, which is cancelled and waited for rather
// than left running. The user and groups then come from one SCIM /Me call.
func TestGetUserPrincipalAndGroupsFallbackCancelsGroupsRead(t *testing.T) {
	groupsSeen := make(chan struct{})
	var scimCalls atomic.Int32
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, func(testserver.Request) any {
		if !awaitRequest(groupsSeen) {
			return testserver.Response{StatusCode: http.StatusGatewayTimeout}
		}
		return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND")
	})
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(req testserver.Request) any {
		close(groupsSeen)
		blockUntilCancelled(req)
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "groups denied"}}
	})
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(testserver.Request) any {
		scimCalls.Add(1)
		return scimUserBody
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	res, err := getWithin(t, func() (*CurrentUserAndGroups, error) {
		return GetUserPrincipalAndGroups(t.Context(), w)
	})
	require.NoError(t, err)
	assert.Equal(t, "scim-user@databricks.com", res.User.UserName)
	assert.True(t, res.UserFromSCIM)
	assert.True(t, res.GroupsFromSCIM)
	assert.Equal(t, int32(1), scimCalls.Load())
}

// The bundle path reads the user and the groups from one SCIM /Me call when identity/me is
// not enabled, and passes the groups on as SCIM returns them.
func TestGetUserPrincipalAndGroupsFallsBackToSCIMForUserAndGroups(t *testing.T) {
	var scimCalls atomic.Int32
	w := newWorkspaceWithSCIM(t,
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any {
			scimCalls.Add(1)
			return scimUserBody
		})

	res, err := GetUserPrincipalAndGroups(t.Context(), w)
	require.NoError(t, err)
	require.NoError(t, res.GroupsErr)
	assert.Equal(t, "scim-user@databricks.com", res.User.UserName)
	assert.Equal(t, []string{"2 zeta", "1 alpha"}, groupPairs(res.Groups))
	assert.True(t, res.UserFromSCIM)
	assert.True(t, res.GroupsFromSCIM)
	assert.Equal(t, int32(1), scimCalls.Load())
}

// Only the parent-groups endpoint not enabled: the user stays from identity/me and the groups
// come from SCIM /Me.
func TestGetUserPrincipalAndGroupsFallsBackToSCIMForGroupsOnly(t *testing.T) {
	w := newWorkspaceWithSCIM(t,
		func(testserver.Request) any {
			return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user", "account_user_status": "ACTIVE"}}
		},
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any { return scimUserBody })

	res, err := GetUserPrincipalAndGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", res.User.UserName)
	assert.Equal(t, []string{"2 zeta", "1 alpha"}, groupPairs(res.Groups))
	assert.False(t, res.UserFromSCIM)
	assert.True(t, res.GroupsFromSCIM)
}

// A failed SCIM read of the groups is a groups failure, not an error of the whole read.
func TestGetUserPrincipalAndGroupsSCIMGroupsFailureIsGroupsError(t *testing.T) {
	w := newWorkspaceWithSCIM(t,
		func(testserver.Request) any {
			return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user", "account_user_status": "ACTIVE"}}
		},
		func(testserver.Request) any { return notEnabledResponse(http.StatusNotFound, "ENDPOINT_NOT_FOUND") },
		func(testserver.Request) any {
			return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "scim denied"}}
		})

	res, err := GetUserPrincipalAndGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", res.User.UserName)
	assert.Nil(t, res.Groups)
	assert.False(t, res.GroupsFromSCIM)
	require.ErrorContains(t, res.GroupsErr, "scim denied")
}

// A groups failure is reported beside the user, never as the error, so the caller can fall
// back to the principal.
func TestGetUserPrincipalAndGroupsReportsGroupsErrorSeparately(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, func(testserver.Request) any {
		return map[string]any{"user": map[string]any{"user_id": "1", "username": "test-user"}}
	})
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "groups denied"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	res, err := GetUserPrincipalAndGroups(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "test-user", res.User.UserName)
	assert.Nil(t, res.Groups)
	require.ErrorContains(t, res.GroupsErr, "groups denied")
}

// An assumed group is rejected even when the groups read fails, and the groups read is
// cancelled rather than left running.
func TestGetUserPrincipalAndGroupsRejectsAssumedGroupAndCancelsGroups(t *testing.T) {
	groupsSeen := make(chan struct{})
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("GET", identityMePath, func(testserver.Request) any {
		if !awaitRequest(groupsSeen) {
			return testserver.Response{StatusCode: http.StatusGatewayTimeout}
		}
		return map[string]any{"group": map[string]any{"group_id": "3000033333", "group_name": "data-engineers"}}
	})
	server.Handle("GET", identityMeTransitiveParentGroupsPath, func(req testserver.Request) any {
		close(groupsSeen)
		blockUntilCancelled(req)
		return testserver.Response{StatusCode: http.StatusForbidden, Body: map[string]any{"error_code": "PERMISSION_DENIED", "message": "groups denied"}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	res, err := getWithin(t, func() (*CurrentUserAndGroups, error) {
		return GetUserPrincipalAndGroups(t.Context(), w)
	})
	assert.Nil(t, res)
	require.ErrorIs(t, err, ErrGroupPrincipal)
	assert.NotContains(t, err.Error(), "groups denied")
}
