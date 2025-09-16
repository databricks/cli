package mutator

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type populateCurrentUser struct {
	cache bundle.Cache
}

// PopulateCurrentUser sets the `current_user` property on the workspace.
func PopulateCurrentUser() bundle.Mutator {
	return &populateCurrentUser{}
}

// initializeCache sets up the cache for authorization headers if not already initialized
func (m *populateCurrentUser) initializeCache(ctx context.Context, b *bundle.Bundle) error {
	if m.cache != nil {
		return nil
	}

	cacheDir, err := b.BundleLevelCacheDir(ctx, "auth")
	if err != nil {
		return err
	}

	m.cache = bundle.NewFileCache(cacheDir)

	log.Debugf(ctx, "[Local Cache] New cache dir initialized: %s\n", cacheDir)

	return nil
}

func (m *populateCurrentUser) Name() string {
	return "PopulateCurrentUser"
}

func (m *populateCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}

	err := m.initializeCache(ctx, b)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to initialize cache: %v \n", err)
	}
	w := b.WorkspaceClient()

	bearerToken := m.getBearerToken(ctx, w)
	me := m.getUserFromCache(ctx, bearerToken)

	if me == nil {
		currentUser, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return diag.FromErr(err)
		}
		me = currentUser
		m.storeUserInCache(ctx, bearerToken, currentUser)
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

// getBearerToken extracts the bearer token from the workspace client's token source
func (m *populateCurrentUser) getBearerToken(ctx context.Context, w *databricks.WorkspaceClient) string {
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
	return bearerToken
}

// getUserFromCache attempts to retrieve user information from cache using the bearer token
func (m *populateCurrentUser) getUserFromCache(ctx context.Context, bearerToken string) *iam.User {
	if bearerToken == "" || m.cache == nil {
		return nil
	}

	log.Debugf(ctx, "[Local Cache] bearer token found: will use that for cache fingerprint\n")
	log.Debugf(ctx, "[Local Cache] bearer token: %s\n", bearerToken)

	fingerprint, err := bundle.GenerateFingerprint("auth_header", bearerToken)
	if err != nil {
		panic(err)
	}

	cachedUserBytes, isCacheHit := m.cache.Read(ctx, fingerprint)
	if isCacheHit {
		var me *iam.User
		if err := json.Unmarshal(cachedUserBytes, &me); err == nil {
			log.Debugf(ctx, "[Local Cache] user info read from cache: %s\n", fingerprint)
			return me
		}
	}

	return nil
}

// storeUserInCache stores user information in cache using the bearer token as key
func (m *populateCurrentUser) storeUserInCache(ctx context.Context, bearerToken string, user *iam.User) {
	if bearerToken == "" || m.cache == nil {
		return
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] could not serialize current user information: %v\n", err)
		return
	}

	fingerprint, err := bundle.GenerateFingerprint("auth_header", bearerToken)
	if err != nil {
		panic(err)
	}

	err = m.cache.Store(ctx, fingerprint, userBytes)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] could not store user information: %v\n", err)
	} else {
		log.Debugf(ctx, "[Local Cache] stored user information in cache: %s\n", fingerprint)
	}
}
