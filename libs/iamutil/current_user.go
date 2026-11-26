package iamutil

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
	"github.com/databricks/databricks-sdk-go/useragent"
	"golang.org/x/sync/errgroup"
)

const (
	// meRetryTimeout bounds how long GetCurrentUser retries transient 500s.
	meRetryTimeout = 30 * time.Second

	// groupsReadTimeout bounds one groups read. The groups are advisory, so a slow read is
	// reported as unknown groups rather than stalling the command. Groups reads are not retried.
	groupsReadTimeout = 10 * time.Second

	identityMePath                       = "/api/2.0/identity/me"
	identityMeTransitiveParentGroupsPath = "/api/2.0/identity/me/transitive-parent-groups"

	scimMePath = "/api/2.0/preview/scim/v2/Me"

	// currentUserAPIKey is the user-agent key that names the API a current-user request went to,
	// so request logs show whether identity/me or the SCIM /Me fallback served it. The SCIM
	// calls that stay on SCIM by design (current-user me, the workspace-ID lookup) carry no tag.
	currentUserAPIKey          = "current-user-api"
	currentUserAPIIdentityMe   = "identity-me"
	currentUserAPISCIMFallback = "scim-me-fallback"
)

// meNotEnabledCodes are the error codes identity/me and its parent-groups endpoint return in a
// workspace that does not serve them yet. The 404 and 501 statuses are matched through
// apierr.ErrNotFound and ErrNotImplemented.
var meNotEnabledCodes = []string{"ENDPOINT_NOT_FOUND", "FEATURE_DISABLED", "NOT_IMPLEMENTED"}

// ErrGroupPrincipal is returned by GetUserPrincipal and GetUserPrincipalOnce when the current request is
// authorized by an assumed group, which has no user identity to deploy as. It is never returned on the
// SCIM /Me fallback, which cannot report an assumed group.
var ErrGroupPrincipal = errors.New("the current identity is an assumed group, but this command needs a user or service principal")

// identityMe is the principal returned by GET /api/2.0/identity/me. Exactly one
// of the three fields is set. Groups and parent groups are not returned inline.
type identityMe struct {
	User             *iamv2.User             `json:"user,omitempty"`
	Group            *iamv2.Group            `json:"group,omitempty"`
	ServicePrincipal *iamv2.ServicePrincipal `json:"service_principal,omitempty"`
}

// currentPrincipal is the authorizing principal as an iam.User, with group set for an assumed group.
type currentPrincipal struct {
	user  *iam.User
	group bool
}

// GetCurrentUser returns the current user, retrying on HTTP 500 responses. It reads the
// authorizing principal of the current request from GET /api/2.0/identity/me, which also
// resolves an assumed group, and falls back to SCIM /Me where identity/me is not enabled.
// A user from identity/me has no Groups, Roles or Entitlements; GetUserPrincipalAndGroups
// reads the groups. A user from the SCIM fallback is the SCIM response as SCIM returns it, so
// its fields depend on the API that answered.
//
// identity/me is read-only, so retrying a 500 is safe. The SDK retries 503 but not 500, and
// this call runs at the start of every bundle command, so an unretried 500 would fail it.
func GetCurrentUser(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := readPrincipal(ctx, w, true, "")
	if err != nil {
		return nil, err
	}
	return p.user, nil
}

// GetCurrentUserOnce is GetCurrentUser with a single attempt, for probes that should
// fail fast rather than wait out transient 500s.
func GetCurrentUserOnce(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := readPrincipal(ctx, w, false, "")
	if err != nil {
		return nil, err
	}
	return p.user, nil
}

// GetUserPrincipal is GetCurrentUser for commands that act as the current user, such
// as bundle deployments and paths under /Workspace/Users. It returns ErrGroupPrincipal
// when the caller is an assumed group. The SCIM fallback cannot report an assumed group,
// so it is never rejected there.
func GetUserPrincipal(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return userPrincipal(ctx, w, true, "")
}

// GetUserPrincipalOnce is GetUserPrincipal with a single attempt, for probes that should
// fail fast rather than wait out transient 500s.
func GetUserPrincipalOnce(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return userPrincipal(ctx, w, false, "")
}

