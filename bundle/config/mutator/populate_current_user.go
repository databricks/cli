package mutator

import (
	"context"

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
	w := b.WorkspaceClient()

	var me *iam.User
	var err error

	fingerprint := b.GetUserFingerprint(ctx)
	me, err = cache.GetOrCompute(b.Cache, ctx, fingerprint, func(ctx context.Context) (*iam.User, error) {
		currentUser, err := w.CurrentUser.Me(ctx)
		return currentUser, err
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName:          iamutil.GetShortUserName(me),
		DomainFriendlyName: iamutil.GetShortUserDomainFriendlyName(me),
		User:               me,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}
