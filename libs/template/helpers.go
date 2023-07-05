package template

import (
	"errors"
	"net/url"
	"text/template"
)

var errSkipThisFile = errors.New("skip generating this file")

var HelperFuncs = template.FuncMap{
	// Text template execution returns the error only if it's the second return
	// value from a function: https://pkg.go.dev/text/template#hdr-Pipelines
	"skipThisFile": func() (any, error) {
		return nil, errSkipThisFile
	},
	"urlParse": func(rawUrl string) (*url.URL, error) {
		return url.Parse(rawUrl)
	},
}
