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

var HelperFuncs = template.FuncMap{
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
