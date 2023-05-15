package init

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
)

var ErrSkipThisFile = errors.New("skip generating this file")

var HelperFuncs = template.FuncMap{
	"skipThisFile": func() error {
		panic(ErrSkipThisFile)
	},
	"eqString": func(a string, b string) bool {
		return a == b
	},
	"eqNumber": func(a float64, b int) bool {
		return int(a) == b
	},
	"validationError": func(message string) error {
		panic(fmt.Errorf(message))
	},
	"assertStartsWith": func(s string, substr string) error {
		if !strings.HasPrefix(s, substr) {
			panic(fmt.Errorf("%s does not start with %s.", s, substr))
		}
		return nil
	},
}
