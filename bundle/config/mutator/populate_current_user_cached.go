package mutator

import (
	"context"
	"net/http"
	"os"

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
func (m *populateCurrentUserCached) initializeCache(ctx context.Context) {
	if m.cache != nil {
		return
	}

	if os.Getenv("DATABRICKS_EXPERIMENTAL_CACHE_ENABLED") != "true" {
		log.Debugf(ctx, "[Local Cache] Local cache is disabled. Enable it be setting an env variable DATABRICKS_EXPERIMENTAL_CACHE_ENABLED=true\n")
		return
	}

	var err error
	m.cache, err = cache.NewFileCache[*iam.User]("auth")
	if err != nil {
		log.Debugf(ctx, "[Local Cache] Failed to initialize cache dir: %v\n", err)
	}
}

func (m *populateCurrentUserCached) Name() string {
	return "populateCurrentUserCached"
}

func (m *populateCurrentUserCached) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}
	m.initializeCache(ctx)
	w := b.WorkspaceClient()

	fingerprint := struct {
		authHeader string
	}{
		authHeader: m.getAuthorizationHeader(ctx, w),
	}

	var me *iam.User
	var err error

	if m.cache != nil && fingerprint.authHeader != "" {
		log.Debugf(ctx, "[Local Cache] local cache is enabled\n")
		me, err = m.cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (*iam.User, error) {
			currentUser, err := w.CurrentUser.Me(ctx)
			return currentUser, err
		})
	} else {
		log.Debugf(ctx, "[Local Cache] local cache is disabled\n")
		me, err = w.CurrentUser.Me(ctx)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	if me == nil {
		return diag.Errorf("could not find current user, but no error was returned")
	}

	b.Config.Workspace.CurrentUser = &config.User{
		ShortName: iamutil.GetShortUserName(me),
		User:      me,
	}

	// Configure tagging object now that we know we have a valid client.
	b.Tagging = tags.ForCloud(w.Config)

	return nil
}

func (m *populateCurrentUserCached) getAuthorizationHeader(ctx context.Context, w *databricks.WorkspaceClient) string {
	// Create a dummy request to extract the Authorization header
	req := &http.Request{Header: http.Header{}}
	if err := w.Config.Authenticate(req); err != nil {
		return ""
	}

	authHeader := req.Header.Get("Authorization")
	log.Debugf(ctx, "[Local Cache] found authorization header with length: %d\n", len(authHeader))
	return authHeader
}
