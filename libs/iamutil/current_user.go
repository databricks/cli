package iamutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
)

const (
	// meRetryTimeout bounds how long GetCurrentUser retries transient 500s.
	meRetryTimeout = 30 * time.Second

	identityMePath                       = "/api/2.0/identity/me"
	identityMeTransitiveParentGroupsPath = "/api/2.0/identity/me/transitive-parent-groups"
)

// ErrMeNotEnabled is wrapped by the identity/me errors of a workspace that does not serve the
// API yet. Callers check it with errors.Is to show the message as the diagnostic.
var ErrMeNotEnabled = errors.New("the identity/me API is not enabled in this workspace, so the current user cannot be read; use an older Databricks CLI version for now")

// meNotEnabledCodes are the error codes identity/me returns in a workspace that does not serve
// the API yet. The 404 and 501 statuses are matched through apierr.ErrNotFound and ErrNotImplemented.
var meNotEnabledCodes = []string{"ENDPOINT_NOT_FOUND", "FEATURE_DISABLED", "NOT_IMPLEMENTED"}

// ErrGroupPrincipal is returned by GetUserPrincipal and GetUserPrincipalOnce when the current request is
// authorized by an assumed group, which has no user identity to deploy as.
var ErrGroupPrincipal = errors.New("the current identity is an assumed group, but this command needs a user or service principal")

// identityMe is the principal returned by GET /api/2.0/identity/me. Exactly one
// of the three fields is set. Groups and parent groups are not returned inline.
type identityMe struct {
	User             *iamv2.User             `json:"user,omitempty"`
	Group            *iamv2.Group            `json:"group,omitempty"`
	ServicePrincipal *iamv2.ServicePrincipal `json:"service_principal,omitempty"`
}

// currentPrincipal is identity/me as an iam.User, with group set for an assumed group.
type currentPrincipal struct {
	user  *iam.User
	group bool
}

// GetCurrentUser returns the current user, retrying on HTTP 500 responses. It reads the
// authorizing principal of the current request from GET /api/2.0/identity/me, which also
// resolves an assumed group. Groups, Roles and Entitlements are not set; CurrentUserGroups
// reads the groups.
//
// identity/me is read-only, so retrying a 500 is safe. The SDK retries 503 but not 500, and
// this call runs at the start of every bundle command, so an unretried 500 would fail it.
func GetCurrentUser(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := pollIdentityMe(ctx, w)
	if err != nil {
		return nil, err
	}
	return p.user, nil
}

// GetCurrentUserOnce is GetCurrentUser with a single attempt, for probes that should
// fail fast rather than wait out transient 500s.
func GetCurrentUserOnce(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := getIdentityMeOnce(ctx, w)
	if err != nil {
		return nil, err
	}
	return p.user, nil
}

// GetUserPrincipal is GetCurrentUser for commands that act as the current user, such
// as bundle deployments and paths under /Workspace/Users. It returns ErrGroupPrincipal
// when the caller is an assumed group.
func GetUserPrincipal(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := pollIdentityMe(ctx, w)
	if err != nil {
		return nil, err
	}
	if p.group {
		return nil, ErrGroupPrincipal
	}
	return p.user, nil
}

// GetUserPrincipalOnce is GetUserPrincipal with a single attempt, for probes that should
// fail fast rather than wait out transient 500s.
func GetUserPrincipalOnce(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	p, err := getIdentityMeOnce(ctx, w)
	if err != nil {
		return nil, err
	}
	if p.group {
		return nil, ErrGroupPrincipal
	}
	return p.user, nil
}

// pollIdentityMe is getIdentityMe that retries HTTP 500 responses for meRetryTimeout.
func pollIdentityMe(ctx context.Context, w *databricks.WorkspaceClient) (*currentPrincipal, error) {
	c, err := newIdentityClient(w)
	if err != nil {
		return nil, err
	}
	return retryOn500(ctx, func() (*currentPrincipal, error) {
		return getIdentityMe(ctx, c)
	})
}

// getIdentityMeOnce is getIdentityMe with a single attempt.
func getIdentityMeOnce(ctx context.Context, w *databricks.WorkspaceClient) (*currentPrincipal, error) {
	c, err := newIdentityClient(w)
	if err != nil {
		return nil, err
	}
	return getIdentityMe(ctx, c)
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

// meNotEnabled reports whether err means the workspace does not serve identity/me yet.
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
	return c.client.Do(ctx, http.MethodGet, path, c.headers, query, nil, out)
}

func getIdentityMe(ctx context.Context, c *identityClient) (*currentPrincipal, error) {
	var me identityMe
	if err := c.get(ctx, identityMePath, nil, &me); err != nil {
		if meNotEnabled(err) {
			return nil, fmt.Errorf("%w: %w", ErrMeNotEnabled, err)
		}
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
		_, err = retryOn500(ctx, func() (struct{}, error) {
			return struct{}{}, c.get(ctx, identityMeTransitiveParentGroupsPath, query, &page)
		})
		if err != nil {
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

// CurrentUserGroups returns the groups the current principal belongs to, directly or
// through parent groups, as iam.ComplexValue entries with the group ID as Value and the
// group name as Display, sorted by Display. It is one paged call and never reads the
// account group list.
func CurrentUserGroups(ctx context.Context, w *databricks.WorkspaceClient) ([]iam.ComplexValue, error) {
	parents, err := listCurrentUserParentGroups(ctx, w)
	if err != nil {
		return nil, err
	}
	var groups []iam.ComplexValue
	for _, g := range parents {
		groups = append(groups, iam.ComplexValue{Value: g.GroupId, Display: g.GroupName})
	}
	slices.SortFunc(groups, func(a, b iam.ComplexValue) int {
		return strings.Compare(a.Display, b.Display)
	})
	return groups, nil
}
