package template

import (
	"fmt"
	"regexp"
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
	// Alias for https://pkg.go.dev/regexp#Compile. Allows usage of all methods of regexp.Regexp
	"regexp": func(expr string) (*regexp.Regexp, error) {
		return regexp.Compile(expr)
	},
}
