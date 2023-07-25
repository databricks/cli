package template

import (
	"fmt"
	"net/url"
	"text/template"
)

type ErrFail struct {
	msg string
}

func (err ErrFail) Error() string {
	return err.msg
}

var helperFuncs = template.FuncMap{
	"fail": func(format string, args ...any) (any, error) {
		return nil, ErrFail{fmt.Sprintf(format, args...)}
	},
	// Alias for https://pkg.go.dev/net/url#Parse. Allows usage of all methods of url.URL
	"url": func(rawUrl string) (*url.URL, error) {
		return url.Parse(rawUrl)
	},
}
