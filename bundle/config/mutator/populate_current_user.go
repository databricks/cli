package mutator

import (
	"context"
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

// cachedCurrentUser is the principal and its groups under one cache entry. Entries
// written without groups decode with a nil User and are read as a miss.
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
	fingerprint := b.GetUserFingerprint(ctx)

	// A cache hit makes no identity/me or parent-groups request. Groups are cached with
	// the principal, so a stale membership lasts as long as the principal entry does.
	var user *iam.User
	var diags diag.Diagnostics
	if cached, ok := cache.Get[cachedCurrentUser](ctx, b.Cache, fingerprint); ok && cached.User != nil {
		user = cached.User
		user.Groups = cached.Groups
	} else {
		me, err := iamutil.GetUserPrincipal(ctx, w)
		if err != nil {
			return diag.FromErr(err)
		}
		user = me
		groups, err := iamutil.CurrentUserGroups(ctx, w)
		if err != nil {
			// Not cached, so the next command reads the groups again.
			diags = diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Could not read the groups of the current user: %v", err),
			}}
			b.CurrentUserGroupsUnknown = true
		} else {
			user.Groups = groups
			cache.Put(ctx, b.Cache, fingerprint, cachedCurrentUser{User: user, Groups: groups})
		}
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