// GetUserPrincipalUserName is GetUserPrincipal for callers that need only the user name. On
// the SCIM fallback it asks SCIM for userName alone, so the response omits the groups and
// entitlements.
func GetUserPrincipalUserName(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return userPrincipal(ctx, w, true, "userName")
}

func userPrincipal(ctx context.Context, w *databricks.WorkspaceClient, retry bool, attributes string) (*iam.User, error) {
	p, err := readPrincipal(ctx, w, retry, attributes)
	if err != nil {
		return nil, err
	}
	if p.group {
		return nil, ErrGroupPrincipal
	}
	return p.user, nil
}

// readPrincipal reads the authorizing principal from identity/me, retrying 500s when retry is set.
// When identity/me is not enabled in the workspace it reads the user from SCIM /Me instead. Any
// other error fails the read.
func readPrincipal(ctx context.Context, w *databricks.WorkspaceClient, retry bool, attributes string) (*currentPrincipal, error) {
	p, err := readIdentityMe(ctx, w, retry)
	if err == nil {
		return p, nil
	}
	if !meNotEnabled(err) {
		return nil, err
	}
	log.Debugf(ctx, "identity/me is not enabled in this workspace; reading the current user from SCIM /Me")
	u, err := scimMe(ctx, w, retry, attributes)
	if err != nil {
		return nil, err
	}
	return &currentPrincipal{user: u}, nil
}

// readIdentityMe reads the principal from identity/me, retrying 500s when retry is set.
func readIdentityMe(ctx context.Context, w *databricks.WorkspaceClient, retry bool) (*currentPrincipal, error) {
	c, err := newIdentityClient(w)
	if err != nil {
		return nil, err
	}
	return withRetry(ctx, retry, func() (*currentPrincipal, error) {
		return getIdentityMe(ctx, c)
	})
}

// scimMe reads the current user from the SCIM /Me API, the fallback for workspaces where
// identity/me is not enabled. attributes, when set, limits the response to those SCIM
// attributes. The request carries the workspace routing header, as on identity/me, and is
// tagged so it is told apart from the SCIM calls that stay on SCIM by design.
func scimMe(ctx context.Context, w *databricks.WorkspaceClient, retry bool, attributes string) (*iam.User, error) {
	c, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	var query map[string]any
	if attributes != "" {
		query = map[string]any{"attributes": attributes}
	}
	headers := auth.WorkspaceIDHeaders(w.Config)
	ctx = useragent.InContext(ctx, currentUserAPIKey, currentUserAPISCIMFallback) //nolint:fatcontext
	return withRetry(ctx, retry, func() (*iam.User, error) {
		var u iam.User
		if err := c.Do(ctx, http.MethodGet, scimMePath, headers, query, nil, &u); err != nil {
			return nil, err
		}
		return &u, nil
	})
}

// withRetry runs get once, or with the 500 retry of retryOn500 when retry is set.
func withRetry[T any](ctx context.Context, retry bool, get func() (T, error)) (T, error) {
	if !retry {
		return get()
	}
	return retryOn500(ctx, get)
}

// retryOn500 runs get, retrying HTTP 500 responses for meRetryTimeout. Other errors fail at once.
func retryOn500[T any](ctx context.Context, get func() (T, error)) (T, error) {
	res, err := retries.Poll(ctx, meRetryTimeout, func() (*T, *retries.Err) {
		v, err := get()
		if err == nil {
			return &v, nil
		}
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusInternalServerError {
			return nil, retries.Continue(err)
		}
		return nil, retries.Halt(err)
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return *res, nil
}

// meNotEnabled reports whether err means identity/me, or its parent-groups endpoint, is not
// enabled in the workspace. Only these errors fall back to SCIM /Me; a 401, 403 or 5xx does not.
func meNotEnabled(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false
	}
	return errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrNotImplemented) ||
		slices.Contains(meNotEnabledCodes, apiErr.ErrorCode)
}

