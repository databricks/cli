package template

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"text/template"

	"golang.org/x/exp/slices"
)

var errSkipThisFile = errors.New("skip generating this file")

var skipPatterns = make([]string, 0)

type ErrFail struct {
	msg string
}

func (err ErrFail) Error() string {
	return err.msg
}

var HelperFuncs = template.FuncMap{
	// Text template execution returns the error only if it's the second return
	// value from a function: https://pkg.go.dev/text/template#hdr-Pipelines
	"skipThisFile": func() (any, error) {
		return nil, errSkipThisFile
	},
	// TODO: write an explanation for this function
	"skip": func(pattern string) error {
		if !slices.Contains(skipPatterns, pattern) {
			skipPatterns = append(skipPatterns, pattern)
		}
		return nil
	},
	"urlParse": func(rawUrl string) (*url.URL, error) {
		return url.Parse(rawUrl)
	},
	"regexpCompile": func(expr string) (*regexp.Regexp, error) {
		return regexp.Compile(expr)
	},
	"fail": func(format string, args ...any) (any, error) {
		return nil, ErrFail{fmt.Sprintf(format, args...)}
	},
}
