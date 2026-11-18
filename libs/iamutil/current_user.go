package iamutil

import (
	"context"
	"errors"
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

// ErrGroupPrincipal is returned by GetUserPrincipal when the current request is
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

// GetCurrentUser returns the authorizing principal of the current request as an
// iam.User, read from GET /api/2.0/identity/me, which also resolves an assumed group.
// Groups, Roles and Entitlements are not set; CurrentUserGroups reads the groups.
//
// The call retries HTTP 500 responses for up to meRetryTimeout. It runs at the
// start of most CLI commands, and an unretried 500 would fail the whole command.
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
	p, err := getIdentityMe(ctx, w)
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

// pollIdentityMe is getIdentityMe that retries HTTP 500 responses for meRetryTimeout.
func pollIdentityMe(ctx context.Context, w *databricks.WorkspaceClient) (*currentPrincipal, error) {
	return retries.Poll(ctx, meRetryTimeout, func() (*currentPrincipal, *retries.Err) {
		p, err := getIdentityMe(ctx, w)
		if err == nil {
			return p, nil
		}
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusInternalServerError {
			return nil, retries.Continue(err)
		}
		return nil, retries.Halt(err)
	})
}

// identityGet sends a GET to path with the workspace's auth headers and decodes the
// response into out.
func identityGet(ctx context.Context, w *databricks.WorkspaceClient, path string, query map[string]any, out any) error {
	c, err := databricksclient.New(w.Config)
	if err != nil {
		return err
	}
	return c.Do(ctx, http.MethodGet, path, auth.WorkspaceIDHeaders(w.Config), query, nil, out)
}

func getIdentityMe(ctx context.Context, w *databricks.WorkspaceClient) (*currentPrincipal, error) {
	var me identityMe
	if err := identityGet(ctx, w, identityMePath, nil, &me); err != nil {
		return nil, err
	}
	user, err := meToUser(&me)
	if err != nil {
		return nil, err
	}
	return &currentPrincipal{user: user, group: me.Group != nil}, nil
}

// meToUser maps a principal from identity/me onto an iam.User, setting Id, UserName,
// DisplayName, Active, ExternalId and Emails. A service principal's UserName is its
// application ID, not its name.
func meToUser(me *identityMe) (*iam.User, error) {
	switch {
	case me.User != nil:
		u := me.User
		return &iam.User{
			Id:          u.UserId,
			UserName:    u.Username,
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
	var groups []parentGroup
	pageToken := ""
	for {
		query := map[string]any{}
		if pageToken != "" {
			query["page_token"] = pageToken
		}
		var page struct {
			Groups        []parentGroup `json:"me_transitive_parent_groups"`
			NextPageToken string        `json:"next_page_token,omitempty"`
		}
		if err := identityGet(ctx, w, identityMeTransitiveParentGroupsPath, query, &page); err != nil {
			return nil, err
		}
		groups = append(groups, page.Groups...)
		if page.NextPageToken == "" {
			return groups, nil
		}
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
