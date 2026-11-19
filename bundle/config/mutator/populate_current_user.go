package mutator

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
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
	// parent-groups request. A failed groups read is not cached: it returns an error from
	// the compute function, and the principal is kept for the warning below.
	var principal *iam.User
	var groupsErr error
	cached, err := cache.GetOrCompute(ctx, b.Cache, key, func(ctx context.Context) (cachedCurrentUser, error) {
		me, err := iamutil.GetUserPrincipal(ctx, w)
		if err != nil {
			return cachedCurrentUser{}, err
		}
		principal = me
		groups, err := iamutil.CurrentUserGroups(ctx, w)
		if err != nil {
			groupsErr = err
			return cachedCurrentUser{}, err
		}
		return cachedCurrentUser{User: me, Groups: groups}, nil
	})

	var diags diag.Diagnostics
	user := cached.User
	switch {
	case err == nil:
		user.Groups = cached.Groups
	case groupsErr != nil && errors.Is(err, groupsErr):
		user = principal
		diags = diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Could not read the groups of the current user: %v", groupsErr),
		}}
		b.CurrentUserGroupsUnknown = true
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

	return diags
}
