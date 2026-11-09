package template

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"uuid"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type ErrFail struct {
	msg string
}

func (err ErrFail) Error() string {
	return err.msg
}

type pair struct {
	k string
	v any
}

var (
	cachedUser               *iam.User
	cachedIsServicePrincipal *bool
	cachedCatalog            *string
)

var metastoreDisabledErrorCodes = []string{
	"PERMISSION_DENIED",
	"METASTORE_DOES_NOT_EXIST", // Default metastore is not assigned to the workspace.
	"FEATURE_DISABLED",         // Unity Catalog is not available for feature tier STANDARD_TIER.
}

// UUID that is stable for the duration of the template execution. This can be used
// to populate the `bundle.uuid` field in databricks.yml by template authors.
//
// It's automatically logged in our telemetry logs when `databricks bundle init`
// is run and can be used to attribute DBU revenue to bundle templates.
var bundleUuid = uuid.New().String()

// deploymentHistoryEnvVar makes the built-in templates set
// experimental.deployment_history in the databricks.yml they generate. It is scoped to
// `bundle init` because it persists the setting in the generated project rather than applying
// it to the bundle being run.
//
// Temporary: remove it and its callers in the templates once this is the default.
const deploymentHistoryEnvVar = "DATABRICKS_BUNDLE_INIT_DEPLOYMENT_HISTORY"

// workspaceHostProfileEnvVar names a profile whose host the workspace_host helper returns instead
// of the selected profile's host. The profile must have the selected profile's workspace ID, so
// that a template never gets its host from another workspace. Everything else, including
// .workspace, keeps the selected profile.
//
// DABs in the Workspace selects the profile for the host of the page, which can be a host shared
// by several workspaces, and sets this to a profile for the workspace's own host, so that
// templates that use workspace_host keep generating that host.
const workspaceHostProfileEnvVar = "DATABRICKS_BUNDLE_INIT_WORKSPACE_HOST_PROFILE"

// workspaceHost returns the host for the workspace_host helper, see workspaceHostProfileEnvVar.
func workspaceHost(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	name := env.Get(ctx, workspaceHostProfileEnvVar)
	if name == "" {
		if w.Config.Host == "" {
			return "", errors.New("cannot determine target workspace, please first setup a configuration profile using 'databricks auth login'")
		}
		return w.Config.Host, nil
	}

	profiles, err := profile.GetProfiler(ctx).LoadProfiles(ctx, profile.WithName(name))
	if err != nil {
		return "", fmt.Errorf("failed to load profile %q from %s: %w", name, workspaceHostProfileEnvVar, err)
	}
	if len(profiles) == 0 || profiles[0].Host == "" {
		return "", fmt.Errorf("profile %q from %s not found or has no host", name, workspaceHostProfileEnvVar)
	}
	p := profiles[0]
	if p.WorkspaceID == "" || p.WorkspaceID == profile.WorkspaceIDNone || p.WorkspaceID != w.Config.WorkspaceID {
		return "", fmt.Errorf("profile %q from %s must have the workspace ID of the selected profile, got %q and %q", name, workspaceHostProfileEnvVar, p.WorkspaceID, w.Config.WorkspaceID)
	}
	// Normalize the host the same way the SDK normalizes the selected profile's host.
	return (&sdkconfig.Config{Host: p.Host}).CanonicalHostName(), nil
}

