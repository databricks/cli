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
)

const (
	// meRetryTimeout bounds how long GetCurrentUser retries transient 500s.
	meRetryTimeout = 30 * time.Second

	identityMePath                       = "/api/2.0/identity/me"
	identityMeTransitiveParentGroupsPath = "/api/2.0/identity/me/transitive-parent-groups"
)

// identityMe is the principal returned by GET /api/2.0/identity/me. Exactly one
// of the three fields is set. Groups and parent groups are not returned inline.
type identityMe struct {
	User             *iamv2.User             `json:"user,omitempty"`
	Group            *iamv2.Group            `json:"group,omitempty"`
	ServicePrincipal *iamv2.ServicePrincipal `json:"service_principal,omitempty"`
}

// GetCurrentUser returns the authorizing principal of the current request as an
// iam.User, read from GET /api/2.0/identity/me, which also resolves an assumed group.
// Groups, Roles and Entitlements are not set; CurrentUserGroups reads the groups.
//
// The call retries HTTP 500 responses for up to meRetryTimeout. It runs at the
// start of most CLI commands, and an unretried 500 would fail the whole command.
func GetCurrentUser(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return retries.Poll(ctx, meRetryTimeout, func() (*iam.User, *retries.Err) {
		user, err := getIdentityMe(ctx, w)
		if err == nil {
			return user, nil
		}
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusInternalServerError {
			return nil, retries.Continue(err)
		}
		return nil, retries.Halt(err)
	})
}

// GetCurrentUserOnce is GetCurrentUser with a single attempt, for probes that should
// fail fast rather than wait out transient 500s.
func GetCurrentUserOnce(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return getIdentityMe(ctx, w)
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

func getIdentityMe(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	var me identityMe
	if err := identityGet(ctx, w, identityMePath, nil, &me); err != nil {
		return nil, err
	}
	return meToUser(&me)
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

// listCurrentUserParentGroupIDs returns the IDs of the groups the current principal
// belongs to, directly or through parent groups, following every page. The keys follow
// ListMeTransitiveParentGroupsResponse in the IAM v2 proto, not the generic SDK response,
// which decodes a different key.
func listCurrentUserParentGroupIDs(ctx context.Context, w *databricks.WorkspaceClient) ([]string, error) {
	var ids []string
	pageToken := ""
	for {
		query := map[string]any{}
		if pageToken != "" {
			query["page_token"] = pageToken
		}
		var page struct {
			Groups []struct {
				GroupId string `json:"group_id"`
			} `json:"me_transitive_parent_groups"`
			NextPageToken string `json:"next_page_token,omitempty"`
		}
		if err := identityGet(ctx, w, identityMeTransitiveParentGroupsPath, query, &page); err != nil {
			return nil, err
		}
		for _, g := range page.Groups {
			ids = append(ids, g.GroupId)
		}
		if page.NextPageToken == "" {
			return ids, nil
		}
		pageToken = page.NextPageToken
	}
}

// CurrentUserGroups returns the groups the current principal belongs to, directly or
// through parent groups, as iam.ComplexValue entries with the group ID as Value and the
// group name as Display, sorted by Display. Names come from one list of all account
// groups, so the cost is one list call per 1000 groups rather than one lookup per group.
// It fails when either call fails, so a caller never sees a partial list as complete.
func CurrentUserGroups(ctx context.Context, w *databricks.WorkspaceClient) ([]iam.ComplexValue, error) {
	ids, err := listCurrentUserParentGroupIDs(ctx, w)
	if err != nil {
		return nil, err
	}
	all, err := w.WorkspaceIamV2.ListGroupsProxyAll(ctx, iamv2.ListGroupsProxyRequest{})
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(all))
	for _, g := range all {
		names[g.GroupId] = g.GroupName
	}
	var groups []iam.ComplexValue
	for _, id := range ids {
		name, ok := names[id]
		if !ok {
			log.Debugf(ctx, "Group %s of the current user is not in the account group list; skipping it", id)
			continue
		}
		groups = append(groups, iam.ComplexValue{Value: id, Display: name})
	}
	slices.SortFunc(groups, func(a, b iam.ComplexValue) int {
		return strings.Compare(a.Display, b.Display)
	})
	return groups, nil
}
