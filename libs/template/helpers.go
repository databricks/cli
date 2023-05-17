package template

import (
	"errors"
	"text/template"
)

var errSkipThisFile = errors.New("skip generating this file")

var HelperFuncs = template.FuncMap{
	"skipThisFile": func() error {
		panic(errSkipThisFile)
	},
}