// identityClient sends GETs to the identity endpoints. Build one per command and reuse it
// across retries and pages.
type identityClient struct {
	client  *databricksclient.DatabricksClient
	headers map[string]string
}

func newIdentityClient(w *databricks.WorkspaceClient) (*identityClient, error) {
	c, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &identityClient{client: c, headers: auth.WorkspaceIDHeaders(w.Config)}, nil
}

// get sends a GET to path and decodes the response into out.
func (c *identityClient) get(ctx context.Context, path string, query map[string]any, out any) error {
	ctx = useragent.InContext(ctx, currentUserAPIKey, currentUserAPIIdentityMe)
	return c.client.Do(ctx, http.MethodGet, path, c.headers, query, nil, out)
}

func getIdentityMe(ctx context.Context, c *identityClient) (*currentPrincipal, error) {
	var me identityMe
	if err := c.get(ctx, identityMePath, nil, &me); err != nil {
		return nil, err
	}
	user, err := meToUser(&me)
	if err != nil {
		return nil, err
	}
	return &currentPrincipal{user: user, group: me.Group != nil}, nil
}

// meToUser maps a principal from identity/me onto an iam.User, setting Id, UserName,
// Name, DisplayName, Active, ExternalId and Emails. identity/me returns the given and
// family names separately, so Name is taken as is (nil when both are empty) and
// DisplayName joins them. A service principal's UserName is its application ID, not its
// name.
func meToUser(me *identityMe) (*iam.User, error) {
	switch {
	case me.User != nil:
		u := me.User
		var name *iam.Name
		if u.FullName.GivenName != "" || u.FullName.FamilyName != "" {
			name = &iam.Name{GivenName: u.FullName.GivenName, FamilyName: u.FullName.FamilyName}
		}
		return &iam.User{
			Id:          u.UserId,
			UserName:    u.Username,
			Name:        name,
			DisplayName: strings.TrimSpace(u.FullName.GivenName + " " + u.FullName.FamilyName),
			Active:      u.AccountUserStatus == iamv2.StateActive,
			ExternalId:  u.ExternalId,
			Emails:      []iam.ComplexValue{{Value: u.Username, Primary: true}},
		}, nil
	case me.ServicePrincipal != nil:
		sp := me.ServicePrincipal
		return &iam.User{
			Id:          sp.ServicePrincipalId,
			UserName:    sp.ApplicationId,
			DisplayName: sp.DisplayName,
			Active:      sp.AccountSpStatus == iamv2.StateActive,
			ExternalId:  sp.ExternalId,
		}, nil
	case me.Group != nil:
		g := me.Group
		return &iam.User{
			Id:          g.GroupId,
			UserName:    g.GroupName,
			DisplayName: g.GroupName,
		}, nil
	default:
		return nil, errors.New("identity/me returned no principal")
	}
}

// parentGroup is one entry of ListMeTransitiveParentGroupsResponse. The keys follow the
// IAM v2 proto, not the generic SDK response, which decodes a different key.
type parentGroup struct {
	GroupId   string `json:"group_id"`
	GroupName string `json:"group_name"`
}

// listCurrentUserParentGroups returns the groups the current principal belongs to,
// directly or through parent groups, following every page. The response also names
// workspace-local groups such as admins and users, which the account group list lacks.
func listCurrentUserParentGroups(ctx context.Context, w *databricks.WorkspaceClient) ([]parentGroup, error) {
	c, err := newIdentityClient(w)
	if err != nil {
		return nil, err
	}
	var groups []parentGroup
	pageToken := ""
	seenTokens := map[string]bool{}
	for {
		query := map[string]any{}
		if pageToken != "" {
			query["page_token"] = pageToken
		}
		var page struct {
			Groups        []parentGroup `json:"me_transitive_parent_groups"`
			NextPageToken string        `json:"next_page_token,omitempty"`
		}
		if err := c.get(ctx, identityMeTransitiveParentGroupsPath, query, &page); err != nil {
			return nil, err
		}
		groups = append(groups, page.Groups...)
		if page.NextPageToken == "" {
			return groups, nil
		}
		// Following a token the server already sent would loop until the context ends.
		if seenTokens[page.NextPageToken] {
			return nil, errors.New("identity/me returned a repeated page token for the parent groups")
		}
		seenTokens[page.NextPageToken] = true
		pageToken = page.NextPageToken
	}
}

