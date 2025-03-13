package template

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"text/template"

	//"github.com/databricks/databricks-sdk-go/apierr"
	//"github.com/databricks/databricks-sdk-go/service/iam"

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

var (
// cachedUser               *iam.User
// cachedIsServicePrincipal *bool
// cachedCatalog            *string
)

// UUID that is stable for the duration of the template execution. This can be used
// to populate the `bundle.uuid` field in databricks.yml by template authors.
//
// It's automatically logged in our telemetry logs when `databricks bundle init`
// is run and can be used to attribute DBU revenue to bundle templates.
var bundleUuid = uuid.New().String()

func loadHelpers(ctx context.Context, values map[string]any) template.FuncMap {
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
		"smallest_node_type": readValuesFunc(values, "smallest_node_type", "i3.xlarge"),
		"path_separator": func() string {
			return string(os.PathSeparator)
		},
		"workspace_host": readValuesFunc(values, "workspace_host", "<host>"),
		"user_name":      readValuesFunc(values, "user_name", "<user>"),
		"short_name":     readValuesFunc(values, "short_name", "<user>"),
		// Get the default workspace catalog. If there is no default, or if
		// Unity Catalog is not enabled, return an empty string.
		"default_catalog": readValuesFunc(values, "default_catalog", ""),


		"is_service_principal": func() (bool, error) {
			return false, nil
		}
	}
}

func readValuesFunc(values map[string]string, name, defaultValue string) (func() string) {
	return func() string {}
		x, ok := values[name]
		if ok {
			return x
		}
		return defaultValue
	}
}
