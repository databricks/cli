package mutator

import (
	"context"

	"github.com/databricks/cli/libs/cache"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type populateCurrentUser struct {
	cache cache.Cache[*iam.User]
}

// PopulateCurrentUser sets the `current_user` property on the workspace.
func PopulateCurrentUser() bundle.Mutator {
	return &populateCurrentUser{}
}

// initializeCache sets up the cache for authorization headers if not already initialized.
// By default, cache operates in measurement-only mode to gather metrics about potential savings.
// Set DATABRICKS_CACHE_ENABLED=true to enable actual caching.
func (m *populateCurrentUser) initializeCache(ctx context.Context, b *bundle.Bundle) {
	m.cache = cache.NewCache[*iam.User](ctx, "auth", 30, &b.Metrics)
}

func (m *populateCurrentUser) Name() string {
	return "PopulateCurrentUser"
}

func (m *populateCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}
	m.initializeCache(ctx, b)
	w := b.WorkspaceClient()

	var me *iam.User
	var err error

	fingerprint := b.GetUserFingerprint(ctx)
	if !fingerprint.IsEmpty() {
		log.Debugf(ctx, "[Local Cache] local cache is enabled")
		me, err = m.cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (*iam.User, error) {
			currentUser, err := w.CurrentUser.Me(ctx)
			return currentUser, err
		})
	} else {
		log.Debugf(ctx, "[Local Cache] local cache is disabled")
		me, err = w.CurrentUser.Me(ctx)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	if me == nil {
		return diag.Errorf("could not find current user, but no error was returned")
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