func loadHelpers(ctx context.Context) template.FuncMap {
	w := cmdctx.WorkspaceClient(ctx)
	return template.FuncMap{
		"fail": func(format string, args ...any) (any, error) {
			return nil, ErrFail{fmt.Sprintf(format, args...)}
		},
		// Alias for https://pkg.go.dev/net/url#Parse. Allows usage of all methods of url.URL
		"url": func(rawUrl string) (*url.URL, error) {
			return url.Parse(rawUrl)
		},
		// Alias for https://pkg.go.dev/regexp#Compile. Allows usage of all methods of regexp.Regexp
		"regexp": func(expr string) (*regexp.Regexp, error) {
			return regexp.Compile(expr)
		},
		// Alias for https://pkg.go.dev/math/rand/v2#IntN. Returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n).
		"random_int": func(n int) int {
			return rand.IntN(n)
		},
		// Alias for https://pkg.go.dev/uuid#New. Returns, as a string, a UUID which is a 128 bit (16 byte) Universal Unique IDentifier as defined in RFC 9562.
		"uuid": func() string {
			return uuid.New().String()
		},
		"bundle_uuid": func() string {
			return bundleUuid
		},
		// A key value pair. This is used with the map function to generate maps
		// to use inside a template
		"pair": func(k string, v any) pair {
			return pair{k, v}
		},
		// map converts a list of pairs to a map object. This is useful to pass multiple
		// objects to templates defined in the library directory. Go text template
		// syntax for invoking a template only allows specifying a single argument,
		// this function can be used to workaround that limitation.
		//
		// For example: {{template "my_template" (map (pair "foo" $arg1) (pair "bar" $arg2))}}
		// $arg1 and $arg2 can be referred from inside "my_template" as ".foo" and ".bar"
		"map": func(pairs ...pair) map[string]any {
			result := make(map[string]any, 0)
			for _, p := range pairs {
				result[p.k] = p.v
			}
			return result
		},
		// Get smallest node type (follows Terraform's GetSmallestNodeType)
		"smallest_node_type": func() (string, error) {
			if w.Config.Host == "" {
				return "", errors.New("cannot determine target workspace, please first setup a configuration profile using 'databricks auth login'")
			}
			if w.Config.IsAzure() {
				return "Standard_D3_v2", nil
			} else if w.Config.IsGcp() {
				return "n1-standard-4", nil
			}
			return "i3.xlarge", nil
		},
		"path_separator": func() string {
			return string(os.PathSeparator)
		},
		"workspace_host": func() (string, error) {
			return workspaceHost(ctx, w)
		},
		"user_name": func() (string, error) {
			if cachedUser == nil {
				var err error
				cachedUser, err = w.CurrentUser.Me(ctx, iam.MeRequest{})
				if err != nil {
					return "", err
				}
			}
			result := cachedUser.UserName
			if result == "" {
				result = cachedUser.Id
			}
			return result, nil
		},
		"short_name": func() (string, error) {
			if cachedUser == nil {
				var err error
				cachedUser, err = w.CurrentUser.Me(ctx, iam.MeRequest{})
				if err != nil {
					return "", err
				}
			}
			return iamutil.GetShortUserName(cachedUser), nil
		},
		// Get the default workspace catalog. If there is no default, or if
		// Unity Catalog is not enabled, return an empty string.
		"default_catalog": func() (string, error) {
			if cachedCatalog == nil {
				metastore, err := w.Metastores.Current(ctx)
				if err != nil {
					if aerr, ok := errors.AsType[*apierr.APIError](err); ok && (slices.Contains(metastoreDisabledErrorCodes, aerr.ErrorCode) || aerr.Message == "Bad Target: /api/2.1/unity-catalog/current-metastore-assignment") {
						// Ignore: access denied or workspace doesn't have a metastore assigned
						empty_default := ""
						cachedCatalog = &empty_default
						return "", nil
					}
					return "", err
				}
				cachedCatalog = &metastore.DefaultCatalogName
			}
			return *cachedCatalog, nil
		},
		"is_service_principal": func() (bool, error) {
			if cachedIsServicePrincipal != nil {
				return *cachedIsServicePrincipal, nil
			}
			if cachedUser == nil {
				var err error
				cachedUser, err = w.CurrentUser.Me(ctx, iam.MeRequest{})
				if err != nil {
					return false, err
				}
			}
			result := iamutil.IsServicePrincipal(cachedUser)
			cachedIsServicePrincipal = &result
			return result, nil
		},
		"deployment_history": func() bool {
			v, _ := env.GetBool(ctx, deploymentHistoryEnvVar)
			return v
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	}
}
