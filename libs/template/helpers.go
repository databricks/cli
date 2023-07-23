package template

import (
	"fmt"
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
	// Returns the input string as is. Useful for printing text that would otherwise
	// get interpreted as a template.
	"raw": func(s string) string {
		return s
	},
}
