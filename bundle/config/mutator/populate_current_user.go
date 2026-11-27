package mutator

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type populateCurrentUser struct{}

// currentUserCacheKey keys the cached principal and groups. Version keeps entries
// written before the current shape, which records the API that served them, from being read.
type currentUserCacheKey struct {
	Fingerprint bundle.UserFingerprint
	Version     int
}

const currentUserCacheVersion = 3

// cachedCurrentUser is the principal with its groups, cached as one entry. The source flags
// are cached too, so a cache hit records the same metrics as the read that filled the entry.
type cachedCurrentUser struct {
	User           *iam.User          `json:"user"`
	Groups         []iam.ComplexValue `json:"groups"`
	UserFromSCIM   bool               `json:"user_from_scim"`
	GroupsFromSCIM bool               `json:"groups_from_scim"`
}

// PopulateCurrentUser sets the `current_user` property on the workspace.
func PopulateCurrentUser() bundle.Mutator {
	return &populateCurrentUser{}
}

func (m *populateCurrentUser) Name() string {
	return "PopulateCurrentUser"
}

func (m *populateCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}
	w := b.WorkspaceClient(ctx)
	key := currentUserCacheKey{Fingerprint: b.GetUserFingerprint(ctx), Version: currentUserCacheVersion}

	// The groups are part of the cached value, so a cache hit makes no identity/me or
	// parent-groups request. A failed groups read is not cached, so the next command reads
	// them again; the read is kept for the fallback below.
	var read *iamutil.CurrentUserAndGroups
	cached, err := cache.GetOrCompute(ctx, b.Cache, key, func(ctx context.Context) (cachedCurrentUser, error) {
		r, err := iamutil.GetUserPrincipalAndGroups(ctx, w)
		if err != nil {
			return cachedCurrentUser{}, err
		}
		read = r
		if r.GroupsErr != nil {
			return cachedCurrentUser{}, r.GroupsErr
		}
		return cachedCurrentUser{User: r.User, Groups: r.Groups, UserFromSCIM: r.UserFromSCIM, GroupsFromSCIM: r.GroupsFromSCIM}, nil
	})

	var user *iam.User
	var userFromSCIM, groupsFromSCIM bool
	switch {
	case err == nil:
		user = cached.User
		user.Groups = cached.Groups
		userFromSCIM, groupsFromSCIM = cached.UserFromSCIM, cached.GroupsFromSCIM
	case read != nil && read.GroupsErr != nil && errors.Is(err, read.GroupsErr):
		// Debug, not a warning: the user cannot act on it, and the permission recommendation is
		// withheld through CurrentUserGroupsUnknown.
		log.Debugf(ctx, "Could not read the groups of the current user: %v", read.GroupsErr)
		user = read.User
		userFromSCIM = read.UserFromSCIM
		b.CurrentUserGroupsUnknown = true
	default:
		return diag.FromErr(err)
	}

	b.Telemetry.CurrentUserFromSCIMFallback = new(userFromSCIM)
	b.Telemetry.CurrentUserGroupsFromSCIMFallback = new(groupsFromSCIM)

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName:          iamutil.GetShortUserName(user),
		DomainFriendlyName: iamutil.GetShortUserDomainFriendlyName(user),
		User:               user,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}
