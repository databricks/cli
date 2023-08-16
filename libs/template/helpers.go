package template

import (
	"fmt"
	"net/url"
	"regexp"
	"text/template"
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

var helperFuncs = template.FuncMap{
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
}
