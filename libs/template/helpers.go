package template

import (
	"errors"
	"text/template"
)

var errSkipThisFile = errors.New("skip generating this file")

var HelperFuncs = template.FuncMap{
	// Text template execution returns the error only if it's the second return
	// value from a function: https://pkg.go.dev/text/template#hdr-Pipelines
	"skipThisFile": func() (any, error) {
		return nil, errSkipThisFile
	},
}
