package template

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"text/template"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"

	"github.com/google/uuid"
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

var cachedUser *iam.User
var cachedIsServicePrincipal *bool
var cachedCatalog *string

func loadHelpers(ctx context.Context) template.FuncMap {
	w := root.WorkspaceClient(ctx)
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
		// Alias for https://pkg.go.dev/math/rand#Intn. Returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n).
		"random_int": func(n int) int {
			return rand.Intn(n)
		},
		// Alias for https://pkg.go.dev/github.com/google/uuid#New. Returns, as a string, a UUID which is a 128 bit (16 byte) Universal Unique IDentifier as defined in RFC 4122.
		"uuid": func() string {
			return uuid.New().String()
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
				return "", errors.New("cannot determine target workspace, please first setup a configuration profile using 'databricks configure'")
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
			if w.Config.Host == "" {
				return "", errors.New("cannot determine target workspace, please first setup a configuration profile using 'databricks configure'")
			}
			return w.Config.Host, nil
		},
		"user_name": func() (string, error) {
			if cachedUser == nil {
				var err error
				cachedUser, err = w.CurrentUser.Me(ctx)
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
				cachedUser, err = w.CurrentUser.Me(ctx)
				if err != nil {
					return "", err
				}
			}
			return auth.GetShortUserName(cachedUser), nil
		},
		// Get the default workspace catalog. If there is no default, or if
		// Unity Catalog is not enabled, return an empty string.
		"default_catalog": func() (string, error) {
			if cachedCatalog == nil {
				metastore, err := w.Metastores.Current(ctx)
				if err != nil {
					var aerr *apierr.APIError
					if errors.As(err, &aerr) && aerr.ErrorCode == "METASTORE_DOES_NOT_EXIST" {
						// Workspace doesn't have a metastore assigned, ignore error
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
				cachedUser, err = w.CurrentUser.Me(ctx)
				if err != nil {
					return false, err
				}
			}
			result := auth.IsServicePrincipal(cachedUser.UserName)
			cachedIsServicePrincipal = &result
			return result, nil
		},
	}
}