// groupValues converts parent groups to the CurrentUserGroups shape.
func groupValues(parents []parentGroup) []iam.ComplexValue {
	var groups []iam.ComplexValue
	for _, g := range parents {
		groups = append(groups, iam.ComplexValue{Value: g.GroupId, Display: g.GroupName})
	}
	slices.SortFunc(groups, func(a, b iam.ComplexValue) int {
		return strings.Compare(a.Display, b.Display)
	})
	return groups
}

// CurrentUserAndGroups is the result of GetUserPrincipalAndGroups. It has no Groups, Roles or
// Entitlements on identity/me; on the SCIM fallback User is the SCIM response as SCIM returns it.
type CurrentUserAndGroups struct {
	User   *iam.User
	Groups []iam.ComplexValue
	// GroupsErr is set when only the groups read failed. User is still set and Groups is nil.
	GroupsErr error
	// UserFromSCIM is set when identity/me is not enabled and User is the SCIM /Me user.
	UserFromSCIM bool
	// GroupsFromSCIM is set when Groups come from the SCIM /Me response, because identity/me
	// or its parent-groups endpoint is not enabled.
	GroupsFromSCIM bool
}

// GetUserPrincipalAndGroups is GetUserPrincipal and the groups read run concurrently, so a
// command pays one round trip for both reads instead of two. The user read retries 500s; the
// groups read is a single attempt bounded by groupsReadTimeout, since the groups are advisory.
//
// Only identity/me decides the result. Its error, including ErrGroupPrincipal, is returned,
// and the groups read is cancelled and waited for before returning, so no request outlives
// the call. A groups failure is reported in GroupsErr and never as the returned error.
//
// When identity/me is not enabled, the user and the groups come from one SCIM /Me call. When
// only the parent-groups endpoint is not enabled, the groups come from SCIM /Me. Neither
// fallback can detect an assumed group, so the result is the SCIM user, as before identity/me.
func GetUserPrincipalAndGroups(ctx context.Context, w *databricks.WorkspaceClient) (*CurrentUserAndGroups, error) {
	g, gctx := errgroup.WithContext(ctx)

	var principal *currentPrincipal
	g.Go(func() error {
		p, err := readIdentityMe(gctx, w, true)
		if err != nil {
			return err
		}
		if p.group {
			return ErrGroupPrincipal
		}
		principal = p
		return nil
	})

	var parents []parentGroup
	var groupsErr error
	g.Go(func() error {
		// Returns nil on purpose: a groups failure must neither cancel nor replace the identity/me result.
		gc, cancel := context.WithTimeout(gctx, groupsReadTimeout)
		defer cancel()
		parents, groupsErr = listCurrentUserParentGroups(gc, w)
		return nil
	})

	if err := g.Wait(); err != nil {
		if !meNotEnabled(err) {
			return nil, err
		}
		log.Debugf(ctx, "identity/me is not enabled in this workspace; reading the current user from SCIM /Me")
		u, err := scimMe(ctx, w, true, "")
		if err != nil {
			return nil, err
		}
		return &CurrentUserAndGroups{User: u, Groups: u.Groups, UserFromSCIM: true, GroupsFromSCIM: true}, nil
	}

	res := &CurrentUserAndGroups{User: principal.user}
	switch {
	case groupsErr == nil:
		res.Groups = groupValues(parents)
	case meNotEnabled(groupsErr):
		log.Debugf(ctx, "the identity/me parent groups are not enabled in this workspace; reading the groups from SCIM /Me")
		gc, cancel := context.WithTimeout(ctx, groupsReadTimeout)
		defer cancel()
		u, err := scimMe(gc, w, false, "")
		if err != nil {
			res.GroupsErr = err
		} else {
			res.Groups = u.Groups
			res.GroupsFromSCIM = true
		}
	default:
		res.GroupsErr = groupsErr
	}
	return res, nil
}
