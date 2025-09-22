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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type populateCurrentUserCached struct {
	cache cache.Cache[*iam.User]
}

// populateCurrentUserCached sets the `current_user` property on the workspace.
func PopulateCurrentUserCached() bundle.Mutator {
	return &populateCurrentUserCached{}
}

// initializeCache sets up the cache for authorization headers if not already initialized
func (m *populateCurrentUserCached) initializeCache(ctx context.Context, b *bundle.Bundle) error {
	if m.cache != nil {
		return nil
	}

	cacheDir, err := b.BundleLevelCacheDir(ctx, "auth")
	if err != nil {
		return err
	}

	m.cache, err = cache.NewFileCache[*iam.User](cacheDir)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] Failed to initialize cache dir: %s\n", cacheDir)
	} else {
		log.Debugf(ctx, "[Local Cache] New cache dir initialized: %s\n", cacheDir)
	}

	return nil
}

func (m *populateCurrentUserCached) Name() string {
	return "populateCurrentUserCached"
}

func (m *populateCurrentUserCached) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}

	err := m.initializeCache(ctx, b)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to initialize cache: %v \n", err)
	}
	w := b.WorkspaceClient()

	bearerToken := m.getBearerToken(ctx, w)

	me, err := m.cache.GetOrCompute(ctx, bearerToken, func(ctx context.Context) (*iam.User, error) {
		currentUser, err := w.CurrentUser.Me(ctx)
		return currentUser, err
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName: iamutil.GetShortUserName(me),
		User:      me,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}

// getBearerToken extracts the bearer token from the workspace client's token source
func (m *populateCurrentUserCached) getBearerToken(ctx context.Context, w *databricks.WorkspaceClient) string {
	bearerToken := ""
	tokenSource := w.Config.GetTokenSource()
	if tokenSource == nil {
		log.Debugf(ctx, "[Local Cache] token source not found\n")
	} else {
		token, err := tokenSource.Token(context.Background())
		if err != nil {
			log.Debugf(ctx, "[Local Cache] error reading token source: %v \n", err)
		} else {
			bearerToken = token.AccessToken
		}
	}
	log.Debugf(ctx, "[Local Cache] found bearer token with length: %d\n", len(bearerToken))
	return bearerToken
}
