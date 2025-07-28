package mutator

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"unsafe"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type populateCurrentUser struct {
	lastKnownAuthorizationHeader string
	cache                        bundle.Cache
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

	fmt.Printf("New cache dir initialized: %s\n", cacheDir)

	return nil
}

// getCachedAuthorizationHeader retrieves the cached authorization header for a given host
func (m *populateCurrentUser) getCachedAuthorizationHeader(ctx context.Context, host string) (string, bool) {
	if m.cache == nil {
		return "", false
	}

	fingerprint, err := bundle.GenerateFingerprint("auth_header", host)
	if err != nil {
		return "", false
	}

	data, found := m.cache.Read(ctx, fingerprint)
	if !found {
		return "", false
	}

	return string(data), true
}

func (m *populateCurrentUser) Name() string {
	return "PopulateCurrentUser"
}

func (m *populateCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.CurrentUser != nil {
		return nil
	}

	// Initialize cache for authorization headers
	if err := m.initializeCache(ctx, b); err != nil {
		return diag.FromErr(err)
	}

	w := b.WorkspaceClient()
	d := getDatabricksClient(w)

	fmt.Printf("populateCurrentUser - getting user auth")

	me, err := m.getCurrentUserWithAuthTracking(ctx, d)
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

// getCurrentUserWithAuthTracking makes the CurrentUser.Me method, caches the authorization header and returns result
func (m *populateCurrentUser) getCurrentUserWithAuthTracking(ctx context.Context, client *client.DatabricksClient) (*iam.User, error) {
	var user iam.User
	path := "/api/2.0/preview/scim/v2/Me"

	headers := make(map[string]string)
	headers["Accept"] = "application/json"

	// Visitor to inspect request headers
	headerInspector := func(req *http.Request) error {
		for name, values := range req.Header {
			if name != "Authorization" {
				continue
			}
			for _, value := range values {
				m.lastKnownAuthorizationHeader = value
				// Store authorization header in cache
				fmt.Printf("got lastKnownAuthorizationHeader: %s\n", value)
				if m.cache != nil {
					fingerprint, err := bundle.GenerateFingerprint("auth_header", req.URL.Host)
					if err == nil {
						fmt.Printf("storing cached auth header: %s\n", fingerprint)
						_ = m.cache.Store(ctx, fingerprint, []byte(value))
					}
				}
			}
		}
		return nil
	}

	err := client.Do(ctx, http.MethodGet, path, headers, nil, nil, &user, headerInspector)
	return &user, err
}

// TODO: find a way to get the client without using reflection
func getDatabricksClient(w *databricks.WorkspaceClient) *client.DatabricksClient {
	v := reflect.ValueOf(w.CurrentUser)
	// value is a pointer. Keep dereferencing it until we get to the actual value
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			panic("nil pointer encountered")
		}
		v = v.Elem()
	}

	clientField := v.FieldByName("client")
	clientInterface := getUnexportedField(clientField)
	client, ok := clientInterface.(*client.DatabricksClient)
	if !ok {
		panic("client is not a client.DatabricksClient")
	}
	return client
}

func getUnexportedField(field reflect.Value) any {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}
