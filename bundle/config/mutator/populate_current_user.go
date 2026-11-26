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
// written before the groups were cached, which have another shape, from being read.
type currentUserCacheKey struct {
	Fingerprint bundle.UserFingerprint
	Version     int
}

const currentUserCacheVersion = 2

// cachedCurrentUser is the principal with its groups, cached as one entry.
type cachedCurrentUser struct {
	User   *iam.User          `json:"user"`
	Groups []iam.ComplexValue `json:"groups"`
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
	// them again; the principal is kept for the fallback below.
	var principal *iam.User
	var groupsErr error
	cached, err := cache.GetOrCompute(ctx, b.Cache, key, func(ctx context.Context) (cachedCurrentUser, error) {
		r, err := iamutil.GetUserPrincipalAndGroups(ctx, w)
		if err != nil {
			return cachedCurrentUser{}, err
		}
		principal = r.User
		if r.GroupsErr != nil {
			groupsErr = r.GroupsErr
			return cachedCurrentUser{}, r.GroupsErr
		}
		return cachedCurrentUser{User: r.User, Groups: r.Groups}, nil
	})

	user := cached.User
	switch {
	case err == nil:
		user.Groups = cached.Groups
	case groupsErr != nil && errors.Is(err, groupsErr):
		// Not a warning: `bundle validate --strict` would fail on it. The permission
		// recommendation is withheld through CurrentUserGroupsUnknown instead.
		log.Infof(ctx, "Could not read the groups of the current user: %v", groupsErr)
		user = principal
		b.CurrentUserGroupsUnknown = true
	case errors.Is(err, iamutil.ErrMeNotEnabled):
		// diag.FromErr would show only the API fields, which drop this message.
		return diag.Errorf("%v", err)
	default:
		return diag.FromErr(err)
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName:          iamutil.GetShortUserName(user),
		DomainFriendlyName: iamutil.GetShortUserDomainFriendlyName(user),
		User:               user,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}
