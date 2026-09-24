//go:build !windows

package ssh

import (
	"errors"
	"strings"
	"unicode"
)

func proxyCommandArgumentsForTest(command string) ([]string, error) {
	var args []string
	var arg strings.Builder
	inArg := false
	inSingleQuote := false
	escaped := false
	flush := func() {
		if inArg {
			args = append(args, arg.String())
			arg.Reset()
			inArg = false
		}
	}
	for _, r := range command {
		switch {
		case escaped:
			arg.WriteRune(r)
			inArg = true
			escaped = false
		case inSingleQuote && r == '\'':
			inSingleQuote = false
		case inSingleQuote:
			arg.WriteRune(r)
		case r == '\\':
			escaped = true
			inArg = true
		case r == '\'':
			inSingleQuote = true
			inArg = true
		case unicode.IsSpace(r):
			flush()
		default:
			arg.WriteRune(r)
			inArg = true
		}
	}
	if inSingleQuote || escaped {
		return nil, errors.New("unterminated ProxyCommand argument")
	}
	flush()
	return args, nil
}
