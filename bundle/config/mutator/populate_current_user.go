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
	me, err := cache.GetOrCompute(ctx, b.Cache, fingerprint, func(ctx context.Context) (*iam.User, error) {
		return iamutil.GetUserPrincipal(ctx, w)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Groups are not part of the cached principal: membership can change while the
	// fingerprint stays the same, so they are read on every command.
	var diags diag.Diagnostics
	groups, err := iamutil.CurrentUserGroups(ctx, w)
	if err != nil {
		diags = diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Could not read the groups of the current user: %v", err),
		}}
	}
	// Copy the cached principal so the groups are not written back into the cache entry.
	user := *me
	user.Groups = groups

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName:          iamutil.GetShortUserName(&user),
		DomainFriendlyName: iamutil.GetShortUserDomainFriendlyName(&user),
		User:               &user,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return diags
}
